package main

import (
	"fmt"
	"log"
	"time"

	"github.com/lab2439/guuid"
)

func main() {
	fmt.Println("=== GUUID Basic Usage Examples ===\n")

	// Example 1: Generate a new UUIDv7
	fmt.Println("1. Generate a new UUIDv7:")
	uuid, err := guuid.New()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   UUID: %s\n", uuid.String())
	fmt.Printf("   Version: %d\n", uuid.Version())
	fmt.Printf("   Variant: %d\n", uuid.Variant())
	fmt.Printf("   Timestamp: %d ms\n", uuid.Timestamp())
	fmt.Printf("   Time: %s\n\n", uuid.Time().Format(time.RFC3339))

	// Example 2: Parse a UUID from string
	fmt.Println("2. Parse UUID from string:")
	uuidStr := "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	parsed, err := guuid.Parse(uuidStr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Input: %s\n", uuidStr)
	fmt.Printf("   Parsed: %s\n\n", parsed.String())

	// Example 3: Different encoding formats
	fmt.Println("3. Different encoding formats:")
	uuid2, _ := guuid.New()
	fmt.Printf("   Canonical: %s\n", uuid2.String())
	fmt.Printf("   Hex: %s\n", uuid2.EncodeToHex())
	fmt.Printf("   Base64: %s\n", uuid2.EncodeToBase64())
	fmt.Printf("   Base64Std: %s\n\n", uuid2.EncodeToBase64Std())

	// Example 4: Generate multiple UUIDs
	fmt.Println("4. Generate 5 sequential UUIDs:")
	for i := 0; i < 5; i++ {
		uuid, _ := guuid.New()
		fmt.Printf("   %d. %s (timestamp: %d)\n", i+1, uuid, uuid.Timestamp())
		time.Sleep(time.Millisecond) // Small delay to show time progression
	}
	fmt.Println()

	// Example 5: Using custom generator
	fmt.Println("5. Using custom generator:")
	gen := guuid.NewGenerator()
	uuid3, _ := gen.New()
	fmt.Printf("   Generated: %s\n\n", uuid3)

	// Example 6: Compare UUIDs
	fmt.Println("6. Compare UUIDs:")
	uuid4, _ := guuid.New()
	time.Sleep(time.Millisecond)
	uuid5, _ := guuid.New()
	result := uuid4.Compare(uuid5)
	fmt.Printf("   UUID4: %s\n", uuid4)
	fmt.Printf("   UUID5: %s\n", uuid5)
	fmt.Printf("   Compare result: %d (UUID4 %s UUID5)\n\n", result, compareResultString(result))

	// Example 7: Check if UUID is nil
	fmt.Println("7. Check nil UUID:")
	var nilUUID guuid.UUID
	fmt.Printf("   Is nil: %v\n", nilUUID.IsNil())
	fmt.Printf("   UUID: %s\n\n", nilUUID)

	// Example 8: Working with bytes
	fmt.Println("8. Working with bytes:")
	uuid6, _ := guuid.New()
	bytes := uuid6.Bytes()
	uuid7, _ := guuid.FromBytes(bytes)
	fmt.Printf("   Original: %s\n", uuid6)
	fmt.Printf("   From bytes: %s\n", uuid7)
	fmt.Printf("   Equal: %v\n", uuid6.Equal(uuid7))
}

func compareResultString(result int) string {
	switch {
	case result < 0:
		return "< "
	case result > 0:
		return ">"
	default:
		return "=="
	}
}
