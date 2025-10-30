package guuid

import (
	"testing"
)

func TestUUID_EncodeToHex(t *testing.T) {
	uuid := UUID{0xf4, 0x7a, 0xc1, 0x0b, 0x58, 0xcc, 0x43, 0x72, 0xa5, 0x67, 0x0e, 0x02, 0xb2, 0xc3, 0xd4, 0x79}
	expected := "f47ac10b58cc4372a5670e02b2c3d479"

	got := uuid.EncodeToHex()
	if got != expected {
		t.Errorf("EncodeToHex() = %v, want %v", got, expected)
	}
}

func TestDecodeFromHex(t *testing.T) {
	input := "f47ac10b58cc4372a5670e02b2c3d479"
	expected := UUID{0xf4, 0x7a, 0xc1, 0x0b, 0x58, 0xcc, 0x43, 0x72, 0xa5, 0x67, 0x0e, 0x02, 0xb2, 0xc3, 0xd4, 0x79}

	got, err := DecodeFromHex(input)
	if err != nil {
		t.Fatalf("DecodeFromHex() error = %v", err)
	}

	if got != expected {
		t.Errorf("DecodeFromHex() = %v, want %v", got, expected)
	}
}

func TestDecodeFromHex_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"too short", "f47ac10b58cc4372"},
		{"too long", "f47ac10b58cc4372a5670e02b2c3d479ff"},
		{"invalid hex", "g47ac10b58cc4372a5670e02b2c3d479"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeFromHex(tt.input)
			if err == nil {
				t.Errorf("DecodeFromHex() expected error for input %q", tt.input)
			}
		})
	}
}

func TestUUID_EncodeDecodeHex_RoundTrip(t *testing.T) {
	gen := NewGenerator()
	uuid, err := gen.New()
	if err != nil {
		t.Fatalf("Failed to generate UUID: %v", err)
	}

	hex := uuid.EncodeToHex()
	decoded, err := DecodeFromHex(hex)
	if err != nil {
		t.Fatalf("DecodeFromHex() error = %v", err)
	}

	if uuid != decoded {
		t.Errorf("Round-trip failed: got %v, want %v", decoded, uuid)
	}
}

func TestUUID_EncodeToBase64(t *testing.T) {
	uuid := UUID{0xf4, 0x7a, 0xc1, 0x0b, 0x58, 0xcc, 0x43, 0x72, 0xa5, 0x67, 0x0e, 0x02, 0xb2, 0xc3, 0xd4, 0x79}

	// Test URL-safe encoding
	b64 := uuid.EncodeToBase64()
	if len(b64) == 0 {
		t.Error("EncodeToBase64() returned empty string")
	}

	// Test standard encoding
	b64std := uuid.EncodeToBase64Std()
	if len(b64std) == 0 {
		t.Error("EncodeToBase64Std() returned empty string")
	}
}

func TestDecodeFromBase64(t *testing.T) {
	uuid := UUID{0xf4, 0x7a, 0xc1, 0x0b, 0x58, 0xcc, 0x43, 0x72, 0xa5, 0x67, 0x0e, 0x02, 0xb2, 0xc3, 0xd4, 0x79}

	// Encode
	b64 := uuid.EncodeToBase64()

	// Decode
	decoded, err := DecodeFromBase64(b64)
	if err != nil {
		t.Fatalf("DecodeFromBase64() error = %v", err)
	}

	if decoded != uuid {
		t.Errorf("DecodeFromBase64() = %v, want %v", decoded, uuid)
	}
}

func TestDecodeFromBase64Std(t *testing.T) {
	uuid := UUID{0xf4, 0x7a, 0xc1, 0x0b, 0x58, 0xcc, 0x43, 0x72, 0xa5, 0x67, 0x0e, 0x02, 0xb2, 0xc3, 0xd4, 0x79}

	// Encode
	b64 := uuid.EncodeToBase64Std()

	// Decode
	decoded, err := DecodeFromBase64Std(b64)
	if err != nil {
		t.Fatalf("DecodeFromBase64Std() error = %v", err)
	}

	if decoded != uuid {
		t.Errorf("DecodeFromBase64Std() = %v, want %v", decoded, uuid)
	}
}

func TestDecodeFromBase64_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"invalid base64", "!!!invalid!!!"},
		{"wrong length", "YWJj"}, // "abc" in base64, only 3 bytes
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeFromBase64(tt.input)
			if err == nil {
				t.Errorf("DecodeFromBase64() expected error for input %q", tt.input)
			}
		})
	}
}

func TestFromBytes(t *testing.T) {
	data := []byte{0xf4, 0x7a, 0xc1, 0x0b, 0x58, 0xcc, 0x43, 0x72, 0xa5, 0x67, 0x0e, 0x02, 0xb2, 0xc3, 0xd4, 0x79}
	expected := UUID{0xf4, 0x7a, 0xc1, 0x0b, 0x58, 0xcc, 0x43, 0x72, 0xa5, 0x67, 0x0e, 0x02, 0xb2, 0xc3, 0xd4, 0x79}

	got, err := FromBytes(data)
	if err != nil {
		t.Fatalf("FromBytes() error = %v", err)
	}

	if got != expected {
		t.Errorf("FromBytes() = %v, want %v", got, expected)
	}
}

func TestFromBytes_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"too short", []byte{0x01, 0x02, 0x03}},
		{"too long", make([]byte, 20)},
		{"empty", []byte{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromBytes(tt.input)
			if err != ErrInvalidLength {
				t.Errorf("FromBytes() error = %v, want %v", err, ErrInvalidLength)
			}
		})
	}
}

func TestMustFromBytes(t *testing.T) {
	data := []byte{0xf4, 0x7a, 0xc1, 0x0b, 0x58, 0xcc, 0x43, 0x72, 0xa5, 0x67, 0x0e, 0x02, 0xb2, 0xc3, 0xd4, 0x79}

	uuid := MustFromBytes(data)
	if uuid.IsNil() {
		t.Error("MustFromBytes() returned nil UUID")
	}

	// Test panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustFromBytes() did not panic on invalid input")
		}
	}()
	MustFromBytes([]byte{0x01})
}

func TestEncodingRoundTrips(t *testing.T) {
	gen := NewGenerator()

	for i := 0; i < 10; i++ {
		uuid, err := gen.New()
		if err != nil {
			t.Fatalf("Failed to generate UUID: %v", err)
		}

		// Hex round-trip
		hex := uuid.EncodeToHex()
		fromHex, err := DecodeFromHex(hex)
		if err != nil {
			t.Errorf("Hex round-trip decode error: %v", err)
		}
		if uuid != fromHex {
			t.Errorf("Hex round-trip failed: got %v, want %v", fromHex, uuid)
		}

		// Base64 round-trip
		b64 := uuid.EncodeToBase64()
		fromB64, err := DecodeFromBase64(b64)
		if err != nil {
			t.Errorf("Base64 round-trip decode error: %v", err)
		}
		if uuid != fromB64 {
			t.Errorf("Base64 round-trip failed: got %v, want %v", fromB64, uuid)
		}

		// Base64 Std round-trip
		b64std := uuid.EncodeToBase64Std()
		fromB64Std, err := DecodeFromBase64Std(b64std)
		if err != nil {
			t.Errorf("Base64Std round-trip decode error: %v", err)
		}
		if uuid != fromB64Std {
			t.Errorf("Base64Std round-trip failed: got %v, want %v", fromB64Std, uuid)
		}

		// Bytes round-trip
		bytes := uuid.Bytes()
		fromBytes, err := FromBytes(bytes)
		if err != nil {
			t.Errorf("Bytes round-trip decode error: %v", err)
		}
		if uuid != fromBytes {
			t.Errorf("Bytes round-trip failed: got %v, want %v", fromBytes, uuid)
		}
	}
}
