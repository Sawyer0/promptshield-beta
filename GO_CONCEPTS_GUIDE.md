# Go Programming Concepts in PromptShield

A comprehensive guide to understanding the Go concepts used in PromptShield, with comparisons to JavaScript for developers coming from a MERN stack background.

---

## Table of Contents

1. [Goroutines (Concurrency)](#1-goroutines-concurrency)
2. [Channels (Communication)](#2-channels-communication)
3. [Interfaces (Duck Typing)](#3-interfaces-duck-typing)
4. [Pointers (Memory Management)](#4-pointers-memory-management)
5. [Struct Tags (Metadata)](#5-struct-tags-metadata)
6. [Defer (Cleanup)](#6-defer-cleanup)
7. [Error Handling (Explicit)](#7-error-handling-explicit)
8. [Context (Request Lifecycle)](#8-context-request-lifecycle)
9. [Select Statement (Multiplexing)](#9-select-statement-multiplexing)
10. [Sync Primitives (Mutexes)](#10-sync-primitives-mutexes)

---

## 1. Goroutines (Concurrency)

### What Are They?

Goroutines are lightweight threads managed by the Go runtime. They enable true parallelism across multiple CPU cores.

**Cost:**
- OS Thread: ~1-2 MB stack
- Goroutine: ~2 KB stack (grows dynamically)
- You can run 100,000+ goroutines easily

### In PromptShield

**Location:** `internal/infrastructure/messaging/nats/subscriber.go`

```go
func NewSubscriber(...) (*Subscriber, error) {
    s := &Subscriber{...}
    
    // Launch background goroutines
    go s.initializeConsumerGroup()  // Runs independently
    go s.recoverPending()            // Runs independently
    
    return s, nil  // Returns immediately!
}
```

**What happens:**

```
T=0ms:   Main thread creates Subscriber
T=1ms:   Launch Goroutine 1 (initializeConsumerGroup)
T=2ms:   Launch Goroutine 2 (recoverPending)
T=3ms:   Main thread returns ✓

T=10ms:  Goroutine 1 tries Redis connection...
T=50ms:  Goroutine 1 retries (Redis down)...
T=200ms: Goroutine 1 succeeds ✓

T=60s:   Goroutine 2 wakes up (1 minute timer)
T=60.01s: Goroutine 2 recovers pending messages...
```

### JavaScript Comparison

```javascript
// JavaScript - event loop (single-threaded)
async function newSubscriber() {
    const s = new Subscriber();
    
    // These run on event loop, not parallel
    s.initializeConsumerGroup();  // Queued
    s.recoverPending();            // Queued
    
    return s;  // Returns immediately
}
```

**Key Difference:**
- **JavaScript**: Concurrency via event loop (interleaved on single thread)
- **Go**: True parallelism (runs on multiple CPU cores simultaneously)

### Common Patterns

**Pattern 1: Fire and Forget**
```go
go doSomething()  // Launch and don't wait
```

**Pattern 2: Wait for Completion**
```go
var wg sync.WaitGroup

for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        processJob(id)
    }(i)
}

wg.Wait()  // Wait for all 10 goroutines
```

**JavaScript equivalent:**
```javascript
const promises = [];
for (let i = 0; i < 10; i++) {
    promises.push(processJob(i));
}
await Promise.all(promises);
```

---

## 2. Channels (Communication)

### What Are They?

Channels are typed conduits for sending and receiving values between goroutines. They provide synchronization and communication.

**Types:**
- **Unbuffered**: `make(chan T)` - Sender blocks until receiver reads
- **Buffered**: `make(chan T, 10)` - Sender blocks only when buffer is full

### In PromptShield

**Location:** `internal/infrastructure/messaging/nats/subscriber.go`

```go
type Subscriber struct {
    done chan struct{}  // Signal channel (no data)
}

// Create channel
s := &Subscriber{
    done: make(chan struct{}),
}

// Start method - runs in goroutine
func (s *Subscriber) Start(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-s.done:      // Block until signal
            return nil
        default:
            // Do work...
        }
    }
}

// Stop method - send shutdown signal
func (s *Subscriber) Stop() {
    close(s.done)  // Signal all waiting goroutines
}
```

**What happens:**

```
Goroutine 1 (Start):
├─ Loop forever
├─ Check s.done → BLOCKS (waiting for signal)
└─ ...waiting...

Main Thread:
├─ User wants shutdown
├─ Call s.Stop()
└─ close(s.done) → SIGNAL SENT!

Goroutine 1 (Start):
├─ s.done receives signal!
└─ return nil ✓
```

### JavaScript Comparison

```javascript
class Subscriber {
    constructor() {
        this.stopped = false;
    }
    
    async start() {
        while (!this.stopped) {  // Poll flag
            // Do work...
            await sleep(100);  // Check periodically
        }
    }
    
    stop() {
        this.stopped = true;  // Set flag
    }
}
```

**Key Difference:**
- **JavaScript**: Poll a boolean flag (wastes CPU)
- **Go**: Block on channel (zero CPU until signal)

### Channel Operations

```go
// Send
ch <- value

// Receive
value := <-ch

// Close (signal completion)
close(ch)

// Check if closed
value, ok := <-ch
if !ok {
    // Channel closed
}
```

---

## 3. Interfaces (Duck Typing)

### What Are They?

Interfaces define behavior (methods) without implementation. Types automatically satisfy interfaces if they have the required methods.

**No `implements` keyword needed!**

### In PromptShield

**Location:** `internal/domain/repositories.go`

```go
// Define interface
type TenantRepository interface {
    Create(ctx context.Context, tenant *Tenant) error
    Get(ctx context.Context, id uuid.UUID) (*Tenant, error)
    List(ctx context.Context, offset, limit int) ([]*Tenant, int, error)
}
```

**Location:** `internal/infrastructure/persistence/postgres/tenants.go`

```go
// Implement interface (implicitly)
type pgTenantRepo struct {
    db *Pool
}

func (r *pgTenantRepo) Create(ctx context.Context, tenant *Tenant) error {
    // PostgreSQL implementation
}

func (r *pgTenantRepo) Get(ctx context.Context, id uuid.UUID) (*Tenant, error) {
    // PostgreSQL implementation
}

func (r *pgTenantRepo) List(ctx context.Context, offset, limit int) ([]*Tenant, int, error) {
    // PostgreSQL implementation
}

// pgTenantRepo automatically implements TenantRepository!
```

**Usage:**

```go
// Can swap implementations easily
var repo TenantRepository

// Use PostgreSQL
repo = &pgTenantRepo{db: pgPool}

// Or use in-memory (for testing)
repo = &memoryTenantRepo{data: make(map[uuid.UUID]*Tenant)}

// Code using repo doesn't care which implementation!
tenant, err := repo.Get(ctx, id)
```

### JavaScript Comparison

```javascript
// JavaScript - duck typing (implicit)
class PostgresTenantRepo {
    create(tenant) { /* ... */ }
    get(id) { /* ... */ }
}

class MemoryTenantRepo {
    create(tenant) { /* ... */ }
    get(id) { /* ... */ }
}

// Both work the same way
const repo = new PostgresTenantRepo();
await repo.get(id);
```

**TypeScript:**

```typescript
interface TenantRepository {
    create(tenant: Tenant): Promise<void>;
    get(id: string): Promise<Tenant>;
}

class PostgresTenantRepo implements TenantRepository {
    // Must implement all methods
}
```

**Key Difference:**
- **JavaScript**: Duck typing (if it has the methods, it works)
- **TypeScript**: Explicit `implements` keyword
- **Go**: Implicit interfaces (no keyword, automatic)

### Benefits

1. **Easy testing**: Swap real DB with mock
2. **Flexibility**: Change implementations without changing callers
3. **Decoupling**: Domain layer doesn't depend on infrastructure

---

## 4. Pointers (Memory Management)

### What Are They?

Pointers hold memory addresses. They let you pass references instead of copies.

**Syntax:**
- `*T` = pointer to type T
- `&x` = address of x
- `*p` = value at pointer p

### In PromptShield

**Location:** `internal/infrastructure/persistence/postgres/tenants.go`

```go
// Pointer receiver (can modify struct)
func (r *pgTenantRepo) Create(ctx context.Context, tenant *Tenant) error {
    //  ↑ pointer receiver                              ↑ pointer parameter
    
    tenant.ID = uuid.New()      // Modifies original
    tenant.CreatedAt = time.Now()  // Modifies original
    
    return r.db.Insert(tenant)
}

// Value receiver (cannot modify struct)
func (r pgTenantRepo) GetName() string {
    // ↑ value receiver (copy)
    return r.name  // Read-only
}
```

**Pointer vs Value:**

```go
// With pointer - modifies original
func updateTenant(t *Tenant) {
    t.Name = "Updated"  // Original is modified
}

tenant := &Tenant{Name: "Original"}
updateTenant(tenant)
fmt.Println(tenant.Name)  // "Updated"

// With value - modifies copy
func updateTenantCopy(t Tenant) {
    t.Name = "Updated"  // Only copy is modified
}

tenant := Tenant{Name: "Original"}
updateTenantCopy(tenant)
fmt.Println(tenant.Name)  // "Original" (unchanged)
```

### JavaScript Comparison

```javascript
// JavaScript - objects always passed by reference
function updateTenant(tenant) {
    tenant.name = "Updated";  // Modifies original
}

const tenant = { name: "Original" };
updateTenant(tenant);
console.log(tenant.name);  // "Updated"

// No way to pass by value (except manual copy)
function updateTenantCopy(tenant) {
    const copy = { ...tenant };
    copy.name = "Updated";  // Only copy modified
}
```

**Key Difference:**
- **JavaScript**: Objects always by reference, primitives by value
- **Go**: Choose pointer (`*T`) or value (`T`) for any type

### When to Use Pointers

**Use pointers when:**
1. You want to modify the original
2. The struct is large (avoid copying)
3. You need nil (pointers can be nil, values cannot)

**Use values when:**
1. The type is small (int, bool, small structs)
2. You want immutability
3. You don't need nil

---

## 5. Struct Tags (Metadata)

### What Are They?

Struct tags are metadata attached to struct fields. They're used by libraries for serialization, validation, etc.

### In PromptShield

**Location:** `internal/domain/models.go`

```go
type Tenant struct {
    ID        uuid.UUID              `json:"id" db:"id"`
    Name      string                 `json:"name" db:"name" validate:"required"`
    Status    TenantStatus           `json:"status" db:"status"`
    Metadata  map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
    CreatedAt time.Time              `json:"created_at" db:"created_at"`
    UpdatedAt time.Time              `json:"updated_at" db:"updated_at"`
}
```

**Tag meanings:**
- `json:"id"` - JSON field name
- `json:"metadata,omitempty"` - Omit if empty
- `db:"id"` - Database column name
- `validate:"required"` - Validation rule

**Usage:**

```go
// JSON serialization
tenant := &Tenant{
    ID:   uuid.New(),
    Name: "Acme Corp",
}

data, _ := json.Marshal(tenant)
// {"id":"...","name":"Acme Corp","status":"","created_at":"..."}
//  ↑ Uses json:"id" tag

// Database mapping
db.Insert("tenants", tenant)
// INSERT INTO tenants (id, name, status, ...) VALUES (...)
//                      ↑ Uses db:"id" tag
```

### JavaScript Comparison

```javascript
// JavaScript - manual mapping
class Tenant {
    constructor(id, name) {
        this.id = id;
        this.name = name;
    }
    
    toJSON() {
        return {
            id: this.id,
            name: this.name,
            created_at: this.createdAt
        };
    }
    
    toDBRow() {
        return {
            id: this.id,
            name: this.name,
            created_at: this.createdAt
        };
    }
}
```

**TypeScript with decorators:**

```typescript
class Tenant {
    @JsonProperty('id')
    @Column('id')
    id: string;
    
    @JsonProperty('name')
    @Column('name')
    name: string;
}
```

**Key Difference:**
- **JavaScript**: Manual serialization methods
- **TypeScript**: Decorators (similar to Go tags)
- **Go**: Struct tags (built into language)

---

## 6. Defer (Cleanup)

### What Is It?

`defer` schedules a function call to run when the surrounding function returns. Used for cleanup (close files, unlock mutexes, etc.).

**Execution order:** LIFO (Last In, First Out)

### In PromptShield

**Location:** `internal/infrastructure/messaging/nats/subscriber.go`

```go
func (s *Subscriber) initializeConsumerGroup() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()  // ← Always called when function returns
    
    // Do work...
    if err != nil {
        return  // cancel() called here
    }
    
    // More work...
    return  // cancel() called here too
}
```

**Multiple defers:**

```go
func processFile(path string) error {
    file, err := os.Open(path)
    if err != nil {
        return err
    }
    defer file.Close()  // Defer 1
    
    mutex.Lock()
    defer mutex.Unlock()  // Defer 2
    
    // Do work...
    
    return nil
    // Execution order:
    // 1. mutex.Unlock() (last defer)
    // 2. file.Close()   (first defer)
}
```

### JavaScript Comparison

```javascript
// JavaScript - try/finally
async function initializeConsumerGroup() {
    const controller = new AbortController();
    try {
        // Do work...
        if (error) {
            throw error;
        }
        // More work...
    } finally {
        controller.abort();  // Always runs
    }
}
```

**Key Difference:**
- **JavaScript**: `finally` block
- **Go**: `defer` statement (cleaner, LIFO order)

### Common Uses

```go
// 1. Unlock mutex
mutex.Lock()
defer mutex.Unlock()

// 2. Close file
file, _ := os.Open("file.txt")
defer file.Close()

// 3. Cancel context
ctx, cancel := context.WithTimeout(...)
defer cancel()

// 4. Recover from panic
defer func() {
    if r := recover(); r != nil {
        log.Error("Recovered from panic", r)
    }
}()
```

---

## 7. Error Handling (Explicit)

### What Is It?

Go functions return errors explicitly. Callers must check them (compiler doesn't force it, but convention does).

### In PromptShield

**Location:** `internal/infrastructure/persistence/postgres/tenants.go`

```go
func (r *pgTenantRepo) Create(ctx context.Context, tenant *Tenant) error {
    _, err := r.db.Exec(ctx, query, tenant.ID, tenant.Name)
    if err != nil {
        return fmt.Errorf("create tenant: %w", err)  // Wrap error
    }
    return nil  // Success
}

// Caller must check
tenant, err := repo.Create(ctx, tenant)
if err != nil {
    log.Error("failed to create tenant", err)
    return err  // Propagate or handle
}
```

**Error wrapping:**

```go
// Wrap with context
return fmt.Errorf("create tenant: %w", err)

// Unwrap later
if errors.Is(err, sql.ErrNoRows) {
    // Handle specific error
}
```

### JavaScript Comparison

```javascript
// JavaScript - try/catch
async function create(tenant) {
    try {
        await db.exec(query, tenant.id, tenant.name);
    } catch (err) {
        throw new Error(`create tenant: ${err.message}`);
    }
}

// Caller
try {
    await repo.create(tenant);
} catch (err) {
    console.error('failed to create tenant', err);
    throw err;
}
```

**Key Difference:**
- **JavaScript**: Exceptions (can be ignored accidentally)
- **Go**: Explicit returns (visible in function signature)

### Error Patterns

**Pattern 1: Check and return**
```go
if err != nil {
    return err
}
```

**Pattern 2: Check and wrap**
```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

**Pattern 3: Check specific error**
```go
if errors.Is(err, sql.ErrNoRows) {
    return ErrNotFound
}
```

**Pattern 4: Ignore (rare)**
```go
_ = file.Close()  // Explicitly ignore
```

---

## 8. Context (Request Lifecycle)

### What Is It?

`context.Context` carries deadlines, cancellation signals, and request-scoped values across API boundaries.

**Standard practice:** First parameter of every function.

### In PromptShield

**Location:** `internal/infrastructure/persistence/postgres/tenants.go`

```go
func (r *pgTenantRepo) Create(ctx context.Context, tenant *Tenant) error {
    //                         ↑ context passed everywhere
    
    // Check if cancelled
    select {
    case <-ctx.Done():
        return ctx.Err()  // Request cancelled
    default:
        // Continue
    }
    
    // Pass to database (respects timeout)
    _, err := r.db.ExecContext(ctx, query, args...)
    return err
}
```

**Creating contexts:**

```go
// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// With deadline
ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
defer cancel()

// With cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// With value (request-scoped data)
ctx = context.WithValue(ctx, "tenant_id", "acme-123")
```

### JavaScript Comparison

```javascript
// JavaScript - AbortController (newer)
const controller = new AbortController();

async function create(tenant, signal) {
    if (signal.aborted) {
        throw new Error('Request cancelled');
    }
    
    await db.insert(tenant, { signal });
}

// Usage
await create(tenant, controller.signal);
controller.abort();  // Cancel
```

**Key Difference:**
- **JavaScript**: AbortController (not universal, newer API)
- **Go**: Context (standard, everywhere)

### Common Uses

```go
// 1. Timeouts
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

// 2. Cancellation
ctx, cancel := context.WithCancel(ctx)
go func() {
    // Do work...
    cancel()  // Signal cancellation
}()

// 3. Request-scoped values
tenantID := ctx.Value("tenant_id").(string)

// 4. Propagation
func handler(ctx context.Context) {
    // Pass context down the call stack
    result, err := service.Process(ctx, data)
}
```

---

## 9. Select Statement (Multiplexing)

### What Is It?

`select` lets a goroutine wait on multiple channel operations. It's like a switch statement for channels.

### In PromptShield

**Location:** `internal/infrastructure/messaging/nats/subscriber.go`

```go
func (s *Subscriber) Start(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()  // Context cancelled
        case <-s.done:
            return nil  // Shutdown signal
        default:
            // Do work (non-blocking)
        }
    }
}
```

**Select rules:**
1. If multiple cases are ready, one is chosen **randomly**
2. If no cases are ready and there's `default`, run default
3. If no cases are ready and no `default`, **block** until one is ready

### JavaScript Comparison

```javascript
// JavaScript - Promise.race (one-time)
await Promise.race([
    contextPromise,   // Context cancelled
    shutdownPromise,  // Shutdown signal
    workPromise       // Do work
]);

// Or polling (inefficient)
while (true) {
    if (ctx.cancelled) return;
    if (this.stopped) return;
    await doWork();
}
```

**Key Difference:**
- **JavaScript**: Promise.race (one-time) or polling
- **Go**: Select in loop (continuous, efficient)

### Select Patterns

**Pattern 1: Timeout**
```go
select {
case result := <-ch:
    return result
case <-time.After(5 * time.Second):
    return ErrTimeout
}
```

**Pattern 2: Non-blocking receive**
```go
select {
case msg := <-ch:
    process(msg)
default:
    // Channel empty, continue
}
```

**Pattern 3: Non-blocking send**
```go
select {
case ch <- value:
    // Sent successfully
default:
    // Channel full, drop message
}
```

---

## 10. Sync Primitives (Mutexes)

### What Are They?

Mutexes (mutual exclusion locks) protect shared data from concurrent access. Required in Go because of true parallelism.

**Types:**
- `sync.Mutex` - Exclusive lock
- `sync.RWMutex` - Read-write lock (multiple readers OR one writer)

### In PromptShield

**Location:** `internal/infrastructure/messaging/nats/subscriber.go`

```go
type Subscriber struct {
    stateMu     sync.Mutex  // Protects tenantState map
    tenantState map[string]*tenantState
}

func (s *Subscriber) getState(tenantID string) *tenantState {
    s.stateMu.Lock()         // Acquire lock
    defer s.stateMu.Unlock()  // Release lock
    
    return s.tenantState[tenantID]
}

func (s *Subscriber) setState(tenantID string, state *tenantState) {
    s.stateMu.Lock()
    defer s.stateMu.Unlock()
    
    s.tenantState[tenantID] = state
}
```

**Read-Write Mutex:**

```go
type Cache struct {
    mu    sync.RWMutex
    data  map[string]string
}

func (c *Cache) Get(key string) string {
    c.mu.RLock()         // Read lock (multiple readers OK)
    defer c.mu.RUnlock()
    return c.data[key]
}

func (c *Cache) Set(key, value string) {
    c.mu.Lock()          // Write lock (exclusive)
    defer c.mu.Unlock()
    c.data[key] = value
}
```

### JavaScript Comparison

```javascript
// JavaScript - no mutexes needed (single-threaded)
class Subscriber {
    constructor() {
        this.tenantState = new Map();
    }
    
    getState(tenantID) {
        return this.tenantState.get(tenantID);  // No locking
    }
    
    setState(tenantID, state) {
        this.tenantState.set(tenantID, state);  // No locking
    }
}
```

**Key Difference:**
- **JavaScript**: Single-threaded, no race conditions
- **Go**: Multi-threaded, must protect shared data

### Race Conditions

**Without mutex (WRONG):**

```go
// Goroutine 1
count := counter
count++
counter = count

// Goroutine 2 (simultaneously)
count := counter  // Reads same value!
count++
counter = count   // Overwrites Goroutine 1's write!

// Result: Lost update!
```

**With mutex (CORRECT):**

```go
// Goroutine 1
mu.Lock()
counter++
mu.Unlock()

// Goroutine 2 (waits)
mu.Lock()  // Blocks until Goroutine 1 unlocks
counter++
mu.Unlock()

// Result: Both updates applied ✓
```

### Mutex Patterns

**Pattern 1: Defer unlock**
```go
mu.Lock()
defer mu.Unlock()
// Work...
```

**Pattern 2: Read-write split**
```go
// Many readers
mu.RLock()
defer mu.RUnlock()
value := data[key]

// One writer
mu.Lock()
defer mu.Unlock()
data[key] = value
```

**Pattern 3: Try lock (non-blocking)**
```go
if mu.TryLock() {
    defer mu.Unlock()
    // Got lock
} else {
    // Couldn't get lock, skip
}
```

---

## Summary: Go vs JavaScript

| Concept | JavaScript | Go |
|---------|-----------|-----|
| **Concurrency** | Event loop, async/await | Goroutines, channels |
| **Parallelism** | Web Workers (limited) | Native (multiple cores) |
| **Type System** | Dynamic (or TypeScript) | Static, compiled |
| **Error Handling** | try/catch (can ignore) | Explicit returns (forced by convention) |
| **Memory** | Garbage collected | Garbage collected + pointers |
| **Interfaces** | Duck typing | Implicit interfaces |
| **Null Safety** | null/undefined chaos | Explicit nil checks |
| **Concurrency Safety** | Not needed (single-thread) | Mutexes, channels required |
| **Cleanup** | finally blocks | defer statements |
| **Request Context** | AbortController (newer) | context.Context (standard) |

---

## Key Takeaways

1. **Goroutines enable true parallelism** - Use them for background work, not just async I/O
2. **Channels are for communication** - Prefer channels over shared memory when possible
3. **Interfaces are implicit** - No `implements` keyword, automatic satisfaction
4. **Pointers give control** - Choose reference or value semantics explicitly
5. **Defer is for cleanup** - Always runs, LIFO order, cleaner than finally
6. **Errors are values** - Check explicitly, wrap with context
7. **Context is everywhere** - First parameter, carries cancellation and values
8. **Select multiplexes channels** - Wait on multiple operations efficiently
9. **Mutexes prevent races** - Required for shared data in concurrent code
10. **Struct tags are metadata** - Used by libraries for serialization, validation

---

## Further Reading

- [Effective Go](https://golang.org/doc/effective_go.html) - Official style guide
- [Go by Example](https://gobyexample.com/) - Practical examples
- [Go Concurrency Patterns](https://www.youtube.com/watch?v=f6kdp27TYZs) - Rob Pike's talk
- [PromptShield Architecture](./Architecture.md) - How these concepts are used

---

**Author:** Dawan Sawyer
