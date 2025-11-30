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

type Segment struct {
	Base   int64 // exclude
	Max    int64 // include
	Step   int
	Cursor int64 // atomic
}

func NewSegment(base, max int64, step int) *Segment {
	return &Segment{
		Base:   base,
		Max:    max,
		Step:   step,
		Cursor: base,
	}
}

func (s *Segment) Remaining() int64 {
	cur := atomic.LoadInt64(&s.Cursor)

	return s.Max - cur
}

// double buffer prefetching strategy
type DoubleBuffer struct {
	bizTag string

	current *Segment
	next    *Segment

	nextReady bool
	isLoading int32
	mu        sync.Mutex

	// dependency
	dao *LeafDAO
}

func NewDoubleBuffer(bizTag string, dao *LeafDAO) *DoubleBuffer {
	return &DoubleBuffer{
		bizTag: bizTag,
		dao:    dao,
	}
}

func (db *DoubleBuffer) Init() error {
	seg, err := db.dao.FetchNextSegment(db.bizTag)
	if err != nil {
		return err
	}

	db.current = seg
	return nil
}

func (db *DoubleBuffer) NextID() (int64, error) {
	if db.current == nil {
		return 0, errors.New("segment not initialized")
	}

	id := atomic.AddInt64(&db.current.Cursor, 1)

	if id <= db.current.Max {
		db.CheckAndLoadNext()
		return id, nil
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if id := atomic.AddInt64(&db.current.Cursor, 1); id <= db.current.Max {
		return id, nil
	}

	if db.nextReady && db.next != nil {
		// log
		db.current = db.next
		db.next = nil
		db.nextReady = false

		id := atomic.AddInt64(&db.current.Cursor, 1)
		return id, nil
	}

	// log
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

func (db *DoubleBuffer) CheckAndLoadNext() {
	if db.nextReady || atomic.LoadInt32(&db.isLoading) == 1 {
		return
	}

	threshold := int64(float64(db.current.Step) * 0.2)
	if db.current.Remaining() > threshold {
		return
	}

	if atomic.CompareAndSwapInt32(&db.isLoading, 0, 1) {
		go func() {
			defer atomic.StoreInt32(&db.isLoading, 0)

			// time.Sleep(50 * time.Millisecond)

			// log.Printf()
			seg, err := db.dao.FetchNextSegment(db.bizTag)
			if err != nil {
				// log
				return
			}

			db.mu.Lock()
			db.next = seg
			db.nextReady = true
			db.mu.Unlock()
			// log
		}()
	}
}

type LeafDAO struct {
	db *sql.DB
}

func NewLeafDAO(dsn string) (*LeafDAO, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	return &LeafDAO{
		db: db,
	}, nil
}

func (dao *LeafDAO) FetchNextSegment(bizTag string) (*Segment, error) {
	tx, err := dao.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(context.Background(),
		"UPDATE leaf_alloc SET max_id = max_id + step WHERE biz_tag = ?", bizTag)
	if err != nil {
		return nil, err
	}

	var maxId int64
	var step int
	err = tx.QueryRowContext(context.Background(),
		"SELECT max_id, step FROM leaf_alloc WHERE biz_tag = ?", bizTag).Scan(&maxId, &step)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &Segment{
		Base:   maxId - int64(step),
		Max:    maxId,
		Step:   step,
		Cursor: maxId - int64(step),
	}, nil
}

type LeafServer struct {
	dao     *LeafDAO
	buffers map[string]*DoubleBuffer
	mu      sync.RWMutex
}

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

func (s *LeafServer) GetID(bizTag string) (int64, error) {
	s.mu.RLock()
	buf, ok := s.buffers[bizTag]
	s.mu.RUnlock()

	if ok {
		return buf.NextID()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

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
	// 请替换为你的实际数据库连接串
	dsn := "lzww:123456@tcp(127.0.0.1:3306)/test_db?parseTime=true"

	server, err := NewLeafServer(dsn)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Leaf Server Started...")

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				id, err := server.GetID("order-service")
				if err != nil {
					log.Printf("Error: %v", err)
				} else {
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
