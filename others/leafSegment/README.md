# Leaf-Segment Distributed ID Generator (Go Implementation)

## 1. Introduction

This project is a simplified Go implementation based on the **Leaf** algorithm (Segment Mode) open-sourced by Meituan-Dianping.

In distributed systems, generating globally unique, strictly increasing, and high-performance IDs is a common requirement. Traditional database auto-increment IDs (`auto_increment`) suffer from severe performance bottlenecks (lock contention) under high concurrency.

The core idea of **Leaf-Segment** is **"Batch Retrieval, Local Distribution"**.
Instead of requesting the database for every single ID, the Server requests a "Segment" (a range of IDs) from the database at once (e.g., `[1000, 2000]`) and loads it into memory. IDs are then distributed directly from memory via atomic operations until the segment is exhausted.

## 2. Core Design Philosophy

To achieve high performance and high availability, this implementation relies on two key mechanisms:

### 2.1 Segment Pattern

The database no longer records the "current ID" but rather the "current maximum ID (`max_id`)".
Every time the Server requests the DB, it executes `UPDATE max_id = max_id + step`.

  * **Step:** Determines the length of a segment (e.g., 1000).
  * **Effect:** A single database I/O operation can support the generation of 1000 IDs in memory. Database pressure is instantly reduced by a factor of 1000.

### 2.2 Double Buffer Optimization

If a single segment is used, the Server must synchronously request the DB when the current segment runs out. If the DB experiences network latency at that moment, requests will block, causing TP999 spikes.

The **Double Buffer** strategy solves this:

  * **Buffer 1 (Current):** The segment currently in use.
  * **Buffer 2 (Next):** The standby segment.
  * **Prefetching Logic:** When the `Current` segment is consumed to a remaining 10% \~ 20%, an asynchronous thread automatically pulls the next segment from the DB and places it into `Next`.
  * **Seamless Switching:** When `Current` is exhausted, the pointer switches to `Next` instantly via memory operations. The user perceives no DB I/O latency.

-----

## 3. Code Analysis

The code is divided into several core components:

### 3.1 `Segment`: The Data Model

This structure stores the ID range in memory.

```go
type Segment struct {
    Base   int64 // Start of the segment (exclusive)
    Max    int64 // End of the segment (inclusive)
    Step   int   // Length of the segment
    Cursor int64 // Current issuance cursor (Core field, accessed via atomic)
}
```

  * **Implementation Detail**: The `Remaining()` method uses `atomic.LoadInt64` to calculate the number of IDs left, which is used to trigger prefetching.

### 3.2 `LeafDAO`: Database Interaction

Responsible for "applying" for new segments from MySQL.

  * **Core SQL**:
    ```sql
    UPDATE leaf_alloc SET max_id = max_id + step WHERE biz_tag = ?
    ```
    This SQL atomically reserves a new range of IDs.
  * **FetchNextSegment**: Executes an Update followed by a Select within a transaction to ensure the obtained `max_id` and `step` are exclusive to the current thread.

### 3.3 `DoubleBuffer`: The Controller (Core Logic)

This is the brain of the algorithm, managing the `current` and `next` Segments.

  * **`NextID()` - Main Flow**:

    1.  Attempts to perform an atomic add (`atomic.AddInt64`) directly on the `current` Segment.
    2.  If successful and within bounds, checks if prefetching is needed (`CheckAndLoadNext`), then returns the ID.
    3.  **If Out of Bounds (Segment Exhausted):**
          * Acquire Lock (`db.mu.Lock()`).
          * Double Check if it is truly exhausted.
          * **Switch Buffer**: Point `current` to `next`, and clear `next`.
          * If `next` is not ready (extreme case), fall back to a synchronous DB request.

  * **`CheckAndLoadNext()` - Automatic Prefetching**:

      * **Threshold Check**: `if db.current.Remaining() > threshold { return }`. The code sets the threshold at 20% of the step.
      * **CAS Control**: Uses `atomic.CompareAndSwapInt32` to ensure only one background Goroutine executes the loading task, preventing duplicate DB requests.

### 3.4 `LeafServer`: Service Entry Point

  * Manages multiple business lines (`bizTag`). For example, `order-service` and `user-service` can have independent ID sequences.
  * Uses `map[string]*DoubleBuffer` to store buffers for different businesses and utilizes `sync.RWMutex` for concurrency safety.

-----

## 4. Quick Start

### 4.1 Prepare Database

You need a MySQL database with the following table structure (corresponding to the SQL in the code):

```sql
CREATE TABLE `leaf_alloc` (
  `biz_tag` varchar(128)  NOT NULL DEFAULT '',
  `max_id` bigint(20) NOT NULL DEFAULT '1',
  `step` int(11) NOT NULL,
  `description` varchar(256)  DEFAULT NULL,
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`biz_tag`)
) ENGINE=InnoDB;

-- Insert test data with a step of 1000
INSERT INTO `leaf_alloc` (`biz_tag`, `max_id`, `step`, `description`) 
VALUES ('order-service', 1, 1000, 'Test Order IDs');
```

### 4.2 Configuration & Run

1.  Modify the `dsn` variable in the `main` function with your database credentials:
    ```go
    dsn := "user:password@tcp(127.0.0.1:3306)/your_db?parseTime=true"
    ```
2.  Run the code:
    ```bash
    go run main.go
    ```

### 4.3 Expected Output

The program simulates 10 concurrent routines, each generating 500 IDs.

```text
2023/10/27 10:00:00 Leaf Server Started...
2023/10/27 10:00:00 Total time: 5.42ms, Finish generating 5000 IDs
```

You will notice that although 5000 IDs were generated, the `max_id` in the database only increased by 5000, and database interactions were minimal (depending on the Step size).

-----

## 5. Summary

This code demonstrates the standard industrial implementation paradigm for high-performance ID generators:

1.  **Reduced I/O**: Reduces DB operation frequency by a factor of `Step`.
2.  **Asynchronous Prefetching**: Uses Double Buffering to eliminate I/O wait time.
3.  **Concurrency Safety**: Cleverly balances `Atomic` (high-frequency ID issuance) and `Mutex` (low-frequency Segment switching) to achieve both performance and safety.