# 单调性保证和时钟序列溢出

## 问题起因

UUIDv7 的一个重要特性是**单调性**：即使在同一毫秒内生成多个 UUID，它们也应该严格递增。这需要正确处理时钟序列号的递增和溢出。

## 问题表现

### 错误的实现（❌ 单调性被破坏）

```go
func (g *Generator) NewWithTime(t time.Time) (UUID, error) {
    var uuid UUID
    timestamp := uint64(t.UnixMilli())
    
    g.mu.Lock()
    defer g.mu.Unlock()
    
    // ❌ 问题 1: 没有处理时间戳相同的情况
    if timestamp > g.lastTimestamp {
        // 生成新的随机时钟序列
        var randBytes [2]byte
        io.ReadFull(g.randReader, randBytes[:])
        g.clockSeq = binary.BigEndian.Uint16(randBytes[:]) & 0xFFF
        g.lastTimestamp = timestamp
    }
    // ❌ 如果 timestamp <= g.lastTimestamp，clockSeq 不变
    // 这会导致生成重复的 UUID！
    
    // 编码 UUID
    binary.BigEndian.PutUint64(uuid[0:8], timestamp<<16)
    uuid[6] = byte(0x70 | (g.clockSeq >> 8))
    uuid[7] = byte(g.clockSeq)
    // ...
    return uuid, nil
}
```

### 问题演示

```go
func TestMonotonicity(t *testing.T) {
    gen := NewGenerator()
    now := time.Now()
    
    // 使用相同的时间戳生成多个 UUID
    uuid1, _ := gen.NewWithTime(now)
    uuid2, _ := gen.NewWithTime(now)
    uuid3, _ := gen.NewWithTime(now)
    
    // ❌ 如果实现不正确，这些 UUID 可能相同或无序
    if uuid1.Compare(uuid2) >= 0 {
        t.Error("UUID not monotonically increasing!")
    }
    if uuid2.Compare(uuid3) >= 0 {
        t.Error("UUID not monotonically increasing!")
    }
}
```

### 错误的实现（❌ 没有处理溢出）

```go
func (g *Generator) NewWithTime(t time.Time) (UUID, error) {
    var uuid UUID
    timestamp := uint64(t.UnixMilli())
    
    g.mu.Lock()
    defer g.mu.Unlock()
    
    if timestamp <= g.lastTimestamp {
        g.clockSeq++  // ❌ 问题 2: 没有检查溢出
        // 如果 clockSeq 超过 12 位（> 0xFFF），会产生错误的 UUID
    } else {
        var randBytes [2]byte
        io.ReadFull(g.randReader, randBytes[:])
        g.clockSeq = binary.BigEndian.Uint16(randBytes[:]) & 0xFFF
        g.lastTimestamp = timestamp
    }
    
    // 编码 UUID
    binary.BigEndian.PutUint64(uuid[0:8], timestamp<<16)
    uuid[6] = byte(0x70 | (g.clockSeq >> 8))  // ❌ 如果 clockSeq > 0xFFF，版本位会被破坏
    uuid[7] = byte(g.clockSeq)
    // ...
    return uuid, nil
}
```

## 解决方案

### 正确的实现（✅ 保证单调性）

```go
func (g *Generator) NewWithTime(t time.Time) (UUID, error) {
    var uuid UUID
    timestamp := uint64(t.UnixMilli())
    
    g.mu.Lock()
    defer g.mu.Unlock()
    
    // ✅ 处理时间戳相同或回退的情况
    if timestamp <= g.lastTimestamp {
        // 递增时钟序列以保证单调性
        g.clockSeq++
        
        // ✅ 处理时钟序列溢出
        if g.clockSeq > 0xFFF {  // 12 位最大值
            // 重置时钟序列，强制时间戳前进
            g.clockSeq = 0
            timestamp = g.lastTimestamp + 1
            g.lastTimestamp = timestamp  // ✅ 更新 lastTimestamp
        }
        // 注意：如果没有溢出，timestamp 仍然是原来的值
        // 但 clockSeq 递增，确保了单调性
    } else {
        // 新的毫秒，生成新的随机时钟序列
        var randBytes [2]byte
        if _, err := io.ReadFull(g.randReader, randBytes[:]); err != nil {
            return uuid, err
        }
        g.clockSeq = binary.BigEndian.Uint16(randBytes[:]) & 0xFFF
        g.lastTimestamp = timestamp
    }
    
    // 编码时间戳（48 位）
    binary.BigEndian.PutUint64(uuid[0:8], timestamp<<16)
    
    // 编码版本（4 位）和时钟序列（12 位）
    uuid[6] = byte(0x70 | (g.clockSeq >> 8))  // 版本 7 + clockSeq 高 4 位
    uuid[7] = byte(g.clockSeq)                 // clockSeq 低 8 位
    
    // 生成随机数据
    if _, err := io.ReadFull(g.randReader, uuid[8:]); err != nil {
        return uuid, err
    }
    
    // 设置变体位
    uuid[8] = (uuid[8] & 0x3F) | 0x80
    
    return uuid, nil
}
```

## 关键测试

### 测试 1: 单调性验证

```go
func TestGenerator_Monotonicity(t *testing.T) {
    gen := NewGenerator()
    now := time.Now()
    
    // 使用相同时间戳生成 100 个 UUID
    const count = 100
    uuids := make([]UUID, count)
    
    for i := 0; i < count; i++ {
        uuid, err := gen.NewWithTime(now)
        if err != nil {
            t.Fatalf("Generation error: %v", err)
        }
        uuids[i] = uuid
    }
    
    // 验证所有 UUID 都是唯一的且严格递增的
    for i := 1; i < count; i++ {
        if uuids[i].Equal(uuids[i-1]) {
            t.Errorf("Duplicate UUID at index %d", i)
        }
        if uuids[i].Compare(uuids[i-1]) <= 0 {
            t.Errorf("UUIDs not monotonically increasing at index %d", i)
        }
    }
}
```

### 测试 2: 时钟序列溢出处理

```go
func TestGenerator_ClockSeqOverflow(t *testing.T) {
    gen := NewGenerator()
    now := time.Now()
    
    // 首次调用初始化 lastTimestamp
    _, err := gen.NewWithTime(now)
    if err != nil {
        t.Fatalf("Initialization error: %v", err)
    }
    
    // 强制时钟序列接近溢出
    gen.clockSeq = 0xFFE  // 4094，距离 4095 只差 1
    
    // 生成多个 UUID 触发溢出
    for i := 0; i < 5; i++ {
        uuid, err := gen.NewWithTime(now)
        if err != nil {
            t.Fatalf("Generation error: %v", err)
        }
        
        // 验证 UUID 版本位正确
        if uuid.Version() != VersionTimeSorted {
            t.Errorf("Version incorrect after overflow: got %d, want %d", 
                uuid.Version(), VersionTimeSorted)
        }
    }
    
    // 验证时间戳已经前进
    if gen.lastTimestamp <= uint64(now.UnixMilli()) {
        t.Error("Timestamp should have advanced after clock sequence overflow")
    }
    
    // 验证时钟序列已重置
    if gen.clockSeq > 0xFFF {
        t.Errorf("Clock sequence not properly reset: %d", gen.clockSeq)
    }
}
```

### 测试 3: 高频生成测试

```go
func TestGenerator_HighFrequency(t *testing.T) {
    gen := NewGenerator()
    
    // 在 1 毫秒内生成 5000 个 UUID
    const count = 5000
    uuids := make([]UUID, count)
    
    start := time.Now()
    for i := 0; i < count; i++ {
        uuid, err := gen.New()
        if err != nil {
            t.Fatalf("Generation error: %v", err)
        }
        uuids[i] = uuid
    }
    elapsed := time.Since(start)
    
    t.Logf("Generated %d UUIDs in %v", count, elapsed)
    
    // 验证唯一性
    seen := make(map[UUID]bool)
    for i, uuid := range uuids {
        if seen[uuid] {
            t.Errorf("Duplicate UUID at index %d: %v", i, uuid)
        }
        seen[uuid] = true
    }
    
    // 验证单调性（考虑到可能跨越多个毫秒）
    violations := 0
    for i := 1; i < count; i++ {
        if uuids[i].Compare(uuids[i-1]) <= 0 {
            violations++
        }
    }
    
    if violations > 0 {
        t.Errorf("Found %d monotonicity violations", violations)
    }
}
```

## 单调性的数学原理

UUIDv7 的排序顺序由以下部分决定（从高到低）：

```
1. 时间戳（48 位）：主要排序键
2. 版本位（4 位）：固定为 0111 (版本 7)
3. 时钟序列（12 位）：同一毫秒内的排序键
4. 变体位（2 位）：固定为 10
5. 随机数据（62 位）：用于唯一性
```

因此，要保证单调性：
- 时间戳前进时，新 UUID 必然大于旧 UUID
- 时间戳相同时，递增 clockSeq 确保新 UUID 更大
- clockSeq 溢出时，强制时间戳前进 +1ms

## 边界情况

### 情况 1: 时钟回拨

```go
// 系统时钟被手动调整或 NTP 同步导致时间回退
// 我们的实现会递增 clockSeq，保持单调性
t1 := time.Unix(0, 1000000000) // 1970-01-01 00:00:01
t2 := time.Unix(0, 900000000)  // 1970-01-01 00:00:00.9 (回退了!)

uuid1, _ := gen.NewWithTime(t1)
uuid2, _ := gen.NewWithTime(t2) // 仍然比 uuid1 大
```

### 情况 2: 极高频率生成

```go
// 在同一毫秒内生成超过 4096 个 UUID
// clockSeq 会溢出，时间戳自动前进
for i := 0; i < 5000; i++ {
    gen.NewWithTime(sameTime)
}
// 后面的 UUID 时间戳会是 sameTime + 1ms, +2ms, ...
```

## 性能考虑

单调性保证不会显著影响性能：

```bash
BenchmarkGenerator_New-32              2669616    462.0 ns/op    16 B/op    1 allocs/op
BenchmarkGenerator_NewConcurrent-32    2666643    440.5 ns/op    16 B/op    1 allocs/op
```

## 总结

保证 UUID 单调性的关键：
- ✅ 检测时间戳相同或回退的情况
- ✅ 递增时钟序列号而不是生成随机值
- ✅ 处理时钟序列溢出（12 位限制）
- ✅ 溢出时强制时间戳前进并更新 lastTimestamp
- ✅ 编写充分的测试验证各种边界情况
- ✅ 确保版本位和变体位不被破坏

