package guuid

import "errors"

var (
	// ErrInvalidFormat indicates that the UUID string format is invalid
	ErrInvalidFormat = errors.New("guuid: invalid UUID format")

	// ErrInvalidLength indicates that the UUID byte slice has incorrect length
	ErrInvalidLength = errors.New("guuid: invalid UUID length (expected 16 bytes)")

	// ErrInvalidVersion indicates that the UUID version is not supported
	ErrInvalidVersion = errors.New("guuid: invalid or unsupported UUID version")

	// ErrInvalidVariant indicates that the UUID variant is not RFC 4122
	ErrInvalidVariant = errors.New("guuid: invalid UUID variant (expected RFC 4122)")
)
