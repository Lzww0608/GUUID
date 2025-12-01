
# Leaf-Snowflake (Go Implementation)

这是一个生产级可用的分布式唯一 ID 生成系统（Distributed Unique ID Generator）。

它基于 Twitter 的 **Snowflake 算法**，利用 **Zookeeper** 作为注册中心来自动分配和管理 WorkerID，解决了传统雪花算法需要人工手动指定 WorkerID 的痛点，并实现了完善的时钟回拨保护机制。

## 📖 目录

1.  [背景知识：雪花算法](https://www.google.com/search?q=%231-%E8%83%8C%E6%99%AF%E7%9F%A5%E8%AF%86%E9%9B%AA%E8%8A%B1%E7%AE%97%E6%B3%95)
2.  [核心特性](https://www.google.com/search?q=%232-%E6%A0%B8%E5%BF%83%E7%89%B9%E6%80%A7)
3.  [代码深度解析](https://www.google.com/search?q=%233-%E4%BB%A3%E7%A0%81%E6%B7%B1%E5%BA%A6%E8%A7%A3%E6%9E%90)
      - [ID 结构定义](https://www.google.com/search?q=%2331-id-%E7%BB%93%E6%9E%84%E5%AE%9A%E4%B9%89)
      - [WorkerID 的自动注册与恢复](https://www.google.com/search?q=%2332-workerid-%E7%9A%84%E8%87%AA%E5%8A%A8%E6%B3%A8%E5%86%8C%E4%B8%8E%E6%81%A2%E5%A4%8D)
      - [核心生成逻辑 (NextID)](https://www.google.com/search?q=%2333-%E6%A0%B8%E5%BF%83%E7%94%9F%E6%88%90%E9%80%BB%E8%BE%91-nextid)
      - [时钟回拨保护](https://www.google.com/search?q=%2334-%E6%97%B6%E9%92%9F%E5%9B%9E%E6%8B%A8%E4%BF%9D%E6%8A%A4)
4.  [快速开始](https://www.google.com/search?q=%234-%E5%BF%AB%E9%80%9F%E5%BC%80%E5%A7%8B)

-----

## 1\. 背景知识：雪花算法

Snowflake 算法生成的 ID 是一个 64 位的整数（`int64`），结构如下（本代码中的配置）：

  * **1 bit**：不使用（符号位，始终为 0，保证 ID 为正数）。
  * **41 bits**：毫秒级时间戳（可以使用约 69 年）。
  * **10 bits**：WorkerID（工作机器 ID）。支持部署 1024 个节点（$2^{10}$）。
  * **12 bits**：序列号。同一毫秒内最多生成 4096 个 ID（$2^{12}$）。

**优势**：

  * 生成的 ID 按时间大致有序（利于数据库索引）。
  * 不依赖数据库，完全在内存中生成，高性能。
  * 分布式环境均不重复。

-----

## 2\. 核心特性

这份代码相比原始的雪花算法，增加了以下工程化特性：

1.  **Zookeeper 集成**：通过 ZK 的持久节点记录 WorkerID，节点重启后能保持 ID 不变。
2.  **双重容灾**：
      * 如果 ZK 挂了，尝试从本地文件系统（Local Cache）恢复。
      * 如果本地文件也没了，使用备用算法分配 ID。
3.  **时钟回拨保护**：
      * 启动时：检查当前时间是否小于上次记录的时间（在 ZK 或缓存中）。
      * 运行时：如果检测到时间回退，拒绝生成 ID 或等待时间追平。

-----

## 3\. 代码深度解析

### 3.1 ID 结构定义

代码顶部定义了算法的“骨架”：

```go
const (
    Epoch int64 = 1672531200000 // 起始时间 (2023-01-01)
    
    WorkerIdBits = 10 // 机器码位数
    SequenceBits = 12 // 序列号位数
    // ... 位移量计算 ...
)
```

**解析**：

  * `Epoch` 是自定义的纪元时间。41 位时间戳记录的是 `当前时间 - Epoch` 的差值。
  * 通过位移（Shift）操作，将时间、机器码、序列号拼接到一个 64 位的整数中。

### 3.2 WorkerID 的自动注册与恢复

这是本实现的亮点。函数 `registerOrRecover` 负责确定当前节点该用哪个 `WorkerID`。

**流程逻辑**：

1.  **构建 ZK 路径**：`/leaf_snowflake/{serviceName}/{port}`。
2.  **检查 ZK**：
      * 如果节点存在：读取数据，恢复 `WorkerID`。同时检查 `LastTime`（上次心跳时间）是否大于当前时间（检查是否发生了时钟回拨）。
      * 如果节点不存在：说明是新机器，或者 ZK 数据丢失。
3.  **降级策略**：
      * 尝试读取本地文件 `.leaf_cache_{port}`。
      * 如果文件也没有，则通过 `port % 1024` 简单计算一个 ID（兜底）。
4.  **注册/更新**：将确定的 `WorkerID` 和当前时间写入 ZK，并保存到本地文件。

**代码片段**：

```go
// 检查系统时钟是否回退（关键安全检查）
if currentTime < myNodeInfo.LastTime {
    return 0, fmt.Errorf("clock moved backwards...")
}
```

### 3.3 核心生成逻辑 (NextID)

`NextID` 方法是高并发下的核心瓶颈，使用了 `sync.Mutex` 保证线程安全。

**逻辑流程**：

1.  **获取当前毫秒**。
2.  **同一毫秒内**：
      * `sequence + 1`。
      * 如果序列号溢出（超过 4095），自旋等待下一毫秒。
3.  **新的一毫秒**：
      * `sequence` 重置为 0。
4.  **位运算拼接**：
    ```go
    id := ((now - Epoch) << TimestampShift) |
          (d.workerID << WorkIdShift) |
          d.sequence
    ```

### 3.4 时钟回拨保护与心跳

雪花算法最怕服务器时间被回调（例如 NTP 自动校准），这会导致 ID 重复。

**运行时保护**：
在 `NextID` 中：

```go
if now < d.lastTime {
    offset := d.lastTime - now
    if offset <= 5 {
        // 如果回拨时间很短，暂停一会等时间追上来
        time.Sleep(...)
    } else {
        // 回拨严重，直接报错，拒绝服务，保护数据一致性
        return 0, fmt.Errorf("clock moved backwards too much")
    }
}
```

**后台心跳 (`scheduledUploadTime`)**：
每隔 3 秒，后台 Goroutine 会将当前时间戳更新到 Zookeeper 和本地文件。这确保了如果服务重启，我们知道服务上次存活的时间点，从而在启动阶段就能检测到时间是否异常。

-----

## 4\. 快速开始

### 前置条件

你需要一个运行中的 Zookeeper 实例。

```bash
# 使用 Docker 启动本地 Zookeeper
docker run --name some-zookeeper -p 2181:2181 -d zookeeper
```

### 运行代码

1.  保存代码为 `main.go`。
2.  安装依赖：
    ```bash
    go get github.com/go-zookeeper/zk
    ```
3.  运行程序：
    ```bash
    go run main.go
    ```

### 预期输出

```text
snowflake driver initialized with workerID: 888  <-- 自动分配的 ID
Start generating IDs...
Done.
```

同时你会发现目录下生成了一个 `.leaf_cache_8080` 的文件，这就是本地缓存。

-----

## 5\. 常见问题 (FAQ)

**Q: 为什么需要本地文件缓存？**
A: 如果 Zookeeper 集群宕机，应用重启时无法连接 ZK 获取 WorkerID。此时本地文件缓存充当“逃生舱”，允许服务沿用旧的 WorkerID 继续启动。

**Q: SequenceMask 的作用是什么？**
A: `SequenceMask` (值为 4095, 二进制 111111111111) 用于位与运算：`(sequence + 1) & SequenceMask`。这是一种高效的取模操作，当 sequence 达到 4096 时，结果会自动变为 0。

-----