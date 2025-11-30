package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Segment represents a range of IDs usable by this generator.
// Base: Start of the range (exclusive).
// Max: End of the range (inclusive).
// Step: The range size.
// Cursor: The current position in the range.
type Segment struct {
	Base   int64 // exclusive (the last granted ID)
	Max    int64 // inclusive (max usable ID)
	Step   int   // step size for segment
	Cursor int64 // current position, accessed atomically
}

// NewSegment creates a new ID segment, starting at base, ending at max, with a given step.
func NewSegment(base, max int64, step int) *Segment {
	return &Segment{
		Base:   base,
		Max:    max,
		Step:   step,
		Cursor: base,
	}
}

// Remaining returns how many IDs are left in the current segment.
func (s *Segment) Remaining() int64 {
	cur := atomic.LoadInt64(&s.Cursor)
	return s.Max - cur
}

// DoubleBuffer orchestrates two Segments - current (in use) and next (being prefetched).
// Implements double buffer prefetching strategy for IDs segment.
type DoubleBuffer struct {
	bizTag string

	current *Segment // currently served segment
	next    *Segment // prefetched next segment

	nextReady bool       // true if next segment ready to be used
	isLoading int32      // atomic flag for ongoing loading goroutine
	mu        sync.Mutex // protects buffer/switch logic

	dao *LeafDAO // database access object
}

// NewDoubleBuffer constructs a double buffer for given bizTag with DB DAO injected.
func NewDoubleBuffer(bizTag string, dao *LeafDAO) *DoubleBuffer {
	return &DoubleBuffer{
		bizTag: bizTag,
		dao:    dao,
	}
}

// Init loads the very first segment for this DoubleBuffer.
func (db *DoubleBuffer) Init() error {
	seg, err := db.dao.FetchNextSegment(db.bizTag)
	if err != nil {
		return err
	}
	db.current = seg
	return nil
}

// NextID atomically allocates and returns the next ID in the buffer, refilling or switching
// segments if needed. Ensures thread safety and minimal DB blocking.
func (db *DoubleBuffer) NextID() (int64, error) {
	if db.current == nil {
		return 0, errors.New("segment not initialized")
	}

	// Fast path: try to increment Cursor for current segment
	id := atomic.AddInt64(&db.current.Cursor, 1)

	// If still within the current segment range
	if id <= db.current.Max {
		db.CheckAndLoadNext() // try to prefetch asynchronously if running low
		return id, nil
	}

	// Slow path: segment may be exhausted. Need to lock and switch segment if possible.
	db.mu.Lock()
	defer db.mu.Unlock()

	// Double-check in case another goroutine already advanced the cursor while we waited for the lock
	if id := atomic.AddInt64(&db.current.Cursor, 1); id <= db.current.Max {
		return id, nil
	}

	// If the next buffer is ready, switch
	if db.nextReady && db.next != nil {
		// Switch to the next segment.
		db.current = db.next
		db.next = nil
		db.nextReady = false

		id := atomic.AddInt64(&db.current.Cursor, 1)
		return id, nil
	}

	// Neither buffer is ready. Synchronously fetch new segment from DB (fallback mode)
	seg, err := db.dao.FetchNextSegment(db.bizTag)
	if err != nil {
		return 0, err
	}

	db.current = seg
	db.next = nil
	db.nextReady = false
	id = atomic.AddInt64(&db.current.Cursor, 1)
	return id, nil
}

// CheckAndLoadNext triggers asynchronous prefetching of the next segment when the current one is running low.
// Only one goroutine can trigger load at a time (CAS protected).
func (db *DoubleBuffer) CheckAndLoadNext() {
	// If next buffer is already ready or loading is in progress, return early.
	if db.nextReady || atomic.LoadInt32(&db.isLoading) == 1 {
		return
	}

	// Calculate prefetch threshold: when only 20% of the segment is left, fire refetch.
	threshold := int64(float64(db.current.Step) * 0.2)
	if db.current.Remaining() > threshold {
		return
	}

	// Set isLoading=1 and start a goroutine to prefetch the next segment
	if atomic.CompareAndSwapInt32(&db.isLoading, 0, 1) {
		go func() {
			defer atomic.StoreInt32(&db.isLoading, 0) // always reset loading flag

			// Uncomment this to simulate prefetch delay
			// time.Sleep(50 * time.Millisecond)

			// Fetch next segment from DB
			seg, err := db.dao.FetchNextSegment(db.bizTag)
			if err != nil {
				// Logging can be added here on prefetch error
				return
			}

			// Lock before writing to .next
			db.mu.Lock()
			db.next = seg
			db.nextReady = true
			db.mu.Unlock()
			// Logging can be added here for successful prefetch
		}()
	}
}

// LeafDAO encapsulates all database operations, such as segment allocation.
type LeafDAO struct {
	db *sql.DB
}

// NewLeafDAO creates a new DAO with provided database DSN.
func NewLeafDAO(dsn string) (*LeafDAO, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// DB performance and safety tuning
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	return &LeafDAO{
		db: db,
	}, nil
}

// FetchNextSegment allocates a new segment from the database for the given bizTag, using a transaction.
// This SQL pattern guarantees atomic step/reservation for this caller.
func (dao *LeafDAO) FetchNextSegment(bizTag string) (*Segment, error) {
	tx, err := dao.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Step 1: Atomically reserve a range of IDs by updating max_id
	_, err = tx.ExecContext(context.Background(),
		"UPDATE leaf_alloc SET max_id = max_id + step WHERE biz_tag = ?", bizTag)
	if err != nil {
		return nil, err
	}

	// Step 2: Read back the new max_id, together with step
	var maxId int64
	var step int
	err = tx.QueryRowContext(context.Background(),
		"SELECT max_id, step FROM leaf_alloc WHERE biz_tag = ?", bizTag).Scan(&maxId, &step)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// Construct a Segment: [maxId-step, maxId]
	return &Segment{
		Base:   maxId - int64(step),
		Max:    maxId,
		Step:   step,
		Cursor: maxId - int64(step), // Cursor always starts at Base
	}, nil
}

// LeafServer manages DoubleBuffers for each business tag, serving as the main point for ID generation.
type LeafServer struct {
	dao     *LeafDAO
	buffers map[string]*DoubleBuffer // per-biz segment double buffer
	mu      sync.RWMutex             // reads/writes to buffers map protected
}

// NewLeafServer creates a new LeafServer with given DB connection string.
func NewLeafServer(dsn string) (*LeafServer, error) {
	dao, err := NewLeafDAO(dsn)
	if err != nil {
		return nil, err
	}

	return &LeafServer{
		dao:     dao,
		buffers: make(map[string]*DoubleBuffer),
	}, nil
}

// GetID returns the next available unique ID for the chosen business tag.
// Instantiates new DoubleBuffer if required. Thread safe.
func (s *LeafServer) GetID(bizTag string) (int64, error) {
	// Fast path with read lock: check if buffer exists.
	s.mu.RLock()
	buf, ok := s.buffers[bizTag]
	s.mu.RUnlock()

	if ok {
		return buf.NextID()
	}

	// Fallback: allocate new DoubleBuffer (write lock required).
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double check in case another goroutine created the buffer in between locks.
	buf, ok = s.buffers[bizTag]
	if ok {
		return buf.NextID()
	}

	buf = NewDoubleBuffer(bizTag, s.dao)
	if err := buf.Init(); err != nil {
		return 0, fmt.Errorf("failed to initialize double buffer: %w", err)
	}

	s.buffers[bizTag] = buf
	return buf.NextID()
}

func main() {
	// Please modify this DSN with your real DB credentials before use.
	dsn := "lzww:123456@tcp(127.0.0.1:3306)/test_db?parseTime=true"

	server, err := NewLeafServer(dsn)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Leaf Server Started...")

	var wg sync.WaitGroup
	start := time.Now()

	// Simulate 10 concurrent goroutines, each acquiring 500 IDs
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				id, err := server.GetID("order-service")
				if err != nil {
					log.Printf("Error: %v", err)
				} else {
					// Print every 100th allocated ID for demonstration, if needed
					if id%100 == 0 {
						// fmt.Printf("Routine %d Got ID: %d\n", routineID, id)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)
	log.Printf("Total time: %s, Finish generating 5000 IDs", elapsed)
}
