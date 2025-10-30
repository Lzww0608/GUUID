package guuid

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "canonical format",
			input:   "f47ac10b-58cc-4372-a567-0e02b2c3d479",
			wantErr: false,
		},
		{
			name:    "without hyphens",
			input:   "f47ac10b58cc4372a5670e02b2c3d479",
			wantErr: false,
		},
		{
			name:    "with URN prefix",
			input:   "urn:uuid:f47ac10b-58cc-4372-a567-0e02b2c3d479",
			wantErr: false,
		},
		{
			name:    "with braces",
			input:   "{f47ac10b-58cc-4372-a567-0e02b2c3d479}",
			wantErr: false,
		},
		{
			name:    "invalid format - wrong length",
			input:   "f47ac10b-58cc-4372-a567",
			wantErr: true,
		},
		{
			name:    "invalid format - invalid hex",
			input:   "g47ac10b-58cc-4372-a567-0e02b2c3d479",
			wantErr: true,
		},
		{
			name:    "invalid format - wrong hyphen position",
			input:   "f47ac10b58cc-4372-a567-0e02b2c3d479",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uuid, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if uuid.IsNil() {
					t.Error("Parse() returned nil UUID for valid input")
				}
				// Verify round-trip
				str := uuid.String()
				uuid2, err := Parse(str)
				if err != nil {
					t.Errorf("Round-trip parse failed: %v", err)
				}
				if uuid != uuid2 {
					t.Errorf("Round-trip UUID mismatch: got %v, want %v", uuid2, uuid)
				}
			}
		})
	}
}

func TestUUID_String(t *testing.T) {
	testUUID := UUID{0xf4, 0x7a, 0xc1, 0x0b, 0x58, 0xcc, 0x43, 0x72, 0xa5, 0x67, 0x0e, 0x02, 0xb2, 0xc3, 0xd4, 0x79}
	want := "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	got := testUUID.String()
	if got != want {
		t.Errorf("String() = %v, want %v", got, want)
	}
}

func TestUUID_IsNil(t *testing.T) {
	nilUUID := Nil
	if !nilUUID.IsNil() {
		t.Error("Nil UUID should return true for IsNil()")
	}

	nonNilUUID := UUID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	if nonNilUUID.IsNil() {
		t.Error("Non-nil UUID should return false for IsNil()")
	}
}

func TestUUID_MarshalUnmarshalText(t *testing.T) {
	uuid := UUID{0xf4, 0x7a, 0xc1, 0x0b, 0x58, 0xcc, 0x43, 0x72, 0xa5, 0x67, 0x0e, 0x02, 0xb2, 0xc3, 0xd4, 0x79}

	// Marshal
	text, err := uuid.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error = %v", err)
	}

	// Unmarshal
	var uuid2 UUID
	err = uuid2.UnmarshalText(text)
	if err != nil {
		t.Fatalf("UnmarshalText() error = %v", err)
	}

	if uuid != uuid2 {
		t.Errorf("Marshal/Unmarshal mismatch: got %v, want %v", uuid2, uuid)
	}
}

func TestUUID_MarshalUnmarshalBinary(t *testing.T) {
	uuid := UUID{0xf4, 0x7a, 0xc1, 0x0b, 0x58, 0xcc, 0x43, 0x72, 0xa5, 0x67, 0x0e, 0x02, 0xb2, 0xc3, 0xd4, 0x79}

	// Marshal
	data, err := uuid.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}

	if len(data) != 16 {
		t.Errorf("MarshalBinary() length = %d, want 16", len(data))
	}

	// Unmarshal
	var uuid2 UUID
	err = uuid2.UnmarshalBinary(data)
	if err != nil {
		t.Fatalf("UnmarshalBinary() error = %v", err)
	}

	if uuid != uuid2 {
		t.Errorf("Marshal/Unmarshal mismatch: got %v, want %v", uuid2, uuid)
	}
}

func TestUUID_JSON(t *testing.T) {
	uuid := UUID{0xf4, 0x7a, 0xc1, 0x0b, 0x58, 0xcc, 0x43, 0x72, 0xa5, 0x67, 0x0e, 0x02, 0xb2, 0xc3, 0xd4, 0x79}

	type TestStruct struct {
		ID UUID `json:"id"`
	}

	ts := TestStruct{ID: uuid}

	// Marshal
	data, err := json.Marshal(ts)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal
	var ts2 TestStruct
	err = json.Unmarshal(data, &ts2)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if ts.ID != ts2.ID {
		t.Errorf("JSON Marshal/Unmarshal mismatch: got %v, want %v", ts2.ID, ts.ID)
	}
}

func TestUUID_Compare(t *testing.T) {
	uuid1 := UUID{0x01}
	uuid2 := UUID{0x02}
	uuid3 := UUID{0x01}

	if uuid1.Compare(uuid2) != -1 {
		t.Error("uuid1 should be less than uuid2")
	}

	if uuid2.Compare(uuid1) != 1 {
		t.Error("uuid2 should be greater than uuid1")
	}

	if uuid1.Compare(uuid3) != 0 {
		t.Error("uuid1 should be equal to uuid3")
	}
}

func TestUUID_Equal(t *testing.T) {
	uuid1 := UUID{0x01, 0x02, 0x03}
	uuid2 := UUID{0x01, 0x02, 0x03}
	uuid3 := UUID{0x03, 0x02, 0x01}

	if !uuid1.Equal(uuid2) {
		t.Error("uuid1 should equal uuid2")
	}

	if uuid1.Equal(uuid3) {
		t.Error("uuid1 should not equal uuid3")
	}
}

func TestUUID_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "string input",
			input:   "f47ac10b-58cc-4372-a567-0e02b2c3d479",
			wantErr: false,
		},
		{
			name:    "byte slice input - 16 bytes",
			input:   []byte{0xf4, 0x7a, 0xc1, 0x0b, 0x58, 0xcc, 0x43, 0x72, 0xa5, 0x67, 0x0e, 0x02, 0xb2, 0xc3, 0xd4, 0x79},
			wantErr: false,
		},
		{
			name:    "byte slice input - string format",
			input:   []byte("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			wantErr: false,
		},
		{
			name:    "nil input",
			input:   nil,
			wantErr: false,
		},
		{
			name:    "invalid type",
			input:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var uuid UUID
			err := uuid.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUUID_Value(t *testing.T) {
	uuid := UUID{0xf4, 0x7a, 0xc1, 0x0b, 0x58, 0xcc, 0x43, 0x72, 0xa5, 0x67, 0x0e, 0x02, 0xb2, 0xc3, 0xd4, 0x79}
	val, err := uuid.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}

	str, ok := val.(string)
	if !ok {
		t.Fatalf("Value() returned non-string type: %T", val)
	}

	expected := "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	if str != expected {
		t.Errorf("Value() = %v, want %v", str, expected)
	}
}

func TestUUID_Version(t *testing.T) {
	// Create a UUIDv7 (version 7)
	uuid := UUID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x70, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	version := uuid.Version()
	if version != VersionTimeSorted {
		t.Errorf("Version() = %v, want %v", version, VersionTimeSorted)
	}
}

func TestUUID_Variant(t *testing.T) {
	// Create a UUID with RFC 4122 variant (10xx xxxx)
	uuid := UUID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	variant := uuid.Variant()
	if variant != VariantRFC4122 {
		t.Errorf("Variant() = %v, want %v", variant, VariantRFC4122)
	}
}

func TestMustParse(t *testing.T) {
	// Valid UUID should not panic
	uuid := MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")
	if uuid.IsNil() {
		t.Error("MustParse() returned nil UUID")
	}

	// Invalid UUID should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParse() did not panic on invalid input")
		}
	}()
	MustParse("invalid-uuid")
}

func TestUUID_Bytes(t *testing.T) {
	uuid := UUID{0xf4, 0x7a, 0xc1, 0x0b, 0x58, 0xcc, 0x43, 0x72, 0xa5, 0x67, 0x0e, 0x02, 0xb2, 0xc3, 0xd4, 0x79}
	b := uuid.Bytes()
	if len(b) != 16 {
		t.Errorf("Bytes() length = %d, want 16", len(b))
	}
	if !bytes.Equal(b, uuid[:]) {
		t.Error("Bytes() did not return correct byte slice")
	}
}
