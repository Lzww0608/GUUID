# 问题 5: 版本位和变体位设置错误

## 问题起因

UUID 包含特殊的**版本位**（4位）和**变体位**（2-3位），用于标识 UUID 的类型和格式。如果设置错误，会导致 UUID 不符合 RFC 4122/9562 标准，无法被其他系统正确识别。

## UUID 位布局

UUIDv7 的完整128位布局：

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    unix_ts_ms (48 bits)                       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|    unix_ts_ms     | ver |     clock_seq (12 bits)             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|var|                    random (62 bits)                       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                       random (continued)                      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

字节索引：
  0    1    2    3    4    5    6    7    8    9   10   11   12   13   14   15
[     时间戳(6字节)     ][ver|clk][var|    随机数据(8字节)              ]
                         ↑   ↑    ↑
                      版本位  |  变体位
                        (4位) |  (2位)
                             时钟序列(12位)
```

## 问题表现

### 错误实现 1: 版本位设置错误（❌）

```go
func (g *Generator) NewWithTime(t time.Time) (UUID, error) {
    var uuid UUID
    timestamp := uint64(t.UnixMilli())
    
    // ... 时间戳和时钟序列处理 ...
    
    binary.BigEndian.PutUint64(uuid[0:8], timestamp<<16)
    
    // ❌ 错误 1: 版本号错误
    uuid[6] = byte(0x40 | (g.clockSeq >> 8))  // 版本 4，不是版本 7！
    uuid[7] = byte(g.clockSeq)
    
    // ❌ 错误 2: 完全覆盖了版本位
    uuid[6] = byte(g.clockSeq >> 8)  // 丢失了版本号！
    uuid[7] = byte(g.clockSeq)
    
    // ❌ 错误 3: 位掩码错误
    uuid[6] = byte(0x77 | (g.clockSeq >> 8))  // 0x77 不是正确的掩码
    uuid[7] = byte(g.clockSeq)
    
    // ...
    return uuid, nil
}
```

### 错误实现 2: 变体位设置错误（❌）

```go
func (g *Generator) NewWithTime(t time.Time) (UUID, error) {
    var uuid UUID
    
    // ... 生成前 8 个字节 ...
    
    // 生成随机数据
    io.ReadFull(g.randReader, uuid[8:])
    
    // ❌ 错误 1: 没有设置变体位
    // uuid[8] 保持随机，变体位未设置
    
    // ❌ 错误 2: 变体位值错误
    uuid[8] = (uuid[8] & 0x3F) | 0xC0  // 变体 110，不是 10！
    
    // ❌ 错误 3: 掩码错误
    uuid[8] = (uuid[8] & 0x7F) | 0x80  // 只清除了 1 位，应该清除 2 位
    
    // ❌ 错误 4: 完全覆盖
    uuid[8] = 0x80  // 丢失了随机数据！
    
    return uuid, nil
}
```

## 正确实现

### ✅ 正确设置版本位

```go
func (g *Generator) NewWithTime(t time.Time) (UUID, error) {
    var uuid UUID
    timestamp := uint64(t.UnixMilli())
    
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
    
    // 编码时间戳
    binary.BigEndian.PutUint64(uuid[0:8], timestamp<<16)
    
    // ✅ 正确设置版本位和时钟序列
    // uuid[6] 的布局：[version(4 bits)][clock_seq_hi(4 bits)]
    // version = 7 = 0111 (二进制) = 0x7
    // 左移 4 位：0111_0000 = 0x70
    uuid[6] = byte(0x70 | (g.clockSeq >> 8))
    
    // 解释：
    // - 0x70 = 0111_0000：版本 7，低 4 位为 0
    // - g.clockSeq >> 8：取时钟序列的高 4 位
    // - | 操作：将版本位和时钟序列高位组合
    
    // uuid[7] 包含时钟序列的低 8 位
    uuid[7] = byte(g.clockSeq)
    
    // 验证：确保版本位正确
    if (uuid[6] >> 4) != 0x7 {
        return uuid, fmt.Errorf("version bits incorrect: %X", uuid[6]>>4)
    }
    
    // ...
    return uuid, nil
}
```

### ✅ 正确设置变体位

```go
func (g *Generator) NewWithTime(t time.Time) (UUID, error) {
    var uuid UUID
    
    // ... 前面的代码 ...
    
    // 生成随机数据（字节 8-15）
    if _, err := io.ReadFull(g.randReader, uuid[8:]); err != nil {
        return uuid, err
    }
    
    // ✅ 正确设置变体位
    // uuid[8] 的布局：[variant(2 bits)][random(6 bits)]
    // RFC 4122 变体 = 10 (二进制)
    
    // 步骤 1: 清除高 2 位
    // 0x3F = 0011_1111 (保留低 6 位)
    // uuid[8] & 0x3F：清除高 2 位，保留低 6 位的随机数据
    
    // 步骤 2: 设置变体位
    // 0x80 = 1000_0000 (变体位 = 10)
    // | 0x80：设置高位为 10xx_xxxx
    
    uuid[8] = (uuid[8] & 0x3F) | 0x80
    
    // 解释：
    // 原始随机: 1101_0110 (0xD6)
    // & 0x3F:   0001_0110 (0x16) - 清除高 2 位
    // | 0x80:   1001_0110 (0x96) - 设置变体位为 10
    //           ↑↑
    //        变体位 = 10 (RFC 4122)
    
    // 验证：确保变体位正确
    variantBits := (uuid[8] & 0xC0) >> 6  // 提取高 2 位
    if variantBits != 0x02 {              // 0x02 = 10 (二进制)
        return uuid, fmt.Errorf("variant bits incorrect: %02b", variantBits)
    }
    
    return uuid, nil
}
```

## 版本和变体的读取

### ✅ 正确读取版本

```go
// Version 返回 UUID 的版本号
func (u UUID) Version() Version {
    // uuid[6] 的高 4 位包含版本号
    return Version(u[6] >> 4)
}

// 示例测试
func TestVersion(t *testing.T) {
    uuid, _ := NewV7()
    
    version := uuid.Version()
    if version != VersionTimeSorted {  // VersionTimeSorted = 7
        t.Errorf("Version incorrect: got %d, want %d", version, VersionTimeSorted)
    }
    
    // 直接检查字节
    versionByte := uuid[6] >> 4
    if versionByte != 0x7 {
        t.Errorf("Version byte incorrect: 0x%X", versionByte)
    }
}
```

### ✅ 正确读取变体

```go
type Variant byte

const (
    VariantNCS Variant = iota       // 0b0xxx_xxxx (NCS, 保留)
    VariantRFC4122                  // 0b10xx_xxxx (RFC 4122)
    VariantMicrosoft                // 0b110x_xxxx (Microsoft, 保留)
    VariantFuture                   // 0b111x_xxxx (未来使用, 保留)
)

// Variant 返回 UUID 的变体
func (u UUID) Variant() Variant {
    // uuid[8] 的高位决定变体
    switch {
    case (u[8] & 0x80) == 0x00:      // 0b0xxx_xxxx
        return VariantNCS
    case (u[8] & 0xC0) == 0x80:      // 0b10xx_xxxx
        return VariantRFC4122
    case (u[8] & 0xE0) == 0xC0:      // 0b110x_xxxx
        return VariantMicrosoft
    default:                          // 0b111x_xxxx
        return VariantFuture
    }
}

// 示例测试
func TestVariant(t *testing.T) {
    uuid, _ := NewV7()
    
    variant := uuid.Variant()
    if variant != VariantRFC4122 {
        t.Errorf("Variant incorrect: got %v, want %v", variant, VariantRFC4122)
    }
    
    // 直接检查字节
    variantBits := (uuid[8] & 0xC0) >> 6
    if variantBits != 0x02 {  // 0x02 = 0b10
        t.Errorf("Variant bits incorrect: 0b%02b", variantBits)
    }
}
```

## 完整测试

### 测试 1: 版本和变体位验证

```go
func TestVersionAndVariantBits(t *testing.T) {
    gen := NewGenerator()
    
    // 生成 100 个 UUID 并验证
    for i := 0; i < 100; i++ {
        uuid, err := gen.New()
        if err != nil {
            t.Fatalf("Generation error: %v", err)
        }
        
        // 检查版本位
        version := uuid.Version()
        if version != VersionTimeSorted {
            t.Errorf("UUID %d: incorrect version %d, want %d", 
                i, version, VersionTimeSorted)
            t.Errorf("  UUID: %v", uuid)
            t.Errorf("  Byte[6]: 0x%02X (binary: %08b)", uuid[6], uuid[6])
        }
        
        // 检查变体位
        variant := uuid.Variant()
        if variant != VariantRFC4122 {
            t.Errorf("UUID %d: incorrect variant %v, want %v", 
                i, variant, VariantRFC4122)
            t.Errorf("  UUID: %v", uuid)
            t.Errorf("  Byte[8]: 0x%02X (binary: %08b)", uuid[8], uuid[8])
        }
        
        // 详细的位级验证
        versionBits := (uuid[6] >> 4) & 0x0F
        if versionBits != 0x7 {
            t.Errorf("UUID %d: version bits 0x%X != 0x7", i, versionBits)
        }
        
        variantBits := (uuid[8] >> 6) & 0x03
        if variantBits != 0x02 {  // 0b10
            t.Errorf("UUID %d: variant bits 0b%02b != 0b10", i, variantBits)
        }
    }
}
```

### 测试 2: 时钟序列不破坏版本位

```go
func TestClockSeqDoesNotCorruptVersion(t *testing.T) {
    gen := NewGenerator()
    now := time.Now()
    
    // 测试所有可能的时钟序列值 (0-4095)
    for clockSeq := uint16(0); clockSeq <= 0xFFF; clockSeq++ {
        gen.clockSeq = clockSeq
        
        uuid, err := gen.NewWithTime(now)
        if err != nil {
            t.Fatalf("Generation error at clockSeq %d: %v", clockSeq, err)
        }
        
        // 验证版本位不被破坏
        version := uuid.Version()
        if version != VersionTimeSorted {
            t.Errorf("ClockSeq %d corrupted version: got %d, want 7", 
                clockSeq, version)
            t.Errorf("  Byte[6]: 0x%02X", uuid[6])
        }
        
        // 验证可以恢复时钟序列
        recoveredClockSeq := uint16(uuid[6]&0x0F)<<8 | uint16(uuid[7])
        if recoveredClockSeq != clockSeq {
            t.Errorf("ClockSeq recovery failed: got %d, want %d", 
                recoveredClockSeq, clockSeq)
        }
    }
}
```

### 测试 3: 随机数据不破坏变体位

```go
func TestRandomDataDoesNotCorruptVariant(t *testing.T) {
    gen := NewGenerator()
    
    // 生成大量 UUID，确保随机数据不会意外破坏变体位
    for i := 0; i < 10000; i++ {
        uuid, err := gen.New()
        if err != nil {
            t.Fatalf("Generation error: %v", err)
        }
        
        // 验证变体位始终正确
        variant := uuid.Variant()
        if variant != VariantRFC4122 {
            t.Errorf("UUID %d: variant corrupted", i)
            t.Errorf("  UUID: %v", uuid)
            t.Errorf("  Byte[8]: 0x%02X (binary: %08b)", uuid[8], uuid[8])
            
            // 显示变体位分析
            bit7 := (uuid[8] >> 7) & 1
            bit6 := (uuid[8] >> 6) & 1
            t.Errorf("  Variant bits: %d%d (should be 10)", bit7, bit6)
        }
    }
}
```

## 位操作可视化

### 版本位设置（uuid[6]）

```
原始 clockSeq = 0x2AB (0010_1010_1011)

步骤 1: 提取高 4 位
  clockSeq >> 8 = 0x2 (0000_0010)

步骤 2: 组合版本位
  0x70 = 0111_0000 (版本 7)
  0x02 = 0000_0010 (clockSeq 高位)
  ----   -------- OR 操作
  0x72 = 0111_0010
         ↑↑↑↑ ↑↑↑↑
         版本7 clockSeq高位

结果: uuid[6] = 0x72
```

### 变体位设置（uuid[8]）

```
原始随机字节 = 0xD6 (1101_0110)

步骤 1: 清除高 2 位
  0xD6 = 1101_0110
  0x3F = 0011_1111 (掩码)
  ----   -------- AND 操作
  0x16 = 0001_0110

步骤 2: 设置变体位
  0x16 = 0001_0110
  0x80 = 1000_0000
  ----   -------- OR 操作
  0x96 = 1001_0110
         ↑↑ ↑↑↑↑↑↑
       变体10 随机数据

结果: uuid[8] = 0x96
```

## 常见错误总结

| 错误 | 代码 | 问题 | 正确方式 |
|------|------|------|----------|
| ❌ 版本号错误 | `0x40 \| ...` | UUIDv4 不是 v7 | `0x70 \| ...` |
| ❌ 丢失版本位 | `byte(clockSeq >> 8)` | 没有设置版本 | `0x70 \| (clockSeq >> 8)` |
| ❌ 变体位错误 | `\| 0xC0` | 变体 110 | `\| 0x80` (变体 10) |
| ❌ 掩码错误 | `& 0x7F` | 只清除 1 位 | `& 0x3F` (清除 2 位) |
| ❌ 覆盖随机数据 | `uuid[8] = 0x80` | 丢失随机位 | `(uuid[8] & 0x3F) \| 0x80` |

## 总结

版本位和变体位的关键要点：
- ✅ 版本位在 uuid[6] 的高 4 位，UUIDv7 = 0x7
- ✅ 使用 `0x70 |` 设置版本位，保留低 4 位
- ✅ 变体位在 uuid[8] 的高 2 位，RFC 4122 = 0b10
- ✅ 使用 `(uuid[8] & 0x3F) | 0x80` 设置变体位
- ✅ 验证设置后的位值是否正确
- ✅ 确保时钟序列和随机数据不破坏这些位
- ✅ 编写充分的测试验证所有可能的值

