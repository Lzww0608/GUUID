package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	"github.com/go-zookeeper/zk"
)

// Constants for bit lengths and masks for Snowflake algorithm.
const (
	Epoch int64 = 1672531200000 // UTC: 2023-01-01 00:00:00

	WorkerIdBits = 10 // Number of bits for Worker ID (max 1024 nodes)
	SequenceBits = 12 // Number of bits for sequence num in same millisecond (max 4096 IDs/ms)

	WorkIdShift    = SequenceBits                // Shift for workerID field in final ID
	TimestampShift = SequenceBits + WorkerIdBits // Shift for timestamp field in final ID
	SequenceMask   = -1 ^ (-1 << SequenceBits)   // Mask to stay within sequence bits
	WorkerIdMask   = -1 ^ (-1 << WorkerIdBits)   // Mask to stay within workerID bits

	ZKRootPath = "/leaf_snowflake" // Root path in Zookeeper for node registration
)

// SnowflakeDriver maintains state for ID generation and Zookeeper communication.
type SnowflakeDriver struct {
	mu       sync.Mutex // Mutex for lock to ensure safe concurrent access
	lastTime int64      // Last timestamp an ID was generated
	workerID int64      // Worker ID for this instance
	sequence int64      // Sequence number for IDs in same millisecond

	zkClient *zk.Conn // Zookeeper client connection
	service  string   // Service name (affects ZK node path)
	port     int      // Port (used to derive node uniqueness)
}

// NodeInfo represents info stored for each worker in both ZK and cache file.
type NodeInfo struct {
	LastTime   int64 `json:"last_time"`   // Last timestamp this node was active
	CreateTime int64 `json:"create_time"` // Creation timestamp
	WorkerID   int64 `json:"worker_id"`   // Worker ID
}

// NewSnowflakeDriver initializes a SnowflakeDriver, registers with Zookeeper, and recovers/assigns a worker ID.
func NewSnowflakeDriver(zkServers []string, serviceName string, port int) (*SnowflakeDriver, error) {
	driver := &SnowflakeDriver{
		service:  serviceName,
		port:     port,
		lastTime: 0,
		sequence: 0,
	}

	c, _, err := zk.Connect(zkServers, time.Second*5) // Connect to Zookeeper
	if err != nil {
		return nil, fmt.Errorf("connect zk failed: %v", err)
	}
	driver.zkClient = c

	workerID, err := driver.registerOrRecover() // Register or recover workerID
	if err != nil {
		return nil, err
	}

	driver.workerID = workerID
	log.Printf("snowflake driver initialized with workerID: %d", workerID)

	// Periodically upload heartbeat and update state to Zookeeper and cache
	go driver.scheduledUploadTime()
	return driver, nil
}

// registerOrRecover registers this node to Zookeeper or recovers assignment from cache or ZK.
func (d *SnowflakeDriver) registerOrRecover() (int64, error) {
	// Build the ZK service path: e.g., /leaf_snowflake/serviceName
	servicePath := fmt.Sprintf("%s%s", ZKRootPath, d.service)
	d.ensurePath(servicePath) // Ensure the base path exists

	nodeKey := fmt.Sprintf("%s%d", servicePath, d.port) // Unique nodeKey per service+port

	var myNodeInfo NodeInfo
	var workerID int64

	exists, _, err := d.zkClient.Exists(nodeKey)
	if err != nil {
		return 0, fmt.Errorf("check node existence failed: %v", err)
	}

	if exists {
		// Attempt to recover workerID from ZK node
		data, _, err := d.zkClient.Get(nodeKey)
		if err != nil {
			return 0, fmt.Errorf("get node info failed: %v", err)
		}
		json.Unmarshal(data, &myNodeInfo)
		workerID = myNodeInfo.WorkerID

		currentTime := int64(time.Now().UnixNano() / int64(1e6))
		// Detect system clock rollback
		if currentTime < myNodeInfo.LastTime {
			return 0, fmt.Errorf("clock moved backwards: %d < %d", currentTime, myNodeInfo.LastTime)
		}

		log.Printf("recover workerID: %d from zk", workerID)
	} else {
		// Not registered in ZK, try local cache first
		cachedNode, err := d.loadLocalCache()
		if err == nil {
			workerID = cachedNode.WorkerID
			// Check for clock rollback against cached time
			if time.Now().UnixNano()/int64(1e6) < cachedNode.LastTime {
				return 0, fmt.Errorf("clock moved backwards: %d < %d", time.Now().UnixNano()/int64(1e6), cachedNode.LastTime)
			}
			log.Printf("recover workerID: %d from local cache", workerID)
		} else {
			// Assign workerID by hash/modulo if nothing found (simple assignment logic)
			workerID = int64(d.port % 1024)
		}

		now := time.Now().UnixNano() / int64(1e6)
		myNodeInfo = NodeInfo{
			WorkerID:   workerID,
			LastTime:   now,
			CreateTime: now,
		}
	}

	// Register or update node info in Zookeeper
	bytes, _ := json.Marshal(myNodeInfo)
	if exists {
		_, err = d.zkClient.Set(nodeKey, bytes, -1)
	} else {
		_, err = d.zkClient.Create(nodeKey, bytes, 0, zk.WorldACL(zk.PermAll))
	}
	if err != nil {
		return 0, fmt.Errorf("register or update node info failed: %v", err)
	}

	// Save to a local cache file for local recovery
	d.saveLocalCache(myNodeInfo)
	return workerID, nil
}

// NextID generates the next distributed unique ID using Snowflake algorithm.
func (d *SnowflakeDriver) NextID() (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now().UnixNano() / 1e6 // Current time in ms

	// Runtime clock rollback check
	if now < d.lastTime {
		offset := d.lastTime - now
		// If offset small (<=5ms), wait for time to catch up
		if offset <= 5 {
			time.Sleep(time.Duration(offset) * time.Millisecond)
			now = time.Now().UnixNano() / 1e6
			if now < d.lastTime {
				return 0, fmt.Errorf("clock moved backwards, refused to generate id")
			}
		} else {
			// If too large, refuse to generate IDs
			return 0, fmt.Errorf("clock moved backwards too much (%d ms)", offset)
		}
	}

	// If still within last generated millisecond, increment sequence number
	if now == d.lastTime {
		// Increment sequence and mask within SequenceBits (to avoid overflow)
		d.sequence = (d.sequence + 1) & SequenceMask
		// If sequence wraps to zero, we have exceeded per-ms capacity, wait for next ms
		if d.sequence == 0 {
			for now <= d.lastTime {
				now = time.Now().UnixNano() / 1e6
			}
		}
	} else {
		// It's a new millisecond: reset sequence to 0
		d.sequence = 0
	}

	d.lastTime = now

	// Compose the final 64-bit ID with bit shifts and bitwise ORs
	// | 1bit(0) | 41bit Timestamp | 10bit WorkerID | 12bit Sequence |
	id := ((now - Epoch) << TimestampShift) |
		(d.workerID << WorkIdShift) |
		d.sequence

	return id, nil
}

// scheduledUploadTime periodically updates this node's info in Zookeeper and the local cache.
func (d *SnowflakeDriver) scheduledUploadTime() {
	ticker := time.NewTicker(3 * time.Second)
	nodeKey := fmt.Sprintf("%s/%s/node-%d", ZKRootPath, d.service, d.port) // Key for this node in Zookeeper

	for range ticker.C {
		now := time.Now().UnixNano() / 1e6

		// If local time is less than lastTime, system clock went backwards! Alert here.
		if now < d.lastTime {
			log.Printf("Clock rollback detected during heartbeat! Local: %d, Last: %d", now, d.lastTime)
			// You may want to trigger alerting or terminate node here
			continue
		}

		info := NodeInfo{
			WorkerID: d.workerID,
			LastTime: now,
		}
		data, _ := json.Marshal(info)

		// Ignore errors, since Zookeeper may occasionally be unavailable
		d.zkClient.Set(nodeKey, data, -1)

		// Update local file cache as well
		d.saveLocalCache(info)
	}
}

// ensurePath recursively creates a ZK path if needed.
// Note: This is a simple check/create for demonstration; use recursive creation in production.
func (d *SnowflakeDriver) ensurePath(path string) {
	exists, _, _ := d.zkClient.Exists(path)
	if !exists {
		// Create the path with open permissions if it doesn't exist yet.
		d.zkClient.Create(path, []byte{}, 0, zk.WorldACL(zk.PermAll))
	}
}

// saveLocalCache saves the given NodeInfo to a file for local state recovery.
func (d *SnowflakeDriver) saveLocalCache(info NodeInfo) {
	data, _ := json.Marshal(info)
	fileName := fmt.Sprintf(".leaf_cache_%d", d.port)
	ioutil.WriteFile(fileName, data, 0644)
}

// loadLocalCache loads NodeInfo from the local cache file, if present.
func (d *SnowflakeDriver) loadLocalCache() (NodeInfo, error) {
	fileName := fmt.Sprintf(".leaf_cache_%d", d.port)
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return NodeInfo{}, err
	}
	var info NodeInfo
	json.Unmarshal(data, &info)
	return info, nil
}

// ==========================================
// 6. Main Test Entrypoint
// ==========================================

func main() {
	// NOTE: This code requires a local Zookeeper at localhost:2181 to run.
	// You can use Docker to start Zookeeper for local testing:
	// docker run --name some-zookeeper -p 2181:2181 -d zookeeper

	zkServers := []string{"127.0.0.1:2181"}

	// Start the ID service, simulating a worker on port 8080
	driver, err := NewSnowflakeDriver(zkServers, "order-service", 8080)
	if err != nil {
		log.Fatalf("Failed to init snowflake: %v", err)
	}

	log.Println("Start generating IDs...")

	var wg sync.WaitGroup
	// Launch 10 goroutines (threads) concurrently to generate IDs in parallel,
	// each generating 100 IDs
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				id, err := driver.NextID()
				if err != nil {
					log.Println(err)
				} else {
					fmt.Println(id)
				}
			}
		}()
	}
	wg.Wait()
	log.Println("Done.")

	// Prevent program from exiting to observe Zookeeper heartbeat updates.
	// (Remove/select{} in real production service/supervisor)
	//select {}
}
