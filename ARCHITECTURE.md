# GUUID 项目架构

## 项目概览

GUUID 是一个高性能、符合工业标准的 Go 语言 UUID 生成库，专注于 UUIDv7 的实现。本项目采用模块化设计，确保代码的可维护性、可测试性和高性能。

## 项目结构

```
GUUID/
├── doc.go                    # 包文档和使用说明
├── errors.go                 # 错误定义
├── uuid.go                   # UUID 核心类型和基础方法
├── v7.go                     # UUIDv7 生成器实现
├── encoding.go               # 编码/解码功能
│
├── uuid_test.go              # UUID 核心功能测试
├── v7_test.go                # UUIDv7 生成器测试
├── encoding_test.go          # 编码/解码测试
├── benchmark_test.go         # 性能基准测试
│
├── examples/                 # 示例代码
│   ├── basic/               # 基础使用示例
│   ├── performance/         # 性能测试示例
│   └── database/            # 数据库集成示例
│
├── .github/
│   └── workflows/
│       └── ci.yml           # CI/CD 配置
│
├── .golangci.yml            # 代码检查配置
├── .gitignore               # Git 忽略文件配置
├── Makefile                 # 构建和测试脚本
├── go.mod                   # Go 模块定义
├── README.md                # 项目说明
├── CONTRIBUTING.md          # 贡献指南
├── ARCHITECTURE.md          # 架构文档（本文件）
└── LICENSE                  # MIT 许可证
```

## 核心模块设计

### 1. UUID 核心类型 (`uuid.go`)

**职责**：定义 UUID 的基本数据结构和通用操作

**主要组件**：
- `UUID` 类型：16 字节数组，表示标准 UUID
- `Version` 和 `Variant` 枚举：UUID 版本和变体定义
- 基础方法：
  - `String()`: 规范字符串表示
  - `Parse()`: 从字符串解析 UUID
  - `Version()` / `Variant()`: 获取版本和变体信息
  - `Compare()` / `Equal()`: UUID 比较操作
  - 序列化接口实现：
    - `encoding.TextMarshaler` / `encoding.TextUnmarshaler`
    - `encoding.BinaryMarshaler` / `encoding.BinaryUnmarshaler`
    - `sql.Scanner` / `driver.Valuer`

**设计特点**：
- 零依赖实现
- 完整的标准库接口支持
- 高效的字符串编码/解码

### 2. UUIDv7 生成器 (`v7.go`)

**职责**：实现符合 RFC 9562 的 UUIDv7 生成算法

**主要组件**：
- `Generator` 结构体：线程安全的 UUID 生成器
  - `lastTimestamp`: 上次生成的时间戳（单调性保证）
  - `clockSeq`: 12 位时钟序列号（同毫秒内的计数器）
  - `randReader`: 可配置的随机数源
- 生成方法：
  - `New()`: 使用当前时间生成 UUID
  - `NewWithTime()`: 使用指定时间生成 UUID

**UUIDv7 格式**：
```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                           unix_ts_ms                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|          unix_ts_ms           |  ver  |       clock_seq       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|var|                        random                             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                            random                             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

**单调性保证**：
1. 当时间戳前进时：生成新的随机时钟序列
2. 当时间戳相同时：递增时钟序列号
3. 当时钟序列溢出时：强制时间戳前进 +1ms

**线程安全**：
- 使用 `sync.Mutex` 保护共享状态
- 支持高并发场景

### 3. 编码模块 (`encoding.go`)

**职责**：提供多种 UUID 编码格式的支持

**支持的编码格式**：
- Hex 编码：32 字符十六进制字符串（无连字符）
- Base64 编码：22 字符 URL 安全的 Base64 字符串
- Base64 标准编码：24 字符标准 Base64 字符串
- 字节数组：16 字节原始格式

**设计特点**：
- 零内存分配的解码操作
- 高效的编码实现
- 完整的往返测试覆盖

### 4. 错误处理 (`errors.go`)

**职责**：定义库中使用的所有错误类型

**错误类型**：
- `ErrInvalidFormat`: UUID 字符串格式无效
- `ErrInvalidLength`: UUID 字节长度错误
- `ErrInvalidVersion`: 不支持的 UUID 版本
- `ErrInvalidVariant`: 非 RFC 4122 变体

## 测试策略

### 单元测试
- **覆盖率目标**：>90%
- **测试文件**：
  - `uuid_test.go`: 核心功能测试
  - `v7_test.go`: 生成器测试（包括并发安全性）
  - `encoding_test.go`: 编码/解码测试

### 关键测试场景
1. **功能正确性**：
   - UUID 格式验证
   - 版本和变体正确性
   - 序列化/反序列化往返测试

2. **单调性验证**：
   - 相同毫秒内生成的 UUID 保持递增
   - 时钟序列溢出处理
   - 时间戳前进验证

3. **并发安全性**：
   - 多 goroutine 并发生成
   - 无数据竞争
   - 唯一性保证

4. **边界条件**：
   - 时钟序列溢出
   - 无效输入处理
   - nil UUID 处理

### 基准测试

**性能指标**（基于 Intel i9-13900K）：
```
操作                         性能          内存分配
-------------------------------------------------
UUID 生成                   ~450 ns/op    16 B/op
UUID 字符串化               ~40 ns/op     48 B/op
UUID 解析（带连字符）        ~35 ns/op     0 B/op
UUID 解析（无连字符）        ~23 ns/op     0 B/op
并发生成                    ~440 ns/op    16 B/op
```

## CI/CD 流程

### GitHub Actions 工作流

**测试矩阵**：
- 操作系统：Ubuntu、macOS、Windows
- Go 版本：1.21、1.22、1.23

**流水线阶段**：
1. **测试**：
   - 运行所有单元测试
   - 启用竞态检测
   - 生成覆盖率报告
   - 上传到 Codecov

2. **代码检查**：
   - golangci-lint 静态分析
   - 代码格式检查

3. **基准测试**：
   - 运行性能基准测试
   - 确保性能回归可见

4. **示例构建**：
   - 验证所有示例代码可编译
   - 确保 API 使用正确

## 性能优化

### 内存优化
1. **零拷贝设计**：
   - UUID 使用固定大小数组（非切片）
   - 避免不必要的内存分配

2. **预分配缓冲区**：
   - 字符串编码使用固定大小缓冲区
   - 减少 GC 压力

3. **高效编码**：
   - 直接操作字节数组
   - 避免中间字符串分配

### 并发优化
1. **锁粒度**：
   - 最小化临界区范围
   - 快速释放锁

2. **无锁读操作**：
   - 时间戳提取
   - UUID 比较和格式化

## 使用场景

### 1. 数据库主键
```go
type User struct {
    ID        guuid.UUID `db:"id"`
    Username  string     `db:"username"`
    CreatedAt time.Time  `db:"created_at"`
}

// 自然时间排序，优化 B-tree 性能
```

### 2. 分布式系统
```go
// 无需中心协调的唯一 ID 生成
gen := guuid.NewGenerator()
id, _ := gen.New()
```

### 3. 事件溯源
```go
// 时间有序的事件 ID
type Event struct {
    ID        guuid.UUID
    Type      string
    Timestamp int64  // 可从 UUID 提取
}
```

## 设计决策

### 为什么选择 UUIDv7？

1. **时间排序**：自然按时间排序，优化数据库索引性能
2. **唯一性**：74 位随机数据提供极高的碰撞抵抗能力
3. **标准化**：遵循最新的 RFC 9562 规范
4. **实用性**：相比 Snowflake，无需节点 ID 管理

### 为什么不支持其他 UUID 版本？

- **专注**：UUIDv7 是最适合现代分布式系统的版本
- **简洁**：避免不必要的复杂性
- **性能**：针对单一版本优化可获得最佳性能

如需支持其他版本，可轻松扩展当前架构。

## 扩展性

### 添加新的编码格式
在 `encoding.go` 中添加新的编码/解码函数：
```go
func (u UUID) EncodeToNewFormat() string {
    // 实现新格式
}

func DecodeFromNewFormat(s string) (UUID, error) {
    // 实现解码
}
```

### 添加新的 UUID 版本
1. 在 `uuid.go` 中添加新的版本常量
2. 创建新的生成器文件（如 `v8.go`）
3. 实现生成逻辑
4. 添加相应的测试

## 依赖管理

本项目**零外部依赖**，仅使用 Go 标准库：
- `crypto/rand`: 加密安全的随机数生成
- `encoding/*`: 序列化支持
- `database/sql`: 数据库集成
- `sync`: 并发控制

## 贡献

欢迎贡献！请参阅 [CONTRIBUTING.md](CONTRIBUTING.md) 了解详细的贡献指南。

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

