# Issue #5: No Connection Limits in TCP Proxy

## Severity: High
## Category: Resource Management / Denial of Service

---

## Problem Summary

The TCP proxy implementation spawns unbounded goroutines for each incoming connection without any tracking, limits, or backpressure mechanisms. This can lead to resource exhaustion under load or malicious traffic.

---

## Affected Code

**File:** `internal/proxy/proxy.go`

```go
func (p *Proxy) acceptLoop() {
    defer p.wg.Done()

    for {
        conn, err := p.listener.Accept()
        if err != nil {
            select {
            case <-p.done:
                return
            default:
                continue
            }
        }

        go p.handleConnection(conn)  // <-- UNBOUNDED GOROUTINE SPAWN
    }
}
```

**Also in `handleConnection`:**

```go
func (p *Proxy) handleConnection(local net.Conn) {
    defer local.Close()

    remote, err := net.Dial("tcp", p.RemoteAddr)
    if err != nil {
        return  // <-- SILENT FAILURE, NO LOGGING
    }
    defer remote.Close()

    done := make(chan struct{}, 2)

    go func() {           // <-- GOROUTINE #1
        io.Copy(remote, local)
        done <- struct{}{}
    }()

    go func() {           // <-- GOROUTINE #2
        io.Copy(local, remote)
        done <- struct{}{}
    }()

    <-done
}
```

---

## Why This Is a Problem

### 1. Resource Exhaustion Attack

An attacker (or even legitimate heavy load) can open thousands of connections:

```
Each connection spawns:
- 1 goroutine for handleConnection
- 2 goroutines for bidirectional io.Copy
- 2 open file descriptors (local + remote sockets)
- Memory for buffers in io.Copy

1000 connections = 3000+ goroutines + 2000+ file descriptors
```

### 2. No Visibility Into System State

There's no way to know:
- How many active connections exist
- Which connections are stale/slow
- Whether the system is approaching limits

### 3. Slowloris-Style Attacks

A client can open connections and send data very slowly, keeping goroutines alive indefinitely. There are no timeouts on the connections.

### 4. File Descriptor Exhaustion

Linux typically defaults to 1024 open file descriptors per process. Each proxied connection uses 2 FDs. After ~500 concurrent connections, new connections will fail.

### 5. Goroutine Leak on Partial Close

In `handleConnection`, only one direction finishing triggers return. The other goroutine may hang if the remote never closes:

```go
<-done  // Only waits for ONE direction to finish
// The other goroutine may still be blocked on io.Copy
```

---

## Demonstration

### Simulating the Issue

```bash
# Open many connections rapidly (requires netcat or similar)
for i in $(seq 1 2000); do
    nc localhost 8000 &
done

# The proxy will spawn 6000+ goroutines and likely crash or become unresponsive
```

### Observing Goroutine Growth

Add this to proxy.go temporarily:

```go
import "runtime"

func (p *Proxy) handleConnection(local net.Conn) {
    fmt.Printf("Active goroutines: %d\n", runtime.NumGoroutine())
    // ... rest of function
}
```

---

## Recommended Fix

### Option A: Semaphore-Based Connection Limiting

```go
package proxy

import (
    "fmt"
    "io"
    "net"
    "sync"
    "time"
)

const (
    MaxConnections    = 100              // Max concurrent connections per proxy
    ConnectionTimeout = 30 * time.Second // Idle timeout
    DialTimeout       = 5 * time.Second  // Remote connection timeout
)

type Proxy struct {
    LocalPort    int
    RemoteAddr   string
    listener     net.Listener
    done         chan struct{}
    wg           sync.WaitGroup

    // NEW: Connection management
    connSem      chan struct{}        // Semaphore for limiting connections
    activeConns  int64                // Counter for monitoring
    mu           sync.Mutex
}

func New(localPort int, remoteHost string, remotePort int) *Proxy {
    return &Proxy{
        LocalPort:  localPort,
        RemoteAddr: fmt.Sprintf("%s:%d", remoteHost, remotePort),
        done:       make(chan struct{}),
        connSem:    make(chan struct{}, MaxConnections), // Buffered channel as semaphore
    }
}

func (p *Proxy) acceptLoop() {
    defer p.wg.Done()

    for {
        conn, err := p.listener.Accept()
        if err != nil {
            select {
            case <-p.done:
                return
            default:
                continue
            }
        }

        // Try to acquire semaphore (non-blocking check first)
        select {
        case p.connSem <- struct{}{}: // Acquired
            p.wg.Add(1)
            go p.handleConnection(conn)
        default:
            // At capacity - reject connection gracefully
            conn.Close()
            // Optionally log: "connection rejected: at capacity"
        }
    }
}

func (p *Proxy) handleConnection(local net.Conn) {
    defer func() {
        local.Close()
        <-p.connSem // Release semaphore
        p.wg.Done()
    }()

    // Set deadline on local connection
    local.SetDeadline(time.Now().Add(ConnectionTimeout))

    // Dial with timeout
    dialer := net.Dialer{Timeout: DialTimeout}
    remote, err := dialer.Dial("tcp", p.RemoteAddr)
    if err != nil {
        return
    }
    defer remote.Close()

    // Set deadline on remote connection
    remote.SetDeadline(time.Now().Add(ConnectionTimeout))

    // Bidirectional copy with proper cleanup
    done := make(chan struct{}, 2)

    go func() {
        io.Copy(remote, local)
        remote.(*net.TCPConn).CloseWrite() // Half-close
        done <- struct{}{}
    }()

    go func() {
        io.Copy(local, remote)
        local.(*net.TCPConn).CloseWrite() // Half-close
        done <- struct{}{}
    }()

    // Wait for BOTH directions to complete
    <-done
    <-done
}
```

### Option B: Worker Pool Pattern

```go
type Proxy struct {
    // ... existing fields
    connQueue chan net.Conn
    workers   int
}

func (p *Proxy) Start() error {
    // ... listener setup

    p.connQueue = make(chan net.Conn, 100) // Buffered queue
    p.workers = 50 // Fixed worker pool

    // Start worker pool
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker()
    }

    p.wg.Add(1)
    go p.acceptLoop()

    return nil
}

func (p *Proxy) acceptLoop() {
    defer p.wg.Done()

    for {
        conn, err := p.listener.Accept()
        if err != nil {
            select {
            case <-p.done:
                close(p.connQueue) // Signal workers to stop
                return
            default:
                continue
            }
        }

        select {
        case p.connQueue <- conn:
            // Queued successfully
        default:
            // Queue full, reject
            conn.Close()
        }
    }
}

func (p *Proxy) worker() {
    defer p.wg.Done()

    for conn := range p.connQueue {
        p.handleConnection(conn)
    }
}
```

---

## Additional Improvements

### 1. Add Metrics/Observability

```go
type ProxyStats struct {
    ActiveConnections int64
    TotalConnections  int64
    RejectedConnections int64
    BytesTransferred  int64
}

func (p *Proxy) Stats() ProxyStats {
    return ProxyStats{
        ActiveConnections: atomic.LoadInt64(&p.activeConns),
        // ...
    }
}
```

### 2. Add Graceful Shutdown

```go
func (p *Proxy) Stop() {
    close(p.done)

    // Give existing connections time to finish
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    done := make(chan struct{})
    go func() {
        p.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        // Clean shutdown
    case <-shutdownCtx.Done():
        // Force close listener to break Accept()
        p.listener.Close()
    }
}
```

### 3. Add Connection Logging

```go
func (p *Proxy) handleConnection(local net.Conn) {
    remoteAddr := local.RemoteAddr().String()
    start := time.Now()

    defer func() {
        duration := time.Since(start)
        log.Printf("proxy: connection from %s closed after %v", remoteAddr, duration)
    }()

    // ... rest of handler
}
```

---

## Testing the Fix

```go
func TestProxy_ConnectionLimit(t *testing.T) {
    proxy := New(0, "localhost", 9999) // Port 0 = random available
    proxy.connSem = make(chan struct{}, 5) // Limit to 5 for test

    if err := proxy.Start(); err != nil {
        t.Fatal(err)
    }
    defer proxy.Stop()

    // Open 10 connections
    var conns []net.Conn
    for i := 0; i < 10; i++ {
        conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", proxy.LocalPort))
        if err == nil {
            conns = append(conns, conn)
        }
    }
    defer func() {
        for _, c := range conns {
            c.Close()
        }
    }()

    // Should only have 5 successful connections
    if len(conns) > 5 {
        t.Errorf("expected max 5 connections, got %d", len(conns))
    }
}
```

---

## References

- [Go Concurrency Patterns: Context](https://go.dev/blog/context)
- [Semaphore Pattern in Go](https://gobyexample.com/rate-limiting)
- [TCP Connection Handling Best Practices](https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/)
- [Slowloris Attack](https://en.wikipedia.org/wiki/Slowloris_(computer_security))
