package sshclient

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// DialFunc define la firma para crear una nueva conexión SSH.
type DialFunc func(host, user, privateKeyPath string, port int) (*Client, error)

func defaultDial(host, user, privateKeyPath string, port int) (*Client, error) {
	c := New()
	if err := c.Connect(host, user, privateKeyPath, port); err != nil {
		return nil, err
	}
	return c, nil
}

// pooledEntry representa un cliente activo dentro del pool con sus metadatos.
type pooledEntry struct {
	client   *Client
	lastUsed time.Time
	inUse    int
}

// Pool administra un conjunto de conexiones SSH activas reutilizables con limpieza de clientes ociosos.
type Pool struct {
	mu              sync.Mutex
	clients         map[string]*pooledEntry
	idleTimeout     time.Duration
	cleanupInterval time.Duration
	dialFn          DialFunc
	stop            chan struct{}
}

// GlobalPool es el pool compartido por defecto en toda la aplicación.
var GlobalPool = NewPool(60*time.Second, 15*time.Second)

// PoolKey genera un identificador unívoco para una tupla de conexión SSH.
func PoolKey(host, user, privateKeyPath string, port int) string {
	return fmt.Sprintf("%s@%s:%d:%s", user, host, port, privateKeyPath)
}

// NewPool instancia un pool de conexiones con los timeouts especificados.
func NewPool(idleTimeout, cleanupInterval time.Duration) *Pool {
	if idleTimeout <= 0 {
		idleTimeout = 60 * time.Second
	}
	if cleanupInterval <= 0 {
		cleanupInterval = 15 * time.Second
	}
	p := &Pool{
		clients:         make(map[string]*pooledEntry),
		idleTimeout:     idleTimeout,
		cleanupInterval: cleanupInterval,
		dialFn:          defaultDial,
		stop:            make(chan struct{}),
	}
	p.startCleaner()
	return p
}

// SetDialer permite sobrescribir la función de conexión (ideal para tests y mocks).
func (p *Pool) SetDialer(fn DialFunc) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.dialFn = fn
}

// Get obtiene una conexión activa del pool o inicializa una nueva si no existe o fue cerrada.
func (p *Pool) Get(host, user, privateKeyPath string, port int) (*Client, error) {
	key := PoolKey(host, user, privateKeyPath, port)

	p.mu.Lock()
	if entry, exists := p.clients[key]; exists {
		if entry.client != nil && entry.client.CheckConnection() {
			entry.lastUsed = time.Now()
			entry.inUse++
			p.mu.Unlock()
			return entry.client, nil
		}
		if entry.client != nil {
			if err := entry.client.Close(); err != nil {
				slog.Debug("falló cierre de cliente ssh inactivo o muerto en pool", "key", key, "error", err)
			}
		}
		delete(p.clients, key)
	}
	dial := p.dialFn
	p.mu.Unlock()

	client, err := dial(host, user, privateKeyPath, port)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if existing, exists := p.clients[key]; exists && existing.client != nil && existing.client.CheckConnection() {
		if err := client.Close(); err != nil {
			slog.Debug("falló cierre de cliente ssh redundante", "key", key, "error", err)
		}
		existing.lastUsed = time.Now()
		existing.inUse++
		return existing.client, nil
	}

	p.clients[key] = &pooledEntry{
		client:   client,
		lastUsed: time.Now(),
		inUse:    1,
	}
	return client, nil
}

// Release marca una conexión como liberada por su llamador, actualizando el timestamp de último uso.
func (p *Pool) Release(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry, exists := p.clients[key]
	if !exists {
		return
	}
	if entry.inUse > 0 {
		entry.inUse--
	}
	entry.lastUsed = time.Now()
}

// Invalidate cierra e invalida de forma inmediata una conexión del pool (por ejemplo tras un error de red).
func (p *Pool) Invalidate(key string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry, exists := p.clients[key]
	if !exists {
		return nil
	}
	delete(p.clients, key)
	if entry.client != nil {
		return entry.client.Close()
	}
	return nil
}

// Count retorna el total de conexiones registradas actualmente en el pool.
func (p *Pool) Count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.clients)
}

// CloseAll cierra todas las conexiones activas registradas en el pool.
func (p *Pool) CloseAll() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error
	for key, entry := range p.clients {
		if entry.client != nil {
			if err := entry.client.Close(); err != nil {
				slog.Debug("falló cierre de cliente en pool.CloseAll", "key", key, "error", err)
				if firstErr == nil {
					firstErr = err
				}
			}
		}
		delete(p.clients, key)
	}
	return firstErr
}

// Stop detiene el ticker de limpieza y cierra todas las conexiones.
func (p *Pool) Stop() {
	close(p.stop)
	if err := p.CloseAll(); err != nil {
		slog.Debug("error cerrando conexiones al detener el pool", "error", err)
	}
}

func (p *Pool) startCleaner() {
	ticker := time.NewTicker(p.cleanupInterval)
	go func() {
		for {
			select {
			case <-p.stop:
				ticker.Stop()
				return
			case now := <-ticker.C:
				p.cleanup(now)
			}
		}
	}()
}

func (p *Pool) cleanup(now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for key, entry := range p.clients {
		if entry.inUse <= 0 && now.Sub(entry.lastUsed) > p.idleTimeout {
			if entry.client != nil {
				if err := entry.client.Close(); err != nil {
					slog.Debug("falló cierre de cliente idle en pool", "key", key, "error", err)
				}
			}
			delete(p.clients, key)
		}
	}
}
