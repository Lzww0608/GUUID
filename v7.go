package guuid

import (
	"crypto/rand"
	"encoding/binary"
	"io"
	"sync"
	"time"
)

// Generator is a thread-safe UUIDv7 generator that ensures monotonicity
// within the same millisecond by using a counter with random data.
type Generator struct {
	mu            sync.Mutex
	lastTimestamp uint64
	clockSeq      uint16 // 12-bit counter for sub-millisecond ordering
	randReader    io.Reader
}

// NewGenerator creates a new UUIDv7 generator with crypto/rand as the random source
func NewGenerator() *Generator {
	return &Generator{
		randReader: rand.Reader,
	}
}

// NewGeneratorWithReader creates a new UUIDv7 generator with a custom random source.
// This is primarily useful for testing with deterministic random sources.
func NewGeneratorWithReader(r io.Reader) *Generator {
	return &Generator{
		randReader: r,
	}
}

// New generates a new UUIDv7 with the current timestamp.
// This method is thread-safe and ensures monotonic ordering of UUIDs
// generated within the same millisecond.
func (g *Generator) New() (UUID, error) {
	return g.NewWithTime(time.Now())
}

// NewWithTime generates a new UUIDv7 with the specified timestamp.
// This method is thread-safe and ensures monotonic ordering.
func (g *Generator) NewWithTime(t time.Time) (UUID, error) {
	var uuid UUID

	// Get Unix timestamp in milliseconds (48 bits)
	timestamp := uint64(t.UnixMilli())

	g.mu.Lock()
	defer g.mu.Unlock()

	// Handle monotonicity: if timestamp is same or earlier, increment counter
	if timestamp <= g.lastTimestamp {
		g.clockSeq++
		// If counter overflows (> 12 bits), we need to wait or use last timestamp + 1
		if g.clockSeq > 0xFFF {
			g.clockSeq = 0
			timestamp = g.lastTimestamp + 1
			g.lastTimestamp = timestamp
		}
	} else {
		/*
		 *The 12-bit rand_a field and the 62-bit rand_b field SHOULD be filled with
		 *random data, such as from a cryptographically secure random number generator.
		 */
		// New millisecond, generate new random clock sequence
		var randBytes [2]byte
		if _, err := io.ReadFull(g.randReader, randBytes[:]); err != nil {
			return uuid, err
		}
		g.clockSeq = binary.BigEndian.Uint16(randBytes[:]) & 0xFFF // 12 bits
		g.lastTimestamp = timestamp
	}

	// Encode timestamp (48 bits) - bytes 0-5
	binary.BigEndian.PutUint64(uuid[0:8], timestamp<<16)

	// Encode version (4 bits) and clock_seq_hi (12 bits) - bytes 6-7
	// Version 7 = 0111
	uuid[6] = byte(0x70 | (g.clockSeq >> 8)) // version (4 bits) + clock_seq_hi (4 bits)
	uuid[7] = byte(g.clockSeq)               // clock_seq_lo (8 bits)

	// Generate random data for bytes 8-15 (64 bits)
	if _, err := io.ReadFull(g.randReader, uuid[8:]); err != nil {
		return uuid, err
	}

	// Set variant to RFC 4122 (10xx xxxx)
	uuid[8] = (uuid[8] & 0x3F) | 0x80

	return uuid, nil
}

// Must is a helper that wraps a call to a function returning (UUID, error)
// and panics if the error is non-nil. It is intended for use in variable
// initializations such as:
//
//	var id = guuid.Must(generator.New())
func Must(uuid UUID, err error) UUID {
	if err != nil {
		panic(err)
	}
	return uuid
}

// defaultGenerator is the package-level generator used by the New* functions
var defaultGenerator = NewGenerator()

// New generates a new UUIDv7 using the default generator.
// This is a convenience function that uses the package-level generator.
func New() (UUID, error) {
	return defaultGenerator.New()
}

// NewV7 is an alias for New() for explicit version specification
func NewV7() (UUID, error) {
	return defaultGenerator.New()
}

// Timestamp extracts the Unix timestamp (in milliseconds) from a UUIDv7
func (u UUID) Timestamp() int64 {
	if u.Version() != VersionTimeSorted {
		return 0
	}
	// Extract 48-bit timestamp from bytes 0-5
	timestamp := uint64(u[0])<<40 |
		uint64(u[1])<<32 |
		uint64(u[2])<<24 |
		uint64(u[3])<<16 |
		uint64(u[4])<<8 |
		uint64(u[5])
	return int64(timestamp)
}

// Time returns the timestamp as a time.Time for UUIDv7
func (u UUID) Time() time.Time {
	if u.Version() != VersionTimeSorted {
		return time.Time{}
	}
	ms := u.Timestamp()
	return time.Unix(ms/1000, (ms%1000)*1000000)
}
