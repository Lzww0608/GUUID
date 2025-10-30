# 问题 1: 并发竞态条件（Race Condition）

## 问题起因

在实现 UUIDv7 生成器时，需要维护一个单调递增的时钟序列号（`clockSeq`）和上次的时间戳（`lastTimestamp`）。如果多个 goroutine 同时调用 `New()` 方法，就会出现并发访问共享状态的情况。

## 问题表现

### 错误的实现（❌ 有竞态条件）

```go
type Generator struct {
    lastTimestamp uint64
    clockSeq      uint16
    randReader    io.Reader
    // 注意：没有互斥锁！
}

func (g *Generator) New() (UUID, error) {
    var uuid UUID
    timestamp := uint64(time.Now().UnixMilli())
    
    // ⚠️ 多个 goroutine 同时读写 g.lastTimestamp 和 g.clockSeq
    if timestamp <= g.lastTimestamp {
        g.clockSeq++  // 竞态条件！
        if g.clockSeq > 0xFFF {
            g.clockSeq = 0
            timestamp = g.lastTimestamp + 1
        }
    } else {
        // 生成新的随机时钟序列
        var randBytes [2]byte
        io.ReadFull(g.randReader, randBytes[:])
        g.clockSeq = binary.BigEndian.Uint16(randBytes[:]) & 0xFFF
        g.lastTimestamp = timestamp  // 竞态条件！
    }
    
    // ... 编码 UUID
    return uuid, nil
}
```

### 问题演示

运行并发测试会触发竞态检测器：

```go
func TestConcurrentGeneration(t *testing.T) {
    gen := NewGenerator()
    
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            gen.New()  // 并发调用
        }()
    }
    wg.Wait()
}
```

运行测试：
```bash
$ go test -race ./...
==================
WARNING: DATA RACE
Read at 0x00c000014090 by goroutine 7:
  main.(*Generator).New()
      generator.go:15 +0x45

Previous write at 0x00c000014090 by goroutine 6:
  main.(*Generator).New()
      generator.go:18 +0x123
==================
```

## 解决方案

### 正确的实现（✅ 线程安全）

```go
type Generator struct {
    mu            sync.Mutex  // ✅ 添加互斥锁
    lastTimestamp uint64
    clockSeq      uint16
    randReader    io.Reader
}

func (g *Generator) NewWithTime(t time.Time) (UUID, error) {
    var uuid UUID
    timestamp := uint64(t.UnixMilli())
    
    g.mu.Lock()         // ✅ 加锁保护共享状态
    defer g.mu.Unlock() // ✅ 确保解锁
    
    // 现在这段代码是线程安全的
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
    
    // 编码 UUID（不需要持有锁）
    binary.BigEndian.PutUint64(uuid[0:8], timestamp<<16)
    uuid[6] = byte(0x70 | (g.clockSeq >> 8))
    uuid[7] = byte(g.clockSeq)
    
    if _, err := io.ReadFull(g.randReader, uuid[8:]); err != nil {
        return uuid, err
    }
    uuid[8] = (uuid[8] & 0x3F) | 0x80
    
    return uuid, nil
}
```

## 验证测试

```go
func TestGenerator_ConcurrentSafety(t *testing.T) {
    gen := NewGenerator()
    const goroutines = 100
    const uuidsPerGoroutine = 100
    
    results := make(chan UUID, goroutines*uuidsPerGoroutine)
    var wg sync.WaitGroup
    
    // 启动多个 goroutine 并发生成 UUID
    for i := 0; i < goroutines; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < uuidsPerGoroutine; j++ {
                uuid, err := gen.New()
                if err != nil {
                    t.Errorf("Generation error: %v", err)
                    return
                }
                results <- uuid
            }
        }()
    }
    
    wg.Wait()
    close(results)
    
    // 验证所有 UUID 都是唯一的
    seen := make(map[UUID]bool)
    for uuid := range results {
        if seen[uuid] {
            t.Errorf("Duplicate UUID generated: %v", uuid)
        }
        seen[uuid] = true
    }
    
    if len(seen) != goroutines*uuidsPerGoroutine {
        t.Errorf("Expected %d unique UUIDs, got %d", 
            goroutines*uuidsPerGoroutine, len(seen))
    }
}
```

运行竞态检测：
```bash
$ go test -race -run TestGenerator_ConcurrentSafety
PASS
ok      github.com/lab2439/guuid    1.028s
```

## 关键要点

1. **识别共享状态**：`lastTimestamp` 和 `clockSeq` 被多个 goroutine 访问
2. **使用互斥锁**：`sync.Mutex` 保护临界区
3. **最小化锁范围**：只在必要时持有锁，UUID 编码部分不需要锁
4. **使用 defer 解锁**：确保即使发生 panic 也能释放锁
5. **运行竞态检测器**：`go test -race` 是发现并发问题的利器

## 性能影响

虽然加了锁，但性能影响很小：

```bash
BenchmarkGenerator_NewConcurrent-32    2666643    440.5 ns/op    16 B/op    1 allocs/op
```

锁的开销被 UUID 生成的其他操作（随机数生成、编码等）摊薄了。

## 避免死锁

确保不要在持有锁时调用其他可能加锁的方法：

```go
// ❌ 错误：可能导致死锁
func (g *Generator) BadMethod() {
    g.mu.Lock()
    defer g.mu.Unlock()
    
    // 如果 SomeOtherMethod 也尝试获取 g.mu，会导致死锁
    g.SomeOtherMethod()
}

// ✅ 正确：分离锁的作用域
func (g *Generator) GoodMethod() {
    g.mu.Lock()
    data := g.sharedData
    g.mu.Unlock()
    
    // 不持有锁时调用其他方法
    processData(data)
}
```

## 总结

并发安全是 Go 程序中最容易出错的地方之一。记住：
- ✅ 使用 `go test -race` 检测竞态条件
- ✅ 使用 `sync.Mutex` 保护共享状态
- ✅ 最小化临界区范围
- ✅ 使用 `defer` 确保解锁
- ✅ 编写并发测试验证线程安全性

