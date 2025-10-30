// Package guuid provides a lightweight and efficient implementation of Universally Unique Identifiers (UUIDs)
// in Go, with primary support for UUIDv7.
//
// UUIDv7 is a time-ordered UUID that combines the benefits of time-based ordering with cryptographic randomness.
// Unlike traditional UUIDs, UUIDv7 generates identifiers that are naturally sortable by creation time, making
// them ideal for:
//   - Database primary keys (improved B-tree performance)
//   - Distributed systems requiring time-ordered identifiers
//   - Event sourcing and audit logs
//   - Any scenario where chronological ordering matters
//
// Basic Usage:
//
//	// Generate a new UUIDv7
//	id, err := guuid.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(id.String())
//
//	// Parse a UUID from string
//	id, err := guuid.Parse("f47ac10b-58cc-4372-a567-0e02b2c3d479")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get timestamp from UUIDv7
//	timestamp := id.Timestamp()
//	time := id.Time()
//
// Custom Generator:
//
//	// Create a custom generator for better performance in tight loops
//	gen := guuid.NewGenerator()
//	for i := 0; i < 1000; i++ {
//	    id, err := gen.New()
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    // Use id...
//	}
//
// Thread Safety:
//
// All operations are thread-safe. The default generator can be used concurrently
// from multiple goroutines without additional synchronization.
//
// Standards Compliance:
//
// This implementation follows RFC 4122 and RFC 9562 specifications for UUIDs.
// The UUIDv7 format includes:
//   - 48-bit timestamp (millisecond precision)
//   - 12-bit random data for sub-millisecond ordering
//   - 62-bit random data for uniqueness
//   - Version and variant bits as per RFC specification
package guuid
