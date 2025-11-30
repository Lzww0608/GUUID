# Leaf-Segment 模式分布式 ID 生成器 (Go 实现)

## 1. 简介

本项目是基于美团点评开源的 **Leaf** 算法（Segment 模式）的 Go 语言精简实现。

在分布式系统中，生成全局唯一、递增且高性能的 ID 是一个常见需求。传统的数据库自增 ID (`auto_increment`) 在高并发下存在严重的性能瓶颈（锁竞争）。

**Leaf-Segment** 方案的核心思想是：**“批量获取，本地分发”**。
不再每次获取 ID 都请求数据库，而是由 Server 从数据库一次性申请一个“号段”（Segment），例如 `[1000, 2000]`，加载到内存中。之后在内存中直接通过原子操作分发 ID，直到该号段用尽。

## 2. 核心设计理念

为了实现极高的性能和可用性，本代码主要实现了以下两个关键机制：

### 2.1 号段模式

数据库不再记录“当前 ID”，而是记录“当前最大 ID (`max_id`)”。
每次 Server 请求 DB，都执行 `UPDATE max_id = max_id + step`。

  * **Step (步长):** 决定了一个号段的长度（例如 1000）。
  * **效果:** 数据库的一次 I/O 操作，可以支撑 Server 端生成 1000 个 ID。数据库压力瞬间降低为原来的 1/1000。

### 2.2 双 Buffer 优化

如果仅使用单个号段，当号段用尽那一刻，Server 需要同步请求 DB 获取新号段。如果 DB 此时有网络抖动，会导致请求阻塞，出现 TP999 尖刺。

**双 Buffer** 策略是为了解决这个问题：

  * **Buffer 1 (Current):** 正在使用的号段。
  * **Buffer 2 (Next):** 备用号段。
  * **预加载逻辑:** 当 Current 号段使用到了 **10% \~ 20%** 的余量时，异步线程会自动去 DB 拉取下一个号段放入 Next。
  * **无缝切换:** 当 Current 用尽时，直接通过内存指针瞬间切换到 Next，用户感知不到 DB 的 I/O 耗时。

-----

## 3. 代码实现详解

代码主要分为以下几个核心部分：

### 3.1 `Segment`：号段模型

这是内存中存储 ID 范围的数据结构。

```go
type Segment struct {
    Base   int64 // 号段起始值 (不包含)
    Max    int64 // 号段最大值 (包含)
    Step   int   // 步长
    Cursor int64 // 当前发放到的游标 (核心字段，使用 atomic 操作)
}
```

  * **实现细节**: `Remaining()` 方法通过 `atomic.LoadInt64` 计算剩余 ID 数量，用于判断是否需要触发预加载。

### 3.2 `LeafDAO`：数据库交互

负责从 MySQL 中“申请”新的号段。

  * **核心 SQL**:
    ```sql
    UPDATE leaf_alloc SET max_id = max_id + step WHERE biz_tag = ?
    ```
    这条 SQL 原子性地占据了一个新的 ID 范围。
  * **FetchNextSegment**: 在一个事务中先 Update 再 Select，确保获取到的 `max_id` 和 `step` 是当前线程独占的。

### 3.3 `DoubleBuffer`：双缓冲控制器 (核心逻辑)

这是整个算法的大脑，管理着 `current` 和 `next` 两个 `Segment`。

  * **`NextID()` - 获取 ID 的主流程**:

    1.  尝试直接在 `current` Segment 上进行原子加 (`atomic.AddInt64`).
    2.  如果成功且未超限，检查是否需要异步加载下一个号段 (`CheckAndLoadNext`)，然后返回 ID。
    3.  **如果超限 (当前号段用完了):**
          * 加锁 (`db.mu.Lock()`)。
          * 再次检查是否真的用完了 (Double Check)。
          * **切换 Buffer**: 将 `current` 指向 `next`，清空 `next`。
          * 如果 `next` 还没准备好（极端情况），则不得不降级为同步请求 DB。

  * **`CheckAndLoadNext()` - 自动预加载**:

      * **阈值判断**: `if db.current.Remaining() > threshold { return }`。代码中设定阈值为步长的 20%。
      * **CAS 控制**: 使用 `atomic.CompareAndSwapInt32` 确保只有一个后台 Goroutine 在执行加载任务，避免并发重复请求 DB。

### 3.4 `LeafServer`：对外服务入口

  * 管理多个业务线 (`bizTag`)。例如，`order-service` 和 `user-service` 可以拥有独立的 ID 序列。
  * 使用 `map[string]*DoubleBuffer` 存储不同业务的缓冲区，并利用 `sync.RWMutex` 保证并发安全。

-----

## 4. 快速开始

### 4.1 准备数据库

你需要一个 MySQL 数据库，并建立如下表结构（对应代码中的 SQL）：

```sql
CREATE TABLE `leaf_alloc` (
  `biz_tag` varchar(128)  NOT NULL DEFAULT '',
  `max_id` bigint(20) NOT NULL DEFAULT '1',
  `step` int(11) NOT NULL,
  `description` varchar(256)  DEFAULT NULL,
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`biz_tag`)
) ENGINE=InnoDB;

-- 插入一条测试数据，步长设为 1000
INSERT INTO `leaf_alloc` (`biz_tag`, `max_id`, `step`, `description`) 
VALUES ('order-service', 1, 1000, 'Test Order IDs');
```

### 4.2 配置运行

1.  修改 `main` 函数中的 `dsn` 变量，填入你的数据库账号密码：
    ```go
    dsn := "user:password@tcp(127.0.0.1:3306)/your_db?parseTime=true"
    ```
2.  运行代码：
    ```bash
    go run main.go
    ```

### 4.3 预期输出

程序将模拟 10 个并发协程，每个生成 500 个 ID。

```text
2023/10/27 10:00:00 Leaf Server Started...
2023/10/27 10:00:00 Total time: 5.42ms, Finish generating 5000 IDs
```

你会发现，虽然生成了 5000 个 ID，但数据库中 `max_id` 只增加了 5000，且数据库交互次数极少（取决于 Step 大小）。

-----

## 5. 总结

这份代码展示了高性能 ID 生成器的标准工业级实现范式：

1.  **减少 I/O**: 将 DB 操作频率降低 Step 倍。
2.  **异步预取**: 利用 Double Buffer 消除 I/O 等待耗时。
3.  **并发安全**: 巧妙结合 `Atomic` (高频发号) 和 `Mutex` (低频切换 Segment)，在性能和安全性之间取得平衡。