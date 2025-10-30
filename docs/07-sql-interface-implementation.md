# 问题 7: SQL 接口实现错误

## 问题起因

为了让 UUID 能够与数据库无缝集成，需要实现 `sql.Scanner` 和 `driver.Valuer` 接口。实现不正确会导致数据库操作失败或数据损坏。

## 问题表现

### 错误实现 1: Scanner 接口处理类型不全（❌）

```go
import (
    "database/sql"
    "database/sql/driver"
)

// ❌ 错误：只处理字符串类型
func (u *UUID) Scan(src interface{}) error {
    // 只处理 string 类型，忽略了 []byte 和 nil
    str := src.(string)  // panic if src is not string!
    
    id, err := Parse(str)
    if err != nil {
        return err
    }
    
    *u = id
    return nil
}
```

**问题**：
- 不同数据库驱动可能传递不同类型（string, []byte, nil）
- 类型断言失败会panic
- 无法处理 NULL 值

### 错误实现 2: Valuer 接口返回错误类型（❌）

```go
// ❌ 错误：返回 []byte 而不是 string
func (u UUID) Value() (driver.Value, error) {
    // 某些数据库不支持 []byte 作为 UUID
    return u[:], nil  // ❌ 可能导致兼容性问题
}

// ❌ 错误：返回自定义类型
func (u UUID) Value() (driver.Value, error) {
    // driver.Value 只支持特定类型
    return u, nil  // ❌ UUID 不是有效的 driver.Value
}
```

**问题**：
- `driver.Value` 只接受特定类型：int64, float64, bool, []byte, string, time.Time, nil
- 返回不支持的类型会导致运行时错误

### 错误实现 3: 没有处理 nil/NULL（❌）

```go
// ❌ 错误：不处理 NULL 值
func (u *UUID) Scan(src interface{}) error {
    if src == nil {
        return fmt.Errorf("cannot scan NULL into UUID")  // ❌ 应该允许 NULL
    }
    
    // ...
}
```

## 正确实现

### ✅ 完整的 Scanner 接口实现

```go
import (
    "database/sql"
    "database/sql/driver"
    "fmt"
)

// Scan implements the sql.Scanner interface for database compatibility
func (u *UUID) Scan(src interface{}) error {
    // 处理 nil (SQL NULL)
    if src == nil {
        return nil  // ✅ 允许 NULL，UUID 保持为零值
    }
    
    switch src := src.(type) {
    case string:
        // ✅ 处理字符串类型（最常见）
        // 例如：PostgreSQL 的 UUID 类型，MySQL 的 CHAR(36)
        id, err := Parse(src)
        if err != nil {
            return err
        }
        *u = id
        return nil
        
    case []byte:
        // ✅ 处理字节数组类型
        // 可能是二进制 UUID (16 bytes) 或字符串格式
        if len(src) == 16 {
            // 直接复制 16 字节的二进制 UUID
            copy(u[:], src)
            return nil
        }
        if len(src) == 0 {
            // 空字节数组视为 NULL
            return nil
        }
        // 尝试作为字符串解析（如 MySQL 的 BINARY 类型）
        id, err := Parse(string(src))
        if err != nil {
            return err
        }
        *u = id
        return nil
        
    default:
        // ✅ 明确的错误消息
        return fmt.Errorf("guuid: cannot scan type %T into UUID", src)
    }
}
```

### ✅ 完整的 Valuer 接口实现

```go
// Value implements the driver.Valuer interface for database compatibility
func (u UUID) Value() (driver.Value, error) {
    // ✅ 返回标准字符串格式
    // 这是最兼容的方式，所有数据库都支持
    return u.String(), nil
}

// 或者，如果数据库支持二进制 UUID：
func (u UUID) ValueBinary() (driver.Value, error) {
    // ✅ 返回 []byte（某些数据库如 PostgreSQL 支持）
    return u[:], nil
}
```

### 为什么选择 String 而不是 []byte？

1. **兼容性**：所有数据库都支持字符串
2. **可读性**：在数据库工具中可以直接看到 UUID
3. **调试方便**：日志和错误消息更清晰
4. **标准化**：UUID 字符串格式是通用标准

如果确定只使用 PostgreSQL 并且需要节省空间，可以返回 []byte（16字节 vs 36字符）。

## 完整测试

### 测试 1: Scanner 接口各种输入类型

```go
func TestUUID_Scan(t *testing.T) {
    tests := []struct {
        name    string
        input   interface{}
        want    UUID
        wantErr bool
    }{
        {
            name:  "string input - canonical format",
            input: "f47ac10b-58cc-4372-a567-0e02b2c3d479",
            want:  MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
        },
        {
            name:  "string input - without hyphens",
            input: "f47ac10b58cc4372a5670e02b2c3d479",
            want:  MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
        },
        {
            name:  "[]byte input - 16 bytes (binary)",
            input: []byte{0xf4, 0x7a, 0xc1, 0x0b, 0x58, 0xcc, 0x43, 0x72, 
                         0xa5, 0x67, 0x0e, 0x02, 0xb2, 0xc3, 0xd4, 0x79},
            want: UUID{0xf4, 0x7a, 0xc1, 0x0b, 0x58, 0xcc, 0x43, 0x72,
                      0xa5, 0x67, 0x0e, 0x02, 0xb2, 0xc3, 0xd4, 0x79},
        },
        {
            name:  "[]byte input - string format",
            input: []byte("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
            want:  MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
        },
        {
            name:  "nil input (SQL NULL)",
            input: nil,
            want:  Nil,  // 零值
        },
        {
            name:  "empty []byte",
            input: []byte{},
            want:  Nil,
        },
        {
            name:    "invalid type - int",
            input:   123,
            wantErr: true,
        },
        {
            name:    "invalid string format",
            input:   "not-a-uuid",
            wantErr: true,
        },
        {
            name:    "[]byte with wrong length",
            input:   []byte{0x01, 0x02},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var uuid UUID
            err := uuid.Scan(tt.input)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if !tt.wantErr && uuid != tt.want {
                t.Errorf("Scan() got = %v, want %v", uuid, tt.want)
            }
        })
    }
}
```

### 测试 2: Valuer 接口

```go
func TestUUID_Value(t *testing.T) {
    tests := []struct {
        name string
        uuid UUID
        want string
    }{
        {
            name: "normal UUID",
            uuid: MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
            want: "f47ac10b-58cc-4372-a567-0e02b2c3d479",
        },
        {
            name: "nil UUID",
            uuid: Nil,
            want: "00000000-0000-0000-0000-000000000000",
        },
        {
            name: "newly generated UUID",
            uuid: Must(NewV7()),
            // 只验证格式，不验证具体值
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := tt.uuid.Value()
            if err != nil {
                t.Fatalf("Value() error = %v", err)
            }
            
            // 验证返回类型是 string
            str, ok := got.(string)
            if !ok {
                t.Fatalf("Value() returned %T, want string", got)
            }
            
            if tt.want != "" && str != tt.want {
                t.Errorf("Value() = %v, want %v", str, tt.want)
            }
            
            // 验证可以往返
            var uuid2 UUID
            err = uuid2.Scan(got)
            if err != nil {
                t.Fatalf("Round-trip Scan() error = %v", err)
            }
            
            if uuid2 != tt.uuid {
                t.Errorf("Round-trip failed: got %v, want %v", uuid2, tt.uuid)
            }
        })
    }
}
```

### 测试 3: 数据库集成测试

```go
import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "testing"
)

func TestDatabaseIntegration(t *testing.T) {
    // 使用内存数据库进行测试
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()
    
    // 创建测试表
    _, err = db.Exec(`
        CREATE TABLE test_uuids (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL
        )
    `)
    if err != nil {
        t.Fatal(err)
    }
    
    // 测试插入
    testUUID := Must(NewV7())
    _, err = db.Exec(
        "INSERT INTO test_uuids (id, name) VALUES (?, ?)",
        testUUID, "test",
    )
    if err != nil {
        t.Fatalf("Insert failed: %v", err)
    }
    
    // 测试查询
    var retrievedUUID UUID
    var name string
    err = db.QueryRow(
        "SELECT id, name FROM test_uuids WHERE id = ?",
        testUUID,
    ).Scan(&retrievedUUID, &name)
    
    if err != nil {
        t.Fatalf("Query failed: %v", err)
    }
    
    if retrievedUUID != testUUID {
        t.Errorf("Retrieved UUID mismatch: got %v, want %v", 
            retrievedUUID, testUUID)
    }
    
    if name != "test" {
        t.Errorf("Retrieved name mismatch: got %v, want test", name)
    }
    
    // 测试 NULL 处理
    _, err = db.Exec(
        "INSERT INTO test_uuids (id, name) VALUES (?, ?)",
        Nil, "nil-test",
    )
    if err != nil {
        t.Fatalf("Insert NULL failed: %v", err)
    }
    
    var nullableUUID sql.NullString
    err = db.QueryRow(
        "SELECT id FROM test_uuids WHERE name = ?",
        "nil-test",
    ).Scan(&nullableUUID)
    
    if err != nil {
        t.Fatalf("Query NULL failed: %v", err)
    }
}
```

### 测试 4: 不同数据库驱动的兼容性

```go
func TestDatabaseDriverCompatibility(t *testing.T) {
    // 模拟不同数据库驱动传递的类型
    
    testUUID := MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")
    
    // PostgreSQL: 可能传递 string 或 []byte
    t.Run("PostgreSQL-like", func(t *testing.T) {
        var u1, u2 UUID
        
        // 作为 string
        err := u1.Scan(testUUID.String())
        if err != nil {
            t.Errorf("Scan string failed: %v", err)
        }
        
        // 作为 []byte (binary)
        err = u2.Scan(testUUID[:])
        if err != nil {
            t.Errorf("Scan []byte failed: %v", err)
        }
        
        if u1 != u2 {
            t.Error("Different representations should yield same UUID")
        }
    })
    
    // MySQL: 通常传递 []byte (string content)
    t.Run("MySQL-like", func(t *testing.T) {
        var u UUID
        
        // MySQL CHAR(36) 返回 []byte
        err := u.Scan([]byte(testUUID.String()))
        if err != nil {
            t.Errorf("Scan MySQL []byte failed: %v", err)
        }
        
        if u != testUUID {
            t.Error("MySQL format parsing failed")
        }
    })
    
    // SQLite: 通常传递 string
    t.Run("SQLite-like", func(t *testing.T) {
        var u UUID
        
        err := u.Scan(testUUID.String())
        if err != nil {
            t.Errorf("Scan SQLite string failed: %v", err)
        }
        
        if u != testUUID {
            t.Error("SQLite format parsing failed")
        }
    })
}
```

## SQL 类型映射

### PostgreSQL

```sql
-- 使用原生 UUID 类型
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL
);

-- 或使用 BINARY(16) 存储二进制
CREATE TABLE users_binary (
    id BYTEA PRIMARY KEY,  -- 16 bytes
    name TEXT NOT NULL
);
```

```go
// 字符串格式（推荐）
var userID UUID
db.QueryRow("SELECT id FROM users WHERE name = $1", "alice").Scan(&userID)

// 二进制格式（节省空间）
db.Exec("INSERT INTO users_binary (id, name) VALUES ($1, $2)", 
    userID[:], "alice")
```

### MySQL

```sql
-- 使用 CHAR(36) 或 BINARY(16)
CREATE TABLE users (
    id CHAR(36) PRIMARY KEY,    -- 字符串格式
    -- 或
    id BINARY(16) PRIMARY KEY,  -- 二进制格式
    name VARCHAR(100) NOT NULL
);
```

```go
var userID UUID
db.QueryRow("SELECT id FROM users WHERE name = ?", "alice").Scan(&userID)
```

### SQLite

```sql
-- 使用 TEXT 类型
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL
);
```

```go
var userID UUID
db.QueryRow("SELECT id FROM users WHERE name = ?", "alice").Scan(&userID)
```

## 常见陷阱

### 陷阱 1: 忘记处理 nil

```go
// ❌ 不处理 nil
func (u *UUID) Scan(src interface{}) error {
    str := src.(string)  // panic if src is nil!
    // ...
}

// ✅ 正确处理
func (u *UUID) Scan(src interface{}) error {
    if src == nil {
        return nil
    }
    // ...
}
```

### 陷阱 2: 类型断言不完整

```go
// ❌ 只处理一种类型
func (u *UUID) Scan(src interface{}) error {
    switch src.(type) {
    case string:
        // ...
    default:
        return errors.New("unsupported type")
    }
}

// ✅ 处理所有可能的类型
func (u *UUID) Scan(src interface{}) error {
    switch src := src.(type) {
    case nil:
        return nil
    case string:
        // ...
    case []byte:
        // ...
    default:
        return fmt.Errorf("unsupported type: %T", src)
    }
}
```

### 陷阱 3: Value 返回不兼容类型

```go
// ❌ 返回自定义类型
func (u UUID) Value() (driver.Value, error) {
    return u, nil  // UUID 不是 driver.Value
}

// ✅ 返回支持的类型
func (u UUID) Value() (driver.Value, error) {
    return u.String(), nil  // string 是有效的 driver.Value
}
```

## 最佳实践

1. ✅ **Scan 必须处理 nil**
2. ✅ **Scan 应处理 string 和 []byte 两种类型**
3. ✅ **Value 返回 string（最兼容）或 []byte**
4. ✅ **编写数据库集成测试**
5. ✅ **测试不同数据库驱动的兼容性**
6. ✅ **提供清晰的错误消息**
7. ✅ **支持往返转换（Value → Scan → Value）**

## 总结

SQL 接口实现的关键点：
- ✅ Scan 必须处理 nil, string, []byte 三种输入
- ✅ Value 返回 string 以获得最佳兼容性
- ✅ 正确处理 NULL 值（nil）
- ✅ 提供有用的错误消息
- ✅ 编写完整的数据库集成测试
- ✅ 验证与不同数据库驱动的兼容性
- ✅ 确保往返转换正确无误

