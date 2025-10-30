# 问题 3: 时间戳精度错误

## 问题起因

UUIDv7 规范要求使用**毫秒精度**的 Unix 时间戳（48 位）。如果使用了错误的时间单位（秒、纳秒或微秒），会导致 UUID 无法正确排序或时间戳提取错误。

## 问题表现

### 错误实现 1: 使用秒级时间戳（❌）

```go
func (g *Generator) NewWithTime(t time.Time) (UUID, error) {
    var uuid UUID
    
    // ❌ 错误：使用秒级时间戳
    timestamp := uint64(t.Unix())  // 秒，不是毫秒！
    
    g.mu.Lock()
    defer g.mu.Unlock()
    
    // 时间戳只会每秒变化一次
    // 在同一秒内生成的所有 UUID 时间戳都相同
    binary.BigEndian.PutUint64(uuid[0:8], timestamp<<16)
    // ...
    return uuid, nil
}
```

**问题**：
- 在同一秒内生成的所有 UUID 时间戳完全相同
- 时钟序列会快速溢出（1 秒内可能生成数万个 UUID）
- 排序粒度太粗（只能按秒排序）

### 错误实现 2: 使用纳秒级时间戳（❌）

```go
func (g *Generator) NewWithTime(t time.Time) (UUID, error) {
    var uuid UUID
    
    // ❌ 错误：使用纳秒级时间戳
    timestamp := uint64(t.UnixNano())  // 纳秒，太大了！
    
    // Unix 纳秒时间戳约为 1.6 × 10^18
    // 需要 60+ 位才能表示，但我们只有 48 位！
    binary.BigEndian.PutUint64(uuid[0:8], timestamp<<16)  // ❌ 高位会被截断
    // ...
    return uuid, nil
}
```

**问题**：
- 纳秒时间戳需要约 60 位，UUID 只有 48 位
- 高位被截断，导致时间戳错误
- UUID 无法正确排序

### 错误实现 3: 位移错误（❌）

```go
func (g *Generator) NewWithTime(t time.Time) (UUID, error) {
    var uuid UUID
    timestamp := uint64(t.UnixMilli())  // ✅ 毫秒正确
    
    // ❌ 错误：位移量不对
    binary.BigEndian.PutUint64(uuid[0:8], timestamp)  // 应该左移 16 位！
    
    // 或者
    binary.BigEndian.PutUint64(uuid[0:8], timestamp<<8)  // ❌ 位移量错误
    // ...
    return uuid, nil
}
```

## 正确实现

### ✅ 正确的时间戳处理

```go
func (g *Generator) NewWithTime(t time.Time) (UUID, error) {
    var uuid UUID
    
    // ✅ 正确：使用毫秒精度
    timestamp := uint64(t.UnixMilli())  // 毫秒级 Unix 时间戳
    
    g.mu.Lock()
    defer g.mu.Unlock()
    
    // 单调性处理...
    if timestamp <= g.lastTimestamp {
        g.clockSeq++
        if g.clockSeq > 0xFFF {
            g.clockSeq = 0
            timestamp = g.lastTimestamp + 1
            g.lastTimestamp = timestamp
        }
    } else {
        var randBytes [2]byte
        if _, err := io.ReadFull(g.randReader, randBytes[:]); err != nil {
            return uuid, err
        }
        g.clockSeq = binary.BigEndian.Uint16(randBytes[:]) & 0xFFF
        g.lastTimestamp = timestamp
    }
    
    // ✅ 正确：时间戳占用前 48 位，左移 16 位放到 64 位整数的高位
    // 布局：[48-bit timestamp][16-bit zero padding]
    binary.BigEndian.PutUint64(uuid[0:8], timestamp<<16)
    
    // 解释：
    // - uuid[0:8] 是前 8 个字节（64 位）
    // - 时间戳是 48 位，左移 16 位后占据这 64 位的高 48 位
    // - 低 16 位用于存储版本和时钟序列
    
    // ✅ 正确：设置版本和时钟序列
    uuid[6] = byte(0x70 | (g.clockSeq >> 8))  // 版本 7 (0111) + clockSeq 高 4 位
    uuid[7] = byte(g.clockSeq)                 // clockSeq 低 8 位
    
    // 生成随机数据和设置变体位
    if _, err := io.ReadFull(g.randReader, uuid[8:]); err != nil {
        return uuid, err
    }
    uuid[8] = (uuid[8] & 0x3F) | 0x80  // 变体位设置为 10
    
    return uuid, nil
}
```

### ✅ 正确的时间戳提取

```go
// Timestamp 从 UUIDv7 提取 Unix 时间戳（毫秒）
func (u UUID) Timestamp() int64 {
    if u.Version() != VersionTimeSorted {
        return 0
    }
    
    // ✅ 正确：提取前 48 位时间戳
    // 字节 0-5 包含完整的 48 位时间戳
    timestamp := uint64(u[0])<<40 |  // 最高 8 位
                 uint64(u[1])<<32 |  // 次高 8 位
                 uint64(u[2])<<24 |  // ...
                 uint64(u[3])<<16 |
                 uint64(u[4])<<8 |
                 uint64(u[5])        // 最低 8 位（时间戳的）
    
    return int64(timestamp)
}

// Time 返回 time.Time 对象
func (u UUID) Time() time.Time {
    if u.Version() != VersionTimeSorted {
        return time.Time{}
    }
    
    ms := u.Timestamp()
    
    // ✅ 正确：毫秒转换为 time.Time
    seconds := ms / 1000
    nanos := (ms % 1000) * 1000000  // 毫秒余数转纳秒
    
    return time.Unix(seconds, nanos)
}
```

## 测试验证

### 测试 1: 时间戳精度

```go
func TestTimestampPrecision(t *testing.T) {
    gen := NewGenerator()
    
    now := time.Now()
    expectedMs := now.UnixMilli()
    
    uuid, err := gen.NewWithTime(now)
    if err != nil {
        t.Fatalf("Generation error: %v", err)
    }
    
    // 验证时间戳精度
    actualMs := uuid.Timestamp()
    
    if actualMs != expectedMs {
        t.Errorf("Timestamp mismatch: got %d, want %d", actualMs, expectedMs)
        t.Errorf("Difference: %d ms", actualMs-expectedMs)
    }
}
```

### 测试 2: 时间往返

```go
func TestTimeRoundTrip(t *testing.T) {
    gen := NewGenerator()
    
    // 使用不同精度的时间
    testTimes := []time.Time{
        time.Unix(1000000000, 0),                    // 整秒
        time.Unix(1000000000, 123000000),            // 秒 + 毫秒
        time.Unix(1698765432, 100000000),            // 现代时间戳
        time.Now(),                                   // 当前时间
        time.Date(2025, 10, 30, 12, 0, 0, 0, time.UTC), // 特定日期
    }
    
    for i, original := range testTimes {
        uuid, err := gen.NewWithTime(original)
        if err != nil {
            t.Fatalf("Test %d: generation error: %v", i, err)
        }
        
        recovered := uuid.Time()
        
        // 比较毫秒精度（纳秒精度会丢失）
        originalMs := original.UnixMilli()
        recoveredMs := recovered.UnixMilli()
        
        if originalMs != recoveredMs {
            t.Errorf("Test %d: time mismatch", i)
            t.Errorf("  Original:  %v (%d ms)", original, originalMs)
            t.Errorf("  Recovered: %v (%d ms)", recovered, recoveredMs)
        }
    }
}
```

### 测试 3: 48 位范围验证

```go
func TestTimestampRange(t *testing.T) {
    // 48 位时间戳的有效范围
    const maxTimestamp = (1 << 48) - 1  // 281,474,976,710,655 毫秒
    
    // 转换为日期
    maxSeconds := int64(maxTimestamp / 1000)
    maxTime := time.Unix(maxSeconds, 0)
    
    t.Logf("48-bit timestamp range:")
    t.Logf("  Max value: %d ms", maxTimestamp)
    t.Logf("  Max date: %v", maxTime)
    t.Logf("  Valid until: %d (year)", maxTime.Year())
    
    // 验证：48 位时间戳可以表示到公元 10889 年
    if maxTime.Year() < 10000 {
        t.Error("48-bit timestamp range too small")
    }
    
    // 验证当前时间在有效范围内
    now := time.Now()
    nowMs := uint64(now.UnixMilli())
    
    if nowMs > maxTimestamp {
        t.Error("Current time exceeds 48-bit range!")
    }
    
    t.Logf("Current time: %d ms (%.1f%% of max)", 
        nowMs, float64(nowMs)/float64(maxTimestamp)*100)
}
```

### 测试 4: 排序正确性

```go
func TestTimestampOrdering(t *testing.T) {
    gen := NewGenerator()
    
    // 生成时间跨度为 1 秒的 UUID
    baseTime := time.Now()
    var uuids []UUID
    
    for i := 0; i < 10; i++ {
        t := baseTime.Add(time.Duration(i*100) * time.Millisecond)
        uuid, _ := gen.NewWithTime(t)
        uuids = append(uuids, uuid)
    }
    
    // 验证 UUID 按时间顺序排列
    for i := 1; i < len(uuids); i++ {
        if uuids[i].Compare(uuids[i-1]) <= 0 {
            t.Errorf("UUIDs not in time order at index %d", i)
            t.Errorf("  UUID[%d]: %v (timestamp: %d)", 
                i-1, uuids[i-1], uuids[i-1].Timestamp())
            t.Errorf("  UUID[%d]: %v (timestamp: %d)", 
                i, uuids[i], uuids[i].Timestamp())
        }
        
        // 验证时间戳差异
        timeDiff := uuids[i].Timestamp() - uuids[i-1].Timestamp()
        expectedDiff := int64(100) // 100 毫秒
        
        if timeDiff != expectedDiff {
            t.Errorf("Timestamp difference incorrect: got %d ms, want %d ms", 
                timeDiff, expectedDiff)
        }
    }
}
```

## 常见错误对比

| 方法 | 代码 | 问题 | 影响 |
|------|------|------|------|
| ❌ 使用秒 | `t.Unix()` | 精度太低 | 1 秒内所有 UUID 时间戳相同 |
| ❌ 使用纳秒 | `t.UnixNano()` | 范围太大 | 48 位无法容纳，高位被截断 |
| ❌ 使用微秒 | `t.UnixMicro()` | 范围太大 | 48 位约在 2248 年溢出 |
| ✅ 使用毫秒 | `t.UnixMilli()` | 正确 | 平衡精度和范围，可用到 10889 年 |

## 位布局说明

UUID v7 的前 8 字节布局：

```
Byte:    0        1        2        3        4        5        6        7
       +--------+--------+--------+--------+--------+--------+--------+--------+
Bits:  |                    48-bit timestamp (ms)              | ver  | clock |
       +--------+--------+--------+--------+--------+--------+--------+--------+
       |<-------------- 48 bits ------------->|<-4->|<---- 12 bits ---->|

- Bytes 0-5 (48 bits): Unix 毫秒时间戳
- Byte 6 高 4 位: 版本号 (0111 = 7)
- Byte 6 低 4 位 + Byte 7: 时钟序列 (12 bits)
```

## 总结

时间戳处理的关键点：
- ✅ 必须使用 `time.UnixMilli()` 获取毫秒时间戳
- ✅ 48 位足够表示到公元 10889 年
- ✅ 毫秒精度是 UUID 排序的基础
- ✅ 正确的位移和字节布局至关重要
- ✅ 编写测试验证时间往返和排序正确性
- ✅ 理解不同时间单位的范围和精度权衡

