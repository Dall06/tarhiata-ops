package auditlog

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"
)

// AuditEntry representa un evento de auditoría individual.
type AuditEntry struct {
	ID           int       `json:"id,omitempty"`
	Action       string    `json:"action"`       // "DEPLOY", "EDIT", "DELETE", "PROVISION", "LINK", "UNLINK"
	ResourceType string    `json:"resourceType"` // "service", "database", "worker", "link"
	ResourceName string    `json:"resourceName"`
	Details      string    `json:"details"`
	Timestamp    time.Time `json:"timestamp"`
}

// Logger escribe entradas de auditoría de forma asíncrona y paralela en un archivo NDJSON.
type Logger struct {
	filePath string
	logChan  chan AuditEntry
	closeCh  chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// InitLogger inicializa un worker de fondo en paralelo para la captura de logs.
func InitLogger(filePath string) *Logger {
	once.Do(func() {
		l := &Logger{
			filePath: filePath,
			logChan:  make(chan AuditEntry, 1000),
			closeCh:  make(chan struct{}),
		}
		l.wg.Add(1)
		go l.worker()
		defaultLogger = l
	})
	return defaultLogger
}

// GetDefaultLogger obtiene el logger singleton configurado.
func GetDefaultLogger() *Logger {
	if defaultLogger == nil {
		path := "docs/logs/audit.log"
		if _, err := os.Stat("/opt/tarhiata"); err == nil {
			path = "/opt/tarhiata/audit.log"
		}
		// Ensure the local logs directory exists
		if err := os.MkdirAll("docs/logs", 0755); err == nil {
			// directory ensured
		}
		return InitLogger(path)
	}
	return defaultLogger
}

// worker corre en segundo plano escribiendo en audit.log sin bloquear peticiones HTTP.
func (l *Logger) worker() {
	defer l.wg.Done()

	for {
		select {
		case entry, ok := <-l.logChan:
			if !ok {
				return
			}
			l.writeToFile(entry)
		case <-l.closeCh:
			for len(l.logChan) > 0 {
				entry := <-l.logChan
				l.writeToFile(entry)
			}
			return
		}
	}
}

func (l *Logger) writeToFile(entry AuditEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	f, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	_, _ = f.WriteString(string(data) + "\n")
}

// Log envía un evento al canal paralelo para ser procesado de forma no bloqueante.
func (l *Logger) Log(entry AuditEntry) {
	select {
	case l.logChan <- entry:
	default:
		go l.writeToFile(entry)
	}
}

// ReadRecent lee las últimas entradas del archivo para la API / UI.
func (l *Logger) ReadRecent(limit int) ([]AuditEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	file, err := os.Open(l.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []AuditEntry{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			lines = append(lines, text)
		}
	}

	if limit <= 0 {
		limit = 100
	}

	var entries []AuditEntry
	start := 0
	if len(lines) > limit {
		start = len(lines) - limit
	}

	for i := len(lines) - 1; i >= start; i-- {
		var e AuditEntry
		if err := json.Unmarshal([]byte(lines[i]), &e); err == nil {
			entries = append(entries, e)
		}
	}

	return entries, nil
}

// Close cierra limpiamente el worker paralelo.
func (l *Logger) Close() {
	close(l.closeCh)
	l.wg.Wait()
}
