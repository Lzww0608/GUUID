package guuid

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"strings"
)

// UUID represents a Universally Unique Identifier as defined by RFC 4122 and RFC 9562.
// The UUID is a 128-bit (16 byte) value that is used to uniquely identify information.
type UUID [16]byte

// Version represents the UUID version
type Version byte

const (
	_ Version = iota
	VersionTimeBased
	VersionDCESecurity
	VersionNameBasedMD5
	VersionRandom
	VersionNameBasedSHA1
	_
	VersionTimeSorted // UUIDv7
	VersionCustom     // UUIDv8
)

// Variant represents the UUID variant
type Variant byte

const (
	VariantNCS Variant = iota
	VariantRFC4122
	VariantMicrosoft
	VariantFuture
)

// Nil is the nil UUID (all zeros)
var Nil UUID

// Version returns the version of the UUID
func (u UUID) Version() Version {
	return Version(u[6] >> 4)
}

// Variant returns the variant of the UUID
func (u UUID) Variant() Variant {
	switch {
	case (u[8] & 0x80) == 0x00:
		return VariantNCS
	case (u[8] & 0xc0) == 0x80:
		return VariantRFC4122
	case (u[8] & 0xe0) == 0xc0:
		return VariantMicrosoft
	default:
		return VariantFuture
	}
}

// String returns the canonical string representation of the UUID
// in the format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
func (u UUID) String() string {
	var buf [36]byte
	encodeHex(buf[:], u)
	return string(buf[:])
}

// encodeHex encodes UUID to its canonical hex representation
func encodeHex(dst []byte, u UUID) {
	hex.Encode(dst[0:8], u[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], u[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], u[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], u[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:36], u[10:16])
}

// Parse parses a UUID from its string representation.
// It accepts the following formats:
//   - xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (canonical)
//   - urn:uuid:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
//   - {xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx}
//   - xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx (without hyphens)
func Parse(s string) (UUID, error) {
	var uuid UUID

	// Remove common prefixes and suffixes
	s = strings.TrimPrefix(s, "urn:uuid:")
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")

	// Handle canonical format with hyphens
	if len(s) == 36 {
		if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
			return uuid, ErrInvalidFormat
		}
		// Decode each segment
		if err := decodeHexSegment(uuid[0:4], s[0:8]); err != nil {
			return uuid, err
		}
		if err := decodeHexSegment(uuid[4:6], s[9:13]); err != nil {
			return uuid, err
		}
		if err := decodeHexSegment(uuid[6:8], s[14:18]); err != nil {
			return uuid, err
		}
		if err := decodeHexSegment(uuid[8:10], s[19:23]); err != nil {
			return uuid, err
		}
		if err := decodeHexSegment(uuid[10:16], s[24:36]); err != nil {
			return uuid, err
		}
		return uuid, nil
	}

	// Handle format without hyphens
	if len(s) == 32 {
		if _, err := hex.Decode(uuid[:], []byte(s)); err != nil {
			return uuid, ErrInvalidFormat
		}
		return uuid, nil
	}

	return uuid, ErrInvalidFormat
}

// MustParse is like Parse but panics if the string cannot be parsed.
// It simplifies safe initialization of global variables.
func MustParse(s string) UUID {
	uuid, err := Parse(s)
	if err != nil {
		panic(fmt.Sprintf("guuid: Parse(%q): %v", s, err))
	}
	return uuid
}

// decodeHexSegment decodes a hex string segment into a byte slice
func decodeHexSegment(dst []byte, src string) error {
	if _, err := hex.Decode(dst, []byte(src)); err != nil {
		return ErrInvalidFormat
	}
	return nil
}

// Bytes returns the UUID as a byte slice
func (u UUID) Bytes() []byte {
	return u[:]
}

// IsNil returns true if the UUID is the nil UUID (all zeros)
func (u UUID) IsNil() bool {
	return u == Nil
}

// MarshalText implements the encoding.TextMarshaler interface
func (u UUID) MarshalText() ([]byte, error) {
	var buf [36]byte
	encodeHex(buf[:], u)
	return buf[:], nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (u *UUID) UnmarshalText(data []byte) error {
	id, err := Parse(string(data))
	if err != nil {
		return err
	}
	*u = id
	return nil
}

// MarshalBinary implements the encoding.BinaryMarshaler interface
func (u UUID) MarshalBinary() ([]byte, error) {
	return u[:], nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface
func (u *UUID) UnmarshalBinary(data []byte) error {
	if len(data) != 16 {
		return ErrInvalidLength
	}
	copy(u[:], data)
	return nil
}

// Scan implements the sql.Scanner interface for database compatibility
func (u *UUID) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		return nil
	case string:
		id, err := Parse(src)
		if err != nil {
			return err
		}
		*u = id
		return nil
	case []byte:
		if len(src) == 16 {
			copy(u[:], src)
			return nil
		}
		if len(src) == 0 {
			return nil
		}
		id, err := Parse(string(src))
		if err != nil {
			return err
		}
		*u = id
		return nil
	default:
		return fmt.Errorf("guuid: cannot scan type %T into UUID", src)
	}
}

// Value implements the driver.Valuer interface for database compatibility
func (u UUID) Value() (driver.Value, error) {
	return u.String(), nil
}

// Compare returns an integer comparing two UUIDs lexicographically.
// The result will be 0 if u==other, -1 if u < other, and +1 if u > other.
func (u UUID) Compare(other UUID) int {
	for i := 0; i < 16; i++ {
		if u[i] < other[i] {
			return -1
		}
		if u[i] > other[i] {
			return 1
		}
	}
	return 0
}

// Equal returns true if u and other represent the same UUID
func (u UUID) Equal(other UUID) bool {
	return u == other
}
