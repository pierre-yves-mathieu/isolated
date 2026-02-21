package proxy

import (
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// getFreePort returns an available port
func getFreePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}

// startEchoServer starts a simple echo server that returns what it receives
func startEchoServer(t *testing.T, port int) (net.Listener, chan struct{}) {
	t.Helper()
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					continue
				}
			}

			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c) // Echo back
			}(conn)
		}
	}()

	return listener, done
}

func TestProxy_Start(t *testing.T) {
	localPort := getFreePort(t)
	remotePort := getFreePort(t)

	proxy := New(localPort, "127.0.0.1", remotePort)
	if err := proxy.Start(); err != nil {
		t.Fatalf("failed to start proxy: %v", err)
	}
	defer proxy.Stop()

	// Verify listening
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", localPort), time.Second)
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	conn.Close()
}

func TestProxy_StartPortInUse(t *testing.T) {
	port := getFreePort(t)

	// Occupy the port
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	// Try to start proxy on same port
	proxy := New(port, "127.0.0.1", 8080)
	err = proxy.Start()
	if err == nil {
		proxy.Stop()
		t.Fatal("expected error when port is in use")
	}
}

func TestProxy_Stop(t *testing.T) {
	localPort := getFreePort(t)

	proxy := New(localPort, "127.0.0.1", 8080)
	if err := proxy.Start(); err != nil {
		t.Fatal(err)
	}

	proxy.Stop()

	// Port should be released
	time.Sleep(10 * time.Millisecond)
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		t.Fatalf("port should be released after stop: %v", err)
	}
	listener.Close()
}

func TestProxy_ForwardsData(t *testing.T) {
	localPort := getFreePort(t)
	remotePort := getFreePort(t)

	// Start echo server
	echoServer, done := startEchoServer(t, remotePort)
	defer func() {
		close(done)
		echoServer.Close()
	}()

	// Start proxy
	proxy := New(localPort, "127.0.0.1", remotePort)
	if err := proxy.Start(); err != nil {
		t.Fatal(err)
	}
	defer proxy.Stop()

	// Connect through proxy
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Send data
	testData := "Hello, World!"
	if _, err := conn.Write([]byte(testData)); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Read response
	buf := make([]byte, len(testData))
	conn.SetReadDeadline(time.Now().Add(time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	if string(buf[:n]) != testData {
		t.Errorf("expected %q, got %q", testData, string(buf[:n]))
	}
}

func TestProxy_BidirectionalData(t *testing.T) {
	localPort := getFreePort(t)
	remotePort := getFreePort(t)

	// Start echo server
	echoServer, done := startEchoServer(t, remotePort)
	defer func() {
		close(done)
		echoServer.Close()
	}()

	// Start proxy
	proxy := New(localPort, "127.0.0.1", remotePort)
	if err := proxy.Start(); err != nil {
		t.Fatal(err)
	}
	defer proxy.Stop()

	// Multiple round trips
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	for i := 0; i < 5; i++ {
		msg := fmt.Sprintf("Message %d", i)
		conn.Write([]byte(msg))

		buf := make([]byte, len(msg))
		conn.SetReadDeadline(time.Now().Add(time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			t.Fatalf("round %d: read error: %v", i, err)
		}
		if string(buf[:n]) != msg {
			t.Errorf("round %d: expected %q, got %q", i, msg, string(buf[:n]))
		}
	}
}

func TestProxy_MultipleConnections(t *testing.T) {
	localPort := getFreePort(t)
	remotePort := getFreePort(t)

	// Start echo server
	echoServer, done := startEchoServer(t, remotePort)
	defer func() {
		close(done)
		echoServer.Close()
	}()

	// Start proxy
	proxy := New(localPort, "127.0.0.1", remotePort)
	if err := proxy.Start(); err != nil {
		t.Fatal(err)
	}
	defer proxy.Stop()

	// Make multiple concurrent connections
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
			if err != nil {
				errors <- fmt.Errorf("conn %d: dial error: %v", id, err)
				return
			}
			defer conn.Close()

			msg := fmt.Sprintf("Connection %d", id)
			conn.Write([]byte(msg))

			buf := make([]byte, len(msg))
			conn.SetReadDeadline(time.Now().Add(time.Second))
			n, err := conn.Read(buf)
			if err != nil {
				errors <- fmt.Errorf("conn %d: read error: %v", id, err)
				return
			}
			if string(buf[:n]) != msg {
				errors <- fmt.Errorf("conn %d: expected %q, got %q", id, msg, string(buf[:n]))
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestProxy_RemoteUnavailable(t *testing.T) {
	localPort := getFreePort(t)
	remotePort := getFreePort(t) // No server listening

	proxy := New(localPort, "127.0.0.1", remotePort)
	if err := proxy.Start(); err != nil {
		t.Fatal(err)
	}
	defer proxy.Stop()

	// Connect to proxy
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Connection should close since remote is unavailable
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	buf := make([]byte, 1)
	_, err = conn.Read(buf)
	if err == nil {
		t.Error("expected error or EOF when remote is unavailable")
	}
}

func TestProxy_LargeData(t *testing.T) {
	localPort := getFreePort(t)
	remotePort := getFreePort(t)

	// Start echo server
	echoServer, done := startEchoServer(t, remotePort)
	defer func() {
		close(done)
		echoServer.Close()
	}()

	// Start proxy
	proxy := New(localPort, "127.0.0.1", remotePort)
	if err := proxy.Start(); err != nil {
		t.Fatal(err)
	}
	defer proxy.Stop()

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Send 1MB of data
	dataSize := 1024 * 1024
	data := make([]byte, dataSize)
	for i := range data {
		data[i] = byte(i % 256)
	}

	go func() {
		conn.Write(data)
	}()

	// Read all data back
	received := make([]byte, dataSize)
	totalRead := 0
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	for totalRead < dataSize {
		n, err := conn.Read(received[totalRead:])
		if err != nil {
			t.Fatalf("read error after %d bytes: %v", totalRead, err)
		}
		totalRead += n
	}

	// Verify data integrity
	for i := 0; i < dataSize; i++ {
		if received[i] != data[i] {
			t.Fatalf("data mismatch at byte %d: expected %d, got %d", i, data[i], received[i])
		}
	}
}

func TestManager_Add(t *testing.T) {
	localPort := getFreePort(t)

	manager := NewManager()
	defer manager.StopAll()

	if err := manager.Add(localPort, "127.0.0.1", 8080); err != nil {
		t.Fatalf("failed to add proxy: %v", err)
	}

	// Verify proxy is running
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", localPort), time.Second)
	if err != nil {
		t.Fatalf("proxy not listening: %v", err)
	}
	conn.Close()
}

func TestManager_AddMultiple(t *testing.T) {
	port1 := getFreePort(t)
	port2 := getFreePort(t)
	port3 := getFreePort(t)

	manager := NewManager()
	defer manager.StopAll()

	for _, port := range []int{port1, port2, port3} {
		if err := manager.Add(port, "127.0.0.1", 8080); err != nil {
			t.Fatalf("failed to add proxy for port %d: %v", port, err)
		}
	}

	// Verify all proxies are running
	for _, port := range []int{port1, port2, port3} {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
		if err != nil {
			t.Errorf("proxy on port %d not listening: %v", port, err)
			continue
		}
		conn.Close()
	}
}

func TestManager_StopAll(t *testing.T) {
	port1 := getFreePort(t)
	port2 := getFreePort(t)

	manager := NewManager()

	manager.Add(port1, "127.0.0.1", 8080)
	manager.Add(port2, "127.0.0.1", 8080)

	manager.StopAll()

	// Ports should be released
	time.Sleep(10 * time.Millisecond)

	for _, port := range []int{port1, port2} {
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			t.Errorf("port %d should be released: %v", port, err)
			continue
		}
		listener.Close()
	}
}

func TestManager_AddDuplicatePort(t *testing.T) {
	port := getFreePort(t)

	manager := NewManager()
	defer manager.StopAll()

	if err := manager.Add(port, "127.0.0.1", 8080); err != nil {
		t.Fatal(err)
	}

	// Try to add same port again
	err := manager.Add(port, "127.0.0.1", 8080)
	if err == nil {
		t.Error("expected error when adding duplicate port")
	}
}
