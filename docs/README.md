# GUUID 开发问题文档

本目录包含在开发 GUUID 库过程中遇到的常见问题、解决方案和最佳实践。这些文档不仅记录了实际问题，也覆盖了人类程序员在开发类似库时经常犯的错误。

## 📚 文档列表

### [01. 并发竞态条件](01-concurrency-race-condition.md)
**关键词**：并发安全、sync.Mutex、数据竞争、竞态检测

**主要内容**：
- 如何正确使用互斥锁保护共享状态
- 为什么需要 `go test -race`
- 临界区的最小化
- 死锁的避免

**适合阅读**：当你需要实现线程安全的数据结构时

---

### [02. 单调性保证和时钟序列](02-monotonicity-and-clock-sequence.md)
**关键词**：单调性、时钟序列、溢出处理、时间回拨

**主要内容**：
- UUIDv7 的单调性要求
- 时钟序列的递增和溢出处理
- 如何处理系统时钟回拨
- 高频生成场景下的单调性保证

**适合阅读**：当你需要实现时间有序的唯一ID时

---

### [03. 时间戳精度错误](03-timestamp-precision-error.md)
**关键词**：时间戳、毫秒精度、UnixMilli、位移操作

**主要内容**：
- 为什么必须使用毫秒而不是秒或纳秒
- 48 位时间戳的范围计算
- 正确的位移操作
- 时间戳的编码和解码

**适合阅读**：当你需要在二进制格式中存储时间戳时

---

### [04. 字节序（大小端）错误](04-byte-order-endianness.md)
**关键词**：字节序、大端序、BigEndian、binary包

**主要内容**：
- 大端序 vs 小端序
- 为什么 UUID 必须使用大端序
- binary.BigEndian 的正确使用
- 跨平台兼容性问题

**适合阅读**：当你需要处理二进制数据或网络协议时

---

### [05. 版本位和变体位设置](05-version-variant-bits.md)
**关键词**：版本位、变体位、位掩码、RFC 4122

**主要内容**：
- UUID 版本位的设置（4 位）
- RFC 4122 变体位的设置（2 位）
- 正确的位掩码操作
- 如何避免破坏这些特殊位

**适合阅读**：当你需要实现符合标准的二进制格式时

---

### [06. 内存分配优化](06-memory-allocation-optimization.md)
**关键词**：性能优化、内存分配、逃逸分析、零分配

**主要内容**：
- 数组 vs 切片的性能差异
- 如何实现零分配的操作
- 预分配缓冲区的技巧
- 使用基准测试验证优化

**适合阅读**：当你需要优化高频调用的函数性能时

---

### [07. SQL 接口实现](07-sql-interface-implementation.md)
**关键词**：sql.Scanner、driver.Valuer、数据库集成、NULL处理

**主要内容**：
- 如何实现 Scanner 和 Valuer 接口
- 处理不同数据库驱动的类型差异
- NULL 值的正确处理
- 数据库集成测试

**适合阅读**：当你需要让自定义类型与数据库集成时

---

## 🎯 快速导航

### 按主题查找

#### 并发和线程安全
- [01-并发竞态条件](01-concurrency-race-condition.md)

#### 时间和单调性
- [02-单调性保证和时钟序列](02-monotonicity-and-clock-sequence.md)
- [03-时间戳精度错误](03-timestamp-precision-error.md)

#### 二进制格式和编码
- [04-字节序错误](04-byte-order-endianness.md)
- [05-版本位和变体位设置](05-version-variant-bits.md)

#### 性能优化
- [06-内存分配优化](06-memory-allocation-optimization.md)

#### 数据库集成
- [07-SQL 接口实现](07-sql-interface-implementation.md)

### 按难度查找

#### 🟢 初级（必读）
- [03-时间戳精度错误](03-timestamp-precision-error.md)
- [07-SQL 接口实现](07-sql-interface-implementation.md)

#### 🟡 中级
- [01-并发竞态条件](01-concurrency-race-condition.md)
- [04-字节序错误](04-byte-order-endianness.md)
- [05-版本位和变体位设置](05-version-variant-bits.md)

#### 🔴 高级
- [02-单调性保证和时钟序列](02-monotonicity-and-clock-sequence.md)
- [06-内存分配优化](06-memory-allocation-optimization.md)

## 💡 学习建议

### 如果你是初学者
建议按以下顺序阅读：
1. [03-时间戳精度错误](03-timestamp-precision-error.md) - 理解基础概念
2. [04-字节序错误](04-byte-order-endianness.md) - 了解二进制编码
3. [01-并发竞态条件](01-concurrency-race-condition.md) - 学习并发安全
4. [07-SQL 接口实现](07-sql-interface-implementation.md) - 实践接口设计

### 如果你在解决特定问题
- **UUID 生成太慢？** → 查看 [06-内存分配优化](06-memory-allocation-optimization.md)
- **并发测试失败？** → 查看 [01-并发竞态条件](01-concurrency-race-condition.md)
- **UUID 无法排序？** → 查看 [04-字节序错误](04-byte-order-endianness.md)
- **时间戳不对？** → 查看 [03-时间戳精度错误](03-timestamp-precision-error.md)
- **数据库操作失败？** → 查看 [07-SQL 接口实现](07-sql-interface-implementation.md)
- **相同毫秒生成重复 UUID？** → 查看 [02-单调性保证和时钟序列](02-monotonicity-and-clock-sequence.md)

### 如果你在准备面试
重点阅读：
1. [01-并发竞态条件](01-concurrency-race-condition.md) - 并发相关高频考点
2. [06-内存分配优化](06-memory-allocation-optimization.md) - 性能优化常见问题
3. [02-单调性保证和时钟序列](02-monotonicity-and-clock-sequence.md) - 算法设计能力

## 🛠️ 实践建议

### 边学边做
每个文档都包含：
- ❌ 错误示例（不要这样写）
- ✅ 正确示例（应该这样写）
- 📝 完整的测试代码
- 📊 性能基准测试

**建议**：
1. 先看错误示例，思考为什么错
2. 再看正确示例，理解如何改进
3. 运行测试代码，验证理解
4. 尝试修改代码，加深印象

### 测试驱动学习
```bash
# 运行相关测试
cd /path/to/guuid
go test -v -race ./...

# 运行基准测试
go test -bench=. -benchmem ./...

# 查看代码覆盖率
go test -cover ./...
```

## 📊 问题统计

| 问题类别 | 难度 | 出现频率 | 影响范围 |
|---------|------|---------|---------|
| 并发安全 | 中 | 高 | 功能正确性 |
| 单调性 | 高 | 中 | 功能正确性 |
| 时间戳 | 低 | 高 | 功能正确性 |
| 字节序 | 中 | 中 | 兼容性 |
| 版本/变体位 | 中 | 低 | 标准兼容性 |
| 内存优化 | 高 | 低 | 性能 |
| SQL接口 | 低 | 高 | 数据库集成 |

## 🔗 相关资源

### 官方规范
- [RFC 4122 - UUID 基础规范](https://www.rfc-editor.org/rfc/rfc4122.html)
- [RFC 9562 - UUIDv7 规范](https://www.rfc-editor.org/rfc/rfc9562.html)

### Go 语言资源
- [Effective Go](https://golang.org/doc/effective_go)
- [Go 并发模式](https://go.dev/blog/pipelines)
- [Go 内存模型](https://go.dev/ref/mem)

### 工具和测试
- [Go 竞态检测器](https://go.dev/doc/articles/race_detector)
- [pprof 性能分析](https://go.dev/blog/pprof)
- [Benchstat 基准比较](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)

## 🤝 贡献

如果你发现了新的问题或有更好的解决方案，欢迎贡献：

1. Fork 项目
2. 在 `docs/` 目录下创建新的 Markdown 文件
3. 遵循现有文档的格式
4. 提交 Pull Request

文档格式建议：
```markdown
# 问题 N: 标题

## 问题起因
[描述问题的背景和原因]

## 问题表现
### 错误实现 X（❌）
[错误的代码示例]

## 正确实现
### ✅ 正确的实现
[正确的代码示例]

## 测试验证
[测试代码]

## 总结
[关键要点列表]
```

## 📝 更新日志

- **2025-10-30**: 初始版本，包含 7 个核心问题文档
- 计划添加：错误处理、JSON 序列化、解析优化等主题

## 📧 反馈

如有问题或建议，请：
- 提交 Issue
- 发送邮件
- 参与讨论

---

**记住**：错误是最好的老师。这些文档记录的"错误"都是宝贵的经验！

Happy Learning! 🚀

