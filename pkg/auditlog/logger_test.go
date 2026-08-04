package auditlog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParallelAuditLogger(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_audit.log")

	logger := InitLogger(logFile)

	// Probar escrituras concurrentes paralelas
	for i := 0; i < 20; i++ {
		go logger.Log(AuditEntry{
			Action:       "DEPLOY",
			ResourceType: "service",
			ResourceName: "api-test",
			Details:      "Paralell test log entry",
			Timestamp:    time.Now(),
		})
	}

	// Pequeña espera para canal asíncrono
	time.Sleep(100 * time.Millisecond)

	entries, err := logger.ReadRecent(50)
	if err != nil {
		t.Fatalf("Error leyendo logs: %v", err)
	}

	if len(entries) == 0 {
		t.Fatalf("Se esperaban entradas grabadas en paralelo, pero la lista está vacía")
	}

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Fatalf("El archivo de log '%s' no fue creado", logFile)
	}
}
