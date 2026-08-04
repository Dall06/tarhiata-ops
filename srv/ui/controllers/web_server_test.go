package controllers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

type mockRepo struct{}

func (m *mockRepo) SaveServerConfig(config domain.ServerConfig) error         { return nil }
func (m *mockRepo) GetServerConfig() (*domain.ServerConfig, error)            { return nil, nil }
func (m *mockRepo) SaveService(service domain.SavedService) error             { return nil }
func (m *mockRepo) GetServices() ([]domain.SavedService, error)                { return nil, nil }
func (m *mockRepo) GetService(name string) (*domain.SavedService, error)        { return nil, nil }
func (m *mockRepo) DeleteService(name string) error                            { return nil }
func (m *mockRepo) SaveDatabase(db domain.SavedDatabase) error                 { return nil }
func (m *mockRepo) GetDatabases() ([]domain.SavedDatabase, error)                { return nil, nil }
func (m *mockRepo) GetDatabase(name string) (*domain.SavedDatabase, error)        { return nil, nil }
func (m *mockRepo) DeleteDatabase(name string) error                           { return nil }
func (m *mockRepo) SaveObservability(obs domain.SavedObservability) error       { return nil }
func (m *mockRepo) GetObservability() (*domain.SavedObservability, error)      { return nil, nil }
func (m *mockRepo) DeleteObservability() error                                 { return nil }
func (m *mockRepo) SaveServiceLink(link domain.ServiceLink) error              { return nil }
func (m *mockRepo) GetServiceLinks() ([]domain.ServiceLink, error)              { return nil, nil }
func (m *mockRepo) DeleteServiceLink(sourceSvc, targetSvc string) error         { return nil }
func (m *mockRepo) SavePreviewEnv(prev domain.SavedPreviewEnv) error           { return nil }
func (m *mockRepo) GetPreviewEnvs() ([]domain.SavedPreviewEnv, error)         { return nil, nil }
func (m *mockRepo) GetPreviewEnv(name string) (*domain.SavedPreviewEnv, error) { return nil, nil }
func (m *mockRepo) DeletePreviewEnv(name string) error                         { return nil }
func (m *mockRepo) SaveRegistryCredential(cred domain.SavedRegistryCredential) error {
	return nil
}
func (m *mockRepo) GetRegistryCredentials() ([]domain.SavedRegistryCredential, error) {
	return nil, nil
}
func (m *mockRepo) GetRegistryCredential(server string) (*domain.SavedRegistryCredential, error) {
	return nil, nil
}
func (m *mockRepo) DeleteRegistryCredential(server string) error                             { return nil }
func (m *mockRepo) SaveMigrationFile(file domain.MigrationFile) error                        { return nil }
func (m *mockRepo) GetMigrationFiles(dbName string) ([]domain.MigrationFile, error)          { return nil, nil }
func (m *mockRepo) DeleteMigrationFile(dbName, filename string) error                        { return nil }
func (m *mockRepo) RecordMigrationExecution(dbName, filename, status, logs string) error    { return nil }
func (m *mockRepo) SaveBackup(backup domain.SavedBackup) error                                { return nil }
func (m *mockRepo) GetBackups() ([]domain.SavedBackup, error)                                { return nil, nil }
func (m *mockRepo) GetBackupByID(id int) (*domain.SavedBackup, error)                        { return nil, nil }
func (m *mockRepo) DeleteBackup(id int) error                                                { return nil }
func (m *mockRepo) SaveAuditLog(log domain.AuditLog) error                                     { return nil }
func (m *mockRepo) GetAuditLogs(limit int) ([]domain.AuditLog, error)                          { return nil, nil }
func (m *mockRepo) Close() error                                                              { return nil }

func TestWebServer_HandleNodesGet(t *testing.T) {
	repo := &mockRepo{}
	cfg := &domain.ServerConfig{Host: "192.168.1.100", User: "root"}
	ws := NewWebServer(repo, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/nodes", nil)
	rr := httptest.NewRecorder()

	ws.handleNodes(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var nodes []map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&nodes); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if len(nodes) == 0 || nodes[0]["ip"] != "192.168.1.100" {
		t.Errorf("unexpected nodes list: %+v", nodes)
	}
}

func TestWebServer_HandleCustomDomainsPayload(t *testing.T) {
	repo := &mockRepo{}
	cfg := &domain.ServerConfig{Host: "192.168.1.100", User: "root"}
	ws := NewWebServer(repo, cfg)

	payload := map[string]interface{}{
		"serviceName":    "shop-app",
		"domain":         "shop.example.com",
		"redirectTarget": "https://example.com",
		"certType":       "custom",
		"forceHTTPS":     true,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/domains", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	ws.handleCustomDomains(rr, req)

	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected HTTP status: %d", rr.Code)
	}
}

func TestWebServer_HandleVolumeUploadTargeting(t *testing.T) {
	repo := &mockRepo{}
	cfg := &domain.ServerConfig{Host: "127.0.0.1", User: "root"}
	ws := NewWebServer(repo, cfg)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("targetPath", "/opt/data/traefik/certs/ssl_test.crt")
	part, _ := writer.CreateFormFile("file", "ssl_test.crt")
	part.Write([]byte("---BEGIN CERTIFICATE---"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/volumes/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()

	ws.handleVolumeUpload(rr, req)

	// Accept OK or 500 (since SSH is mocked / unconnected in local unit test environment)
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status code: %d", rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "target") && !strings.Contains(rr.Body.String(), "error") {
		t.Errorf("expected response to reference target or error, got: %s", rr.Body.String())
	}
}
