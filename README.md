# GUUID

GUUID is a lightweight and efficient library for generating Universally Unique Identifiers (UUIDs) in Go. It provides fast, standards-compliant UUID generation suitable for distributed systems, databases, and any application requiring unique identifiers. GUUID supports multiple UUID versions and is designed for simplicity, performance, and ease of integration.

## Features

- **UUIDv7 Support**: Generate time-ordered UUIDs with millisecond precision timestamps, perfect for database primary keys and distributed systems requiring sortable identifiers
- **High Performance**: Optimized for speed with minimal memory allocations
- **Standards Compliant**: Follows RFC 4122 and draft RFC for UUIDv7 specifications
- **Thread Safe**: Concurrent generation without performance penalties
- **Zero Dependencies**: Pure Go implementation with no external dependencies

## UUIDv7 Overview

UUIDv7 is the latest UUID version that combines the benefits of time-based ordering with cryptographic randomness. Unlike traditional UUIDs, UUIDv7 generates identifiers that are naturally sortable by creation time, making them ideal for:

- Database primary keys (improved B-tree performance)
- Distributed systems requiring time-ordered identifiers
- Event sourcing and audit logs
- Any scenario where chronological ordering matters

Each UUIDv7 contains:
- 48-bit timestamp (millisecond precision)
- 12-bit random data for sub-millisecond ordering
- 62-bit random data for uniqueness
- Version and variant bits as per RFC specification

## Comparison: Snowflake Algorithm vs UUIDv7

| Feature | Snowflake Algorithm | UUIDv7 |
|---------|-------------------|---------|
| Total Size | 64-bit (8 bytes) | 128-bit (16 bytes) |
| Data Type | bigint (long integer) | UUID / binary(16) |
| ID Structure | 1-bit sign + 41-bit timestamp + 10-bit node ID + 12-bit sequence | 48-bit timestamp + 6-bit version/variant + 74-bit random data |
| Conflict Avoidance | Assign unique node ID (Worker ID) | Probabilistic (relies on 74-bit random data) |
| Sortability | Yes (chronologically increasing) | Yes (chronologically increasing) |
| Generation Limits | Hard limit: 4096 per node per millisecond (12-bit sequence) | No hard limit: probabilistic, can generate astronomical numbers in same millisecond |
| Operational Cost | High | Zero |

https://www.rfc-editor.org/rfc/rfc9562.html