package proxy

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const (
	// MaxConnectionsPerProxy limits concurrent connections per proxy to prevent resource exhaustion
	MaxConnectionsPerProxy = 100
	// ConnectionTimeout is the idle timeout for proxied connections
	ConnectionTimeout = 30 * time.Second
	// DialTimeout is the timeout for establishing remote connections
	DialTimeout = 5 * time.Second
)

// Proxy represents a TCP proxy for a single port
type Proxy struct {
	LocalPort  int
	RemoteAddr string
	listener   net.Listener
	done       chan struct{}
	wg         sync.WaitGroup
	connSem    chan struct{} // Semaphore for limiting concurrent connections
}

// New creates a new proxy
func New(localPort int, remoteHost string, remotePort int) *Proxy {
	return &Proxy{
		LocalPort:  localPort,
		RemoteAddr: fmt.Sprintf("%s:%d", remoteHost, remotePort),
		done:       make(chan struct{}),
		connSem:    make(chan struct{}, MaxConnectionsPerProxy),
	}
}

// Start begins listening and proxying connections
func (p *Proxy) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", p.LocalPort))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", p.LocalPort, err)
	}
	p.listener = listener

	p.wg.Add(1)
	go p.acceptLoop()

	return nil
}

// Stop stops the proxy
func (p *Proxy) Stop() {
	close(p.done)
	if p.listener != nil {
		p.listener.Close()
	}
	p.wg.Wait()
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

		// Try to acquire semaphore (non-blocking)
		select {
		case p.connSem <- struct{}{}: // Acquired slot
			p.wg.Add(1)
			go p.handleConnection(conn)
		default:
			// At capacity - reject connection
			conn.Close()
		}
	}
}

func (p *Proxy) handleConnection(local net.Conn) {
	defer func() {
		local.Close()
		<-p.connSem // Release semaphore slot
		p.wg.Done()
	}()

	// Set deadline on local connection
	local.SetDeadline(time.Now().Add(ConnectionTimeout))

	// Dial remote with timeout
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
		// Half-close: signal we're done writing to remote
		if tc, ok := remote.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
		done <- struct{}{}
	}()

	go func() {
		io.Copy(local, remote)
		// Half-close: signal we're done writing to local
		if tc, ok := local.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
		done <- struct{}{}
	}()

	// Wait for BOTH directions to complete (prevents goroutine leak)
	<-done
	<-done
}

// Manager manages multiple proxies
type Manager struct {
	proxies []*Proxy
	mu      sync.Mutex
}

// NewManager creates a new proxy manager
func NewManager() *Manager {
	return &Manager{}
}

// Add adds a proxy for a port
func (m *Manager) Add(localPort int, remoteHost string, remotePort int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	proxy := New(localPort, remoteHost, remotePort)
	if err := proxy.Start(); err != nil {
		return err
	}

	m.proxies = append(m.proxies, proxy)
	return nil
}

// StopAll stops all proxies
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.proxies {
		p.Stop()
	}
	m.proxies = nil
}
