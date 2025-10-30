package guuid

import (
	"bytes"
	"crypto/rand"
	"testing"
	"time"
)

func TestNewV7(t *testing.T) {
	uuid, err := NewV7()
	if err != nil {
		t.Fatalf("NewV7() error = %v", err)
	}

	if uuid.IsNil() {
		t.Error("NewV7() returned nil UUID")
	}

	if uuid.Version() != VersionTimeSorted {
		t.Errorf("NewV7() version = %v, want %v", uuid.Version(), VersionTimeSorted)
	}

	if uuid.Variant() != VariantRFC4122 {
		t.Errorf("NewV7() variant = %v, want %v", uuid.Variant(), VariantRFC4122)
	}
}

func TestGenerator_New(t *testing.T) {
	gen := NewGenerator()

	uuid, err := gen.New()
	if err != nil {
		t.Fatalf("Generator.New() error = %v", err)
	}

	if uuid.IsNil() {
		t.Error("Generator.New() returned nil UUID")
	}

	if uuid.Version() != VersionTimeSorted {
		t.Errorf("Generator.New() version = %v, want %v", uuid.Version(), VersionTimeSorted)
	}

	if uuid.Variant() != VariantRFC4122 {
		t.Errorf("Generator.New() variant = %v, want %v", uuid.Variant(), VariantRFC4122)
	}
}

func TestGenerator_NewWithTime(t *testing.T) {
	gen := NewGenerator()
	now := time.Now()

	uuid, err := gen.NewWithTime(now)
	if err != nil {
		t.Fatalf("Generator.NewWithTime() error = %v", err)
	}

	// Check that timestamp is approximately correct (within 1 second)
	uuidTime := uuid.Time()
	diff := now.Sub(uuidTime)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Second {
		t.Errorf("UUID timestamp differs by %v, expected less than 1 second", diff)
	}
}

func TestGenerator_Monotonicity(t *testing.T) {
	gen := NewGenerator()
	now := time.Now()

	// Generate multiple UUIDs with the same timestamp
	const count = 100
	uuids := make([]UUID, count)

	for i := 0; i < count; i++ {
		uuid, err := gen.NewWithTime(now)
		if err != nil {
			t.Fatalf("Generator.NewWithTime() error = %v", err)
		}
		uuids[i] = uuid
	}

	// Verify all UUIDs are unique and monotonically increasing
	for i := 1; i < count; i++ {
		if uuids[i].Equal(uuids[i-1]) {
			t.Errorf("Generated duplicate UUID at index %d", i)
		}
		if uuids[i].Compare(uuids[i-1]) <= 0 {
			t.Errorf("UUIDs not monotonically increasing at index %d: %v <= %v", i, uuids[i], uuids[i-1])
		}
	}
}

func TestGenerator_ConcurrentSafety(t *testing.T) {
	gen := NewGenerator()
	const goroutines = 10
	const uuidsPerGoroutine = 100

	results := make(chan UUID, goroutines*uuidsPerGoroutine)
	done := make(chan bool, goroutines)

	// Start multiple goroutines generating UUIDs concurrently
	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < uuidsPerGoroutine; j++ {
				uuid, err := gen.New()
				if err != nil {
					t.Errorf("Concurrent generation error: %v", err)
					return
				}
				results <- uuid
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < goroutines; i++ {
		<-done
	}
	close(results)

	// Check for uniqueness
	seen := make(map[UUID]bool)
	for uuid := range results {
		if seen[uuid] {
			t.Errorf("Duplicate UUID generated in concurrent test: %v", uuid)
		}
		seen[uuid] = true
	}

	if len(seen) != goroutines*uuidsPerGoroutine {
		t.Errorf("Expected %d unique UUIDs, got %d", goroutines*uuidsPerGoroutine, len(seen))
	}
}

func TestUUID_Timestamp(t *testing.T) {
	gen := NewGenerator()
	now := time.Now()

	uuid, err := gen.NewWithTime(now)
	if err != nil {
		t.Fatalf("Generator.NewWithTime() error = %v", err)
	}

	timestamp := uuid.Timestamp()
	expectedTimestamp := now.UnixMilli()

	if timestamp != expectedTimestamp {
		t.Errorf("UUID.Timestamp() = %v, want %v", timestamp, expectedTimestamp)
	}
}

func TestUUID_Time(t *testing.T) {
	gen := NewGenerator()
	now := time.Now()

	uuid, err := gen.NewWithTime(now)
	if err != nil {
		t.Fatalf("Generator.NewWithTime() error = %v", err)
	}

	uuidTime := uuid.Time()

	// Compare timestamps in milliseconds (since UUIDv7 has millisecond precision)
	if now.UnixMilli() != uuidTime.UnixMilli() {
		t.Errorf("UUID.Time() = %v, want %v", uuidTime.UnixMilli(), now.UnixMilli())
	}
}

func TestMust(t *testing.T) {
	// Valid UUID should not panic
	gen := NewGenerator()
	uuid := Must(gen.New())
	if uuid.IsNil() {
		t.Error("Must() returned nil UUID")
	}

	// Error should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Must() did not panic on error")
		}
	}()

	// Create an error scenario by using a broken reader
	brokenGen := NewGeneratorWithReader(&brokenReader{})
	Must(brokenGen.New())
}

// brokenReader is a reader that always returns an error
type brokenReader struct{}

func (br *brokenReader) Read(p []byte) (n int, err error) {
	return 0, bytes.ErrTooLarge
}

func TestGenerator_ClockSeqOverflow(t *testing.T) {
	gen := NewGenerator()
	now := time.Now()

	// First call to initialize lastTimestamp
	_, err := gen.NewWithTime(now)
	if err != nil {
		t.Fatalf("NewWithTime() error = %v", err)
	}

	// Force clock sequence to near overflow
	gen.clockSeq = 0xFFE

	// Generate multiple UUIDs with same timestamp to trigger overflow
	for i := 0; i < 5; i++ {
		uuid, err := gen.NewWithTime(now)
		if err != nil {
			t.Fatalf("NewWithTime() error = %v", err)
		}
		if uuid.IsNil() {
			t.Error("NewWithTime() returned nil UUID")
		}
	}

	// After overflow, timestamp should have been incremented
	if gen.lastTimestamp <= uint64(now.UnixMilli()) {
		t.Error("Timestamp was not incremented after clock sequence overflow")
	}
}

func TestNewGeneratorWithReader(t *testing.T) {
	// Create a generator with crypto/rand
	gen := NewGeneratorWithReader(rand.Reader)

	uuid, err := gen.New()
	if err != nil {
		t.Fatalf("NewGeneratorWithReader() generation error = %v", err)
	}

	if uuid.IsNil() {
		t.Error("NewGeneratorWithReader() generated nil UUID")
	}
}

func TestUUID_Timestamp_NonV7(t *testing.T) {
	// Create a non-v7 UUID
	uuid := UUID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	timestamp := uuid.Timestamp()
	if timestamp != 0 {
		t.Errorf("Timestamp() for non-v7 UUID = %v, want 0", timestamp)
	}
}

func TestUUID_Time_NonV7(t *testing.T) {
	// Create a non-v7 UUID
	uuid := UUID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	uuidTime := uuid.Time()
	if !uuidTime.IsZero() {
		t.Errorf("Time() for non-v7 UUID = %v, want zero time", uuidTime)
	}
}

func TestNew(t *testing.T) {
	uuid, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if uuid.IsNil() {
		t.Error("New() returned nil UUID")
	}

	if uuid.Version() != VersionTimeSorted {
		t.Errorf("New() version = %v, want %v", uuid.Version(), VersionTimeSorted)
	}
}

func TestSortability(t *testing.T) {
	gen := NewGenerator()

	// Generate UUIDs over time
	uuids := make([]UUID, 10)
	for i := 0; i < 10; i++ {
		uuid, err := gen.New()
		if err != nil {
			t.Fatalf("Generation error: %v", err)
		}
		uuids[i] = uuid
		time.Sleep(time.Millisecond) // Small delay to ensure different timestamps
	}

	// Verify they are in ascending order
	for i := 1; i < len(uuids); i++ {
		if uuids[i].Compare(uuids[i-1]) <= 0 {
			t.Errorf("UUIDs not in ascending order at index %d", i)
		}
		if uuids[i].Timestamp() < uuids[i-1].Timestamp() {
			t.Errorf("Timestamps not in ascending order at index %d", i)
		}
	}
}
