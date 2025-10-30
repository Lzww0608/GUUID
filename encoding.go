package guuid

import (
	"encoding/base64"
	"encoding/hex"
)

// EncodeToHex encodes the UUID to a hexadecimal string without hyphens
func (u UUID) EncodeToHex() string {
	return hex.EncodeToString(u[:])
}

// EncodeToBase64 encodes the UUID to a base64 string (URL-safe, no padding)
func (u UUID) EncodeToBase64() string {
	return base64.RawURLEncoding.EncodeToString(u[:])
}

// EncodeToBase64Std encodes the UUID to a standard base64 string
func (u UUID) EncodeToBase64Std() string {
	return base64.StdEncoding.EncodeToString(u[:])
}

// DecodeFromHex decodes a hexadecimal string to UUID
func DecodeFromHex(s string) (UUID, error) {
	var uuid UUID
	if len(s) != 32 {
		return uuid, ErrInvalidFormat
	}
	_, err := hex.Decode(uuid[:], []byte(s))
	if err != nil {
		return uuid, ErrInvalidFormat
	}
	return uuid, nil
}

// DecodeFromBase64 decodes a base64 string to UUID (URL-safe encoding)
func DecodeFromBase64(s string) (UUID, error) {
	var uuid UUID
	data, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return uuid, ErrInvalidFormat
	}
	if len(data) != 16 {
		return uuid, ErrInvalidLength
	}
	copy(uuid[:], data)
	return uuid, nil
}

// DecodeFromBase64Std decodes a standard base64 string to UUID
func DecodeFromBase64Std(s string) (UUID, error) {
	var uuid UUID
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return uuid, ErrInvalidFormat
	}
	if len(data) != 16 {
		return uuid, ErrInvalidLength
	}
	copy(uuid[:], data)
	return uuid, nil
}

// FromBytes creates a UUID from a byte slice
func FromBytes(b []byte) (UUID, error) {
	var uuid UUID
	if len(b) != 16 {
		return uuid, ErrInvalidLength
	}
	copy(uuid[:], b)
	return uuid, nil
}

// MustFromBytes is like FromBytes but panics on error
func MustFromBytes(b []byte) UUID {
	uuid, err := FromBytes(b)
	if err != nil {
		panic(err)
	}
	return uuid
}
