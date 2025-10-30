# GUUID 快速上手指南

## 5 分钟快速开始

### 1. 安装

```bash
go get github.com/lab2439/guuid
```

### 2. 第一个程序

创建 `main.go`:

```go
package main

import (
    "fmt"
    "log"
    "github.com/lab2439/guuid"
)

func main() {
    // 生成一个新的 UUIDv7
    id, err := guuid.New()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("UUID:", id.String())
    fmt.Println("时间戳:", id.Timestamp(), "ms")
    fmt.Println("时间:", id.Time())
}
```

运行：
```bash
go run main.go
```

输出示例：
```
UUID: 01899c3a-6b3c-7f9a-8123-456789abcdef
时间戳: 1698765432100 ms
时间: 2025-10-30 20:30:32.1 +0800 CST
```

### 3. 常用场景

#### 场景 1: 数据库主键

```go
type User struct {
    ID        guuid.UUID `json:"id" db:"id"`
    Username  string     `json:"username" db:"username"`
    Email     string     `json:"email" db:"email"`
    CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

func CreateUser(username, email string) (*User, error) {
    user := &User{
        ID:        guuid.Must(guuid.New()),
        Username:  username,
        Email:     email,
        CreatedAt: time.Now(),
    }
    
    // 插入数据库
    _, err := db.Exec(
        "INSERT INTO users (id, username, email, created_at) VALUES (?, ?, ?, ?)",
        user.ID, user.Username, user.Email, user.CreatedAt,
    )
    
    return user, err
}
```

#### 场景 2: 批量生成（高性能）

```go
func GenerateBatch(count int) ([]guuid.UUID, error) {
    gen := guuid.NewGenerator()
    uuids := make([]guuid.UUID, count)
    
    for i := 0; i < count; i++ {
        uuid, err := gen.New()
        if err != nil {
            return nil, err
        }
        uuids[i] = uuid
    }
    
    return uuids, nil
}

// 使用
ids, _ := GenerateBatch(10000) // 生成 1 万个 UUID
```

#### 场景 3: 分布式系统

```go
// 服务 A
func ServiceA() {
    requestID, _ := guuid.New()
    
    // 传递给其他服务
    resp, _ := http.Post(
        "http://service-b/api",
        "application/json",
        body,
    )
}

// 服务 B
func ServiceB(w http.ResponseWriter, r *http.Request) {
    requestIDStr := r.Header.Get("X-Request-ID")
    requestID, _ := guuid.Parse(requestIDStr)
    
    // 使用相同的 request ID 进行日志追踪
    log.Printf("[%s] Processing request", requestID)
}
```

#### 场景 4: 编码转换

```go
id, _ := guuid.New()

// 多种编码格式
canonical := id.String()                    // f47ac10b-58cc-4372-a567-0e02b2c3d479
hex := id.EncodeToHex()                     // f47ac10b58cc4372a5670e02b2c3d479
base64 := id.EncodeToBase64()               // 9HrBC1jMQ3KlZw4CssP0eQ
bytes := id.Bytes()                         // []byte{0xf4, 0x7a, ...}

// 解析
id1, _ := guuid.Parse(canonical)
id2, _ := guuid.DecodeFromHex(hex)
id3, _ := guuid.DecodeFromBase64(base64)
id4, _ := guuid.FromBytes(bytes)
```

#### 场景 5: JSON 序列化

```go
type Event struct {
    ID        guuid.UUID `json:"id"`
    Type      string     `json:"type"`
    Timestamp int64      `json:"timestamp"`
}

event := Event{
    ID:        guuid.Must(guuid.New()),
    Type:      "user.created",
    Timestamp: time.Now().Unix(),
}

// 自动序列化为 JSON
data, _ := json.Marshal(event)
fmt.Println(string(data))
// {"id":"01899c3a-6b3c-7f9a-8123-456789abcdef","type":"user.created","timestamp":1698765432}

// 自动反序列化
var decoded Event
json.Unmarshal(data, &decoded)
```

### 4. 性能优化技巧

#### 技巧 1: 复用 Generator

```go
// ❌ 不推荐：每次创建新的 Generator
for i := 0; i < 10000; i++ {
    gen := guuid.NewGenerator()  // 每次都创建
    id, _ := gen.New()
}

// ✅ 推荐：复用 Generator
gen := guuid.NewGenerator()
for i := 0; i < 10000; i++ {
    id, _ := gen.New()
}
```

#### 技巧 2: 并发生成

```go
func GenerateConcurrent(count int, workers int) []guuid.UUID {
    gen := guuid.NewGenerator()
    results := make(chan guuid.UUID, count)
    perWorker := count / workers
    
    var wg sync.WaitGroup
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < perWorker; j++ {
                id, _ := gen.New()
                results <- id
            }
        }()
    }
    
    go func() {
        wg.Wait()
        close(results)
    }()
    
    uuids := make([]guuid.UUID, 0, count)
    for id := range results {
        uuids = append(uuids, id)
    }
    
    return uuids
}
```

#### 技巧 3: 预分配内存

```go
// ✅ 预分配切片容量
uuids := make([]guuid.UUID, 0, 10000)
gen := guuid.NewGenerator()

for i := 0; i < 10000; i++ {
    id, _ := gen.New()
    uuids = append(uuids, id)
}
```

### 5. 常见问题

#### Q: UUIDv7 和 UUIDv4 有什么区别？

**A**: 
- **UUIDv4**: 完全随机，无序
- **UUIDv7**: 时间排序，前 48 位是时间戳

优势：
- ✅ 数据库索引友好（B-tree 性能更好）
- ✅ 自然按时间排序
- ✅ 可以提取时间信息

#### Q: 如何保证唯一性？

**A**: UUIDv7 的唯一性通过三个机制保证：
1. 48 位时间戳（毫秒级）
2. 12 位单调计数器（同一毫秒内递增）
3. 62 位随机数据

碰撞概率：约 2^-74，实际可忽略不计。

#### Q: 是否需要担心时钟回拨？

**A**: 不需要。Generator 内部使用单调计数器，即使系统时钟回拨，生成的 UUID 仍然保持递增。

#### Q: 性能如何？

**A**: 非常高：
- 单线程：每秒 ~220 万个 UUID
- 并发：线程安全，无性能损失
- 内存：每个 UUID 仅 16 字节

#### Q: 可以用于分布式系统吗？

**A**: 可以！UUIDv7 不需要中心协调，每个节点独立生成，无需配置节点 ID。

### 6. 数据库使用示例

#### MySQL

```sql
CREATE TABLE users (
    id BINARY(16) PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_id (id)  -- 时间排序的索引
);
```

```go
// 插入
id := guuid.Must(guuid.New())
db.Exec("INSERT INTO users (id, username, email) VALUES (?, ?, ?)", 
    id, username, email)

// 查询
var user User
db.QueryRow("SELECT id, username, email FROM users WHERE id = ?", id).
    Scan(&user.ID, &user.Username, &user.Email)
```

#### PostgreSQL

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

```go
// 直接使用，PostgreSQL 原生支持 UUID
id := guuid.Must(guuid.New())
db.Exec("INSERT INTO users (id, username, email) VALUES ($1, $2, $3)", 
    id, username, email)
```

### 7. 测试

#### 测试中使用固定的随机源

```go
import "bytes"

func TestWithDeterministicUUID(t *testing.T) {
    // 使用固定种子的随机源
    seed := bytes.NewReader(make([]byte, 1024))
    gen := guuid.NewGeneratorWithReader(seed)
    
    id1, _ := gen.New()
    id2, _ := gen.New()
    
    // 可重现的测试
    assert.NotEqual(t, id1, id2)
}
```

### 8. 命令行工具

#### 生成 UUID

```bash
# 创建 tools/uuid-gen/main.go
package main

import (
    "fmt"
    "os"
    "strconv"
    "github.com/lab2439/guuid"
)

func main() {
    count := 1
    if len(os.Args) > 1 {
        count, _ = strconv.Atoi(os.Args[1])
    }
    
    for i := 0; i < count; i++ {
        id, _ := guuid.New()
        fmt.Println(id)
    }
}
```

使用：
```bash
go run tools/uuid-gen/main.go 10  # 生成 10 个 UUID
```

### 9. 更多示例

查看 `examples/` 目录：
- `examples/basic/` - 基础功能演示
- `examples/performance/` - 性能测试
- `examples/database/` - 数据库集成

运行示例：
```bash
cd examples/basic && go run main.go
cd examples/performance && go run main.go
cd examples/database && go run main.go
```

### 10. 下一步

- 📖 阅读 [README.md](README.md) 了解项目详情
- 🏗️ 阅读 [ARCHITECTURE.md](ARCHITECTURE.md) 了解架构设计
- 🤝 阅读 [CONTRIBUTING.md](CONTRIBUTING.md) 参与贡献
- 📊 查看基准测试：`make bench`
- 🧪 运行测试：`make test`

## 获取帮助

- 📧 提交 Issue: https://github.com/lab2439/guuid/issues
- 📚 查看文档: https://pkg.go.dev/github.com/lab2439/guuid
- ⭐ Star 项目: https://github.com/lab2439/guuid

---

**Happy Coding! 🚀**

