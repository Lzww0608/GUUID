package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-zookeeper/zk"
)

const (
	Epoch int64 = 1672531200000 // UTC: 2023-01-01 00:00:00

	WorkerIdBits = 10
	SequenceBits = 12

	WorkIdShift    = SequenceBits
	TimestampShift = SequenceBits + WorkerIdBits
	SequenceMask   = -1 ^ (-1 << SequenceBits)
	WorkerIdMask   = -1 ^ (-1 << WorkerIdBits)

	ZKRootPath = "/leaf_snowflake"
)

type SnowflakeDriver struct {
	mu       sync.Mutex
	lastTime int64
	workerID int64
	sequence int64

	zkClient *zk.Conn
	service  string
	port     int
}

type NodeInfo struct {
	LastTime   int64 `json:"last_time"`
	CreateTime int64 `json:"create_time"`
	WorkerID   int64 `json:"worker_id"`
}

func NewSnowflakeDriver(zkServers []string, serviceName string, port int) (*SnowflakeDriver, error) {
	driver := &SnowflakeDriver{
		service:  serviceName,
		port:     port,
		lastTime: 0,
		sequence: 0,
	}

	c, _, err := zk.Connect(zkServers, time.Second*5)
	if err != nil {
		return nil, fmt.Errorf("connect zk failed: %v", err)
	}
	driver.zkClient = c

	workerID, err := driver.registerOrRecover()
	if err != nil {
		return nil, err
	}

	driver.workerID = workerID
	log.Printf("snowflake driver initialized with workerID: %d", workerID)

	go driver.scheduledUploadTime()
	return driver, nil
}
