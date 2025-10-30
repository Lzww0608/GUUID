package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/lab2439/guuid"
)

func main() {
	fmt.Println("=== GUUID Performance Examples ===\n")

	// Example 1: Sequential generation
	sequentialGeneration()

	// Example 2: Concurrent generation
	concurrentGeneration()

	// Example 3: Batch generation
	batchGeneration()

	// Example 4: Monotonicity test
	monotonicityTest()
}

func sequentialGeneration() {
	fmt.Println("1. Sequential Generation (1 million UUIDs):")
	start := time.Now()
	for i := 0; i < 1000000; i++ {
		_, err := guuid.New()
		if err != nil {
			panic(err)
		}
	}
	elapsed := time.Since(start)
	fmt.Printf("   Time: %s\n", elapsed)
	fmt.Printf("   Rate: %.0f UUIDs/second\n\n", 1000000/elapsed.Seconds())
}

func concurrentGeneration() {
	fmt.Println("2. Concurrent Generation (10 goroutines, 100k each):")
	const (
		goroutines = 10
		perRoutine = 100000
	)

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perRoutine; j++ {
				_, err := guuid.New()
				if err != nil {
					panic(err)
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)
	total := goroutines * perRoutine
	fmt.Printf("   Time: %s\n", elapsed)
	fmt.Printf("   Rate: %.0f UUIDs/second\n\n", float64(total)/elapsed.Seconds())
}

func batchGeneration() {
	fmt.Println("3. Batch Generation with Custom Generator:")
	gen := guuid.NewGenerator()
	const batchSize = 10000

	start := time.Now()
	uuids := make([]guuid.UUID, batchSize)
	for i := 0; i < batchSize; i++ {
		uuid, err := gen.New()
		if err != nil {
			panic(err)
		}
		uuids[i] = uuid
	}
	elapsed := time.Since(start)

	fmt.Printf("   Generated: %d UUIDs\n", batchSize)
	fmt.Printf("   Time: %s\n", elapsed)
	fmt.Printf("   Average: %s per UUID\n\n", time.Duration(elapsed.Nanoseconds()/int64(batchSize)))
}

func monotonicityTest() {
	fmt.Println("4. Monotonicity Test (same millisecond):")
	gen := guuid.NewGenerator()
	now := time.Now()

	// Generate multiple UUIDs with the same timestamp
	const count = 1000
	uuids := make([]guuid.UUID, count)

	start := time.Now()
	for i := 0; i < count; i++ {
		uuid, err := gen.NewWithTime(now)
		if err != nil {
			panic(err)
		}
		uuids[i] = uuid
	}
	elapsed := time.Since(start)

	// Check for uniqueness and monotonicity
	unique := make(map[guuid.UUID]bool)
	monotonic := true
	for i := 0; i < count; i++ {
		if unique[uuids[i]] {
			fmt.Printf("   ❌ Found duplicate UUID!\n")
		}
		unique[uuids[i]] = true

		if i > 0 && uuids[i].Compare(uuids[i-1]) <= 0 {
			monotonic = false
		}
	}

	fmt.Printf("   Generated: %d UUIDs with same timestamp\n", count)
	fmt.Printf("   Unique: %d\n", len(unique))
	fmt.Printf("   Monotonic: %v\n", monotonic)
	fmt.Printf("   Time: %s\n", elapsed)
	fmt.Printf("   First: %s\n", uuids[0])
	fmt.Printf("   Last:  %s\n\n", uuids[count-1])
}
