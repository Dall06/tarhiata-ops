package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

type mockRepo struct{}

func (m *mockRepo) SaveServerConfig(config domain.ServerConfig) error         { return nil }
func (m *mockRepo) GetServerConfig() (*domain.ServerConfig, error)            { return nil, nil }
func (m *mockRepo) GetAllServerConfigs() ([]domain.ServerConfig, error)       { return nil, nil }
func (m *mockRepo) GetServerConfigByName(name string) (*domain.ServerConfig, error) { return nil, nil }
func (m *mockRepo) SetActiveServerConfig(name string) error                   { return nil }
func (m *mockRepo) DeleteServerConfig(name string) error                      { return nil }
func (m *mockRepo) SaveService(service domain.SavedService) error             { return nil }
func (m *mockRepo) GetServices(serverName string) ([]domain.SavedService, error) { return nil, nil }
func (m *mockRepo) GetService(name, serverName string) (*domain.SavedService, error) { return nil, nil }
func (m *mockRepo) DeleteService(name, serverName string) error                { return nil }
func (m *mockRepo) SaveDatabase(db domain.SavedDatabase) error                 { return nil }
func (m *mockRepo) GetDatabases(serverName string) ([]domain.SavedDatabase, error) { return nil, nil }
func (m *mockRepo) GetDatabase(name, serverName string) (*domain.SavedDatabase, error) { return nil, nil }
func (m *mockRepo) DeleteDatabase(name, serverName string) error               { return nil }
func (m *mockRepo) SaveObservability(obs domain.SavedObservability) error       { return nil }
func (m *mockRepo) GetObservability() (*domain.SavedObservability, error)      { return nil, nil }
func (m *mockRepo) DeleteObservability() error                                 { return nil }
func (m *mockRepo) SaveServiceLink(link domain.ServiceLink) error              { return nil }
func (m *mockRepo) GetServiceLinks(serverName string) ([]domain.ServiceLink, error) { return nil, nil }
func (m *mockRepo) DeleteServiceLink(sourceSvc, targetSvc, serverName string) error { return nil }
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
func (m *mockRepo) SaveAlertSettings(settings domain.AlertSettings) error                       { return nil }
func (m *mockRepo) GetAlertSettings() (*domain.AlertSettings, error)                           { return &domain.AlertSettings{Enabled: false}, nil }
func (m *mockRepo) SaveDeploymentRecord(record domain.DeploymentRecord) error                   { return nil }
func (m *mockRepo) GetDeploymentHistory(serviceName string, limit int) ([]domain.DeploymentRecord, error) { return nil, nil }
func (m *mockRepo) GetDeploymentRecordByID(id int) (*domain.DeploymentRecord, error)           { return nil, nil }
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
	body, errMarshal := json.Marshal(payload)
	if errMarshal != nil {
		t.Fatalf("falló json.Marshal: %v", errMarshal)
	}

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
	if err := writer.WriteField("targetPath", "/opt/data/traefik/certs/ssl_test.crt"); err != nil {
		t.Fatalf("failed to write field: %v", err)
	}
	part, errPart := writer.CreateFormFile("file", "ssl_test.crt")
	if errPart != nil {
		t.Fatalf("failed to create form file: %v", errPart)
	}
	if _, err := part.Write([]byte("---BEGIN CERTIFICATE---")); err != nil {
		t.Fatalf("failed to write part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

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

func TestWebServer_HandleConnect(t *testing.T) {
	repo := &mockRepo{}
	cfg := &domain.ServerConfig{
		Host: "localhost",
	}
	ws := NewWebServer(repo, cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/connect", strings.NewReader(`{"host":"localhost"}`))
	rr := httptest.NewRecorder()

	ws.handleConnect(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got: %d", rr.Code)
	}

	var result domain.ConnectionResult
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !result.Connected {
		t.Errorf("expected Connected to be true for localhost, got false: %s", result.Message)
	}
	if !result.IsLocal {
		t.Errorf("expected IsLocal to be true")
	}
}

func TestWebServer_HandleProvisionServer(t *testing.T) {
	repo := &mockRepo{}
	cfg := &domain.ServerConfig{
		Host: "localhost",
	}
	ws := NewWebServer(repo, cfg)

	t.Run("Method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/servers/provision", nil)
		rr := httptest.NewRecorder()
		ws.handleProvisionServer(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", rr.Code)
		}
	})

	t.Run("Missing token returns bad request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/servers/provision", strings.NewReader(`{"name":"test-srv","apiToken":""}`))
		rr := httptest.NewRecorder()
		ws.handleProvisionServer(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})
}

func TestWebServer_HandleTerminalExec(t *testing.T) {
	repo := &mockRepo{}
	cfg := &domain.ServerConfig{
		Name:          "local",
		Host:          "localhost",
		CloudProvider: "local",
	}
	ws := NewWebServer(repo, cfg)

	t.Run("Executes safe command on localhost", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/servers/terminal", strings.NewReader(`{"command":"echo 'hello tarhiata'"}`))
		rr := httptest.NewRecorder()
		ws.handleTerminalExec(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var res map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		out, ok := res["output"].(string)
		if !ok || !strings.Contains(out, "hello tarhiata") {
			t.Errorf("unexpected output: %v", res)
		}
	})

	t.Run("Blocks dangerous commands", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/servers/terminal", strings.NewReader(`{"command":"rm -rf /"}`))
		rr := httptest.NewRecorder()
		ws.handleTerminalExec(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Errorf("expected status 403 Forbidden, got %d", rr.Code)
		}
	})
}

func TestWebServer_HandleHostEndpoints(t *testing.T) {
	repo := &mockRepo{}
	cfg := &domain.ServerConfig{
		Name:          "local",
		Host:          "localhost",
		CloudProvider: "local",
	}
	ws := NewWebServer(repo, cfg)

	t.Run("Host inspect on localhost", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/host/inspect", nil)
		rr := httptest.NewRecorder()
		ws.handleHostInspect(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var res domain.HostInspection
		if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode HostInspection: %v", err)
		}
		if res.ServerName != "local" {
			t.Errorf("expected serverName 'local', got '%s'", res.ServerName)
		}
	})

	t.Run("Host metrics on localhost", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/host/metrics", nil)
		rr := httptest.NewRecorder()
		ws.handleHostMetrics(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
		bodyStr := rr.Body.String()
		var res domain.HostMetrics
		if err := json.Unmarshal([]byte(bodyStr), &res); err != nil {
			t.Fatalf("failed to decode HostMetrics: %v", err)
		}
		if res.CPUCores <= 0 {
			t.Errorf("expected CPUCores > 0, got %d", res.CPUCores)
		}
	})

	t.Run("Host services on localhost", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/host/services", nil)
		rr := httptest.NewRecorder()
		ws.handleHostServices(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var res []domain.HostSystemService
		if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode HostSystemService list: %v", err)
		}
		if len(res) == 0 {
			t.Logf("no services found or non-fatal empty list on test host")
		}
	})

	t.Run("Host devices on localhost", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/host/devices", nil)
		rr := httptest.NewRecorder()
		ws.handleHostDevices(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var res domain.HostDevices
		if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode HostDevices: %v", err)
		}
	})

	t.Run("Swarm status on localhost", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/swarm/status", nil)
		rr := httptest.NewRecorder()
		ws.handleSwarmStatus(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var res domain.SwarmStatus
		if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode SwarmStatus: %v", err)
		}
	})
}

func TestWebServer_HandleServiceRestart(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		body           string
		cfg            *domain.ServerConfig
		expectedStatus int
	}{
		{
			name:           "Method Not Allowed (GET)",
			method:         http.MethodGet,
			url:            "/api/services/restart?name=my-app",
			body:           "",
			cfg:            &domain.ServerConfig{Host: "1.2.3.4"},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Missing Name Parameter",
			method:         http.MethodPost,
			url:            "/api/services/restart",
			body:           `{}`,
			cfg:            &domain.ServerConfig{Host: "1.2.3.4"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid Service Name Characters",
			method:         http.MethodPost,
			url:            "/api/services/restart",
			body:           `{"name":"app; rm -rf /"}`,
			cfg:            &domain.ServerConfig{Host: "1.2.3.4"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing VPS Configuration",
			method:         http.MethodPost,
			url:            "/api/services/restart",
			body:           `{"name":"valid-app"}`,
			cfg:            nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepo{}
			ws := NewWebServer(repo, tc.cfg)

			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
			} else {
				req = httptest.NewRequest(tc.method, tc.url, nil)
			}

			rr := httptest.NewRecorder()
			ws.handleServiceRestart(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("[%s] expected status %d, got %d (body: %s)", tc.name, tc.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestWebServer_HandleDatabasesBackup(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		body           string
		cfg            *domain.ServerConfig
		expectedStatus int
	}{
		{
			name:           "List Backups (GET)",
			method:         http.MethodGet,
			url:            "/api/databases/backup",
			body:           "",
			cfg:            &domain.ServerConfig{Host: "1.2.3.4"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Create Backup Missing Target Name (POST)",
			method:         http.MethodPost,
			url:            "/api/databases/backup",
			body:           `{"engine":"postgres"}`,
			cfg:            &domain.ServerConfig{Host: "1.2.3.4"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Create Backup Missing VPS Config (POST)",
			method:         http.MethodPost,
			url:            "/api/databases/backup",
			body:           `{"targetName":"db-main","engine":"postgres"}`,
			cfg:            nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Delete Backup Invalid ID (DELETE)",
			method:         http.MethodDelete,
			url:            "/api/databases/backup?id=invalid",
			body:           "",
			cfg:            &domain.ServerConfig{Host: "1.2.3.4"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Download Backup Invalid ID (GET)",
			method:         http.MethodGet,
			url:            "/api/backups/download?id=invalid",
			body:           "",
			cfg:            &domain.ServerConfig{Host: "1.2.3.4"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "List Backups with Target Filter (GET)",
			method:         http.MethodGet,
			url:            "/api/databases/backup?targetName=postgres-app",
			body:           "",
			cfg:            &domain.ServerConfig{Host: "1.2.3.4"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Restore Backup Method Not Allowed (GET)",
			method:         http.MethodGet,
			url:            "/api/backups/restore",
			body:           "",
			cfg:            &domain.ServerConfig{Host: "1.2.3.4"},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Restore Backup Invalid JSON (POST)",
			method:         http.MethodPost,
			url:            "/api/backups/restore",
			body:           `{invalid}`,
			cfg:            &domain.ServerConfig{Host: "1.2.3.4"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Restore Backup Missing VPS Config (POST)",
			method:         http.MethodPost,
			url:            "/api/backups/restore",
			body:           `{"backupId":1}`,
			cfg:            nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Download Backup Missing VPS Config (GET)",
			method:         http.MethodGet,
			url:            "/api/backups/download?id=1",
			body:           "",
			cfg:            nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Create Backup with Unconfigured Server (POST)",
			method:         http.MethodPost,
			url:            "/api/databases/backup?server=nonexistent",
			body:           `{"targetName":"db-main","engine":"postgres"}`,
			cfg:            nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepo{}
			ws := NewWebServer(repo, tc.cfg)

			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
			} else {
				req = httptest.NewRequest(tc.method, tc.url, nil)
			}

			rr := httptest.NewRecorder()
			if strings.HasPrefix(tc.url, "/api/backups/download") {
				ws.handleDownloadBackup(rr, req)
			} else if strings.HasPrefix(tc.url, "/api/backups/restore") {
				ws.handleRestoreBackup(rr, req)
			} else {
				ws.handleBackups(rr, req)
			}

			if rr.Code != tc.expectedStatus {
				t.Errorf("[%s] expected status %d, got %d (body: %s)", tc.name, tc.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestWebServer_HandleDNSCheck(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		cfg            *domain.ServerConfig
		expectedStatus int
	}{
		{
			name:           "Method Not Allowed (POST)",
			method:         http.MethodPost,
			url:            "/api/dns/check?domain=example.com",
			cfg:            &domain.ServerConfig{Host: "93.184.216.34"},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Missing Domain Parameter",
			method:         http.MethodGet,
			url:            "/api/dns/check",
			cfg:            &domain.ServerConfig{Host: "1.2.3.4"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid Domain Format",
			method:         http.MethodGet,
			url:            "/api/dns/check?domain=bad_domain;rm",
			cfg:            &domain.ServerConfig{Host: "1.2.3.4"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Valid Domain Query (Non-existent)",
			method:         http.MethodGet,
			url:            "/api/dns/check?domain=tarhiata-non-existent-test-domain-999.xyz",
			cfg:            &domain.ServerConfig{Host: "1.2.3.4"},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepo{}
			ws := NewWebServer(repo, tc.cfg)

			req := httptest.NewRequest(tc.method, tc.url, nil)
			rr := httptest.NewRecorder()
			ws.handleDNSCheck(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("[%s] expected status %d, got %d (body: %s)", tc.name, tc.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestWebServer_HandleEnvVars(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		body           string
		cfg            *domain.ServerConfig
		expectedStatus int
	}{
		{
			name:           "GET Missing Service Parameter",
			method:         http.MethodGet,
			url:            "/api/env",
			body:           "",
			cfg:            &domain.ServerConfig{Host: "1.2.3.4"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "GET Valid Service Parameter",
			method:         http.MethodGet,
			url:            "/api/env?service=api-gateway",
			body:           "",
			cfg:            &domain.ServerConfig{Host: "1.2.3.4"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST Missing ServiceName In Body",
			method:         http.MethodPost,
			url:            "/api/env",
			body:           `{"rawContent":"KEY=VALUE"}`,
			cfg:            &domain.ServerConfig{Host: "1.2.3.4"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "POST Valid Env Update",
			method:         http.MethodPost,
			url:            "/api/env",
			body:           `{"serviceName":"api-gateway","rawContent":"PORT=8080\nDEBUG=true"}`,
			cfg:            &domain.ServerConfig{Host: "1.2.3.4"},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepo{}
			ws := NewWebServer(repo, tc.cfg)

			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
			} else {
				req = httptest.NewRequest(tc.method, tc.url, nil)
			}

			rr := httptest.NewRecorder()
			ws.handleEnvVars(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("[%s] expected status %d, got %d (body: %s)", tc.name, tc.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestWebServer_HandleVolumes_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		handler        func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request)
		method         string
		url            string
		body           string
		expectedStatus []int
	}{
		{
			name: "ListVolumes returns empty array or vols without config",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleVolumes(rr, req)
			},
			method:         http.MethodGet,
			url:            "/api/volumes",
			expectedStatus: []int{http.StatusOK, http.StatusInternalServerError},
		},
		{
			name: "ListVolumeFiles returns bad request or 500 when unconfigured",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleVolumeFiles(rr, req)
			},
			method:         http.MethodGet,
			url:            "/api/volumes/files?path=/opt/data",
			expectedStatus: []int{http.StatusOK, http.StatusBadRequest, http.StatusInternalServerError},
		},
		{
			name: "VolumeRead missing path returns 400",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleVolumeRead(rr, req)
			},
			method:         http.MethodGet,
			url:            "/api/volumes/read",
			expectedStatus: []int{http.StatusBadRequest},
		},
		{
			name: "VolumeWrite missing path returns 400",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleVolumeWrite(rr, req)
			},
			method:         http.MethodPost,
			url:            "/api/volumes/write",
			body:           `{"content":"hello"}`,
			expectedStatus: []int{http.StatusBadRequest},
		},
		{
			name: "VolumeDownload missing path returns 400",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleVolumeDownload(rr, req)
			},
			method:         http.MethodGet,
			url:            "/api/volumes/download",
			expectedStatus: []int{http.StatusBadRequest},
		},
		{
			name: "VolumeDelete missing path returns 400",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleVolumeDelete(rr, req)
			},
			method:         http.MethodDelete,
			url:            "/api/volumes/delete",
			expectedStatus: []int{http.StatusBadRequest},
		},
		{
			name: "VolumeMkdir missing path returns 400",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleVolumeMkdir(rr, req)
			},
			method:         http.MethodPost,
			url:            "/api/volumes/mkdir",
			body:           `{}`,
			expectedStatus: []int{http.StatusBadRequest},
		},
		{
			name: "VolumeMkdir wrong method returns 405",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleVolumeMkdir(rr, req)
			},
			method:         http.MethodGet,
			url:            "/api/volumes/mkdir",
			expectedStatus: []int{http.StatusMethodNotAllowed},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepo{}
			cfg := &domain.ServerConfig{Host: "127.0.0.1", User: "root"}
			ws := NewWebServer(repo, cfg)

			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
			} else {
				req = httptest.NewRequest(tc.method, tc.url, nil)
			}

			rr := httptest.NewRecorder()
			tc.handler(ws, rr, req)

			matched := false
			for _, exp := range tc.expectedStatus {
				if rr.Code == exp {
					matched = true
					break
				}
			}
			if !matched {
				t.Errorf("[%s] unexpected status %d, allowed: %v (body: %s)", tc.name, rr.Code, tc.expectedStatus, rr.Body.String())
			}
		})
	}
}

func TestWebServer_HandleTerminal_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		handler        func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request)
		method         string
		url            string
		body           string
		expectedStatus int
	}{
		{
			name: "TerminalExec wrong method returns 405",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleTerminalExec(rr, req)
			},
			method:         http.MethodGet,
			url:            "/api/terminal/exec",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name: "TerminalExec invalid json returns 400",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleTerminalExec(rr, req)
			},
			method:         http.MethodPost,
			url:            "/api/terminal/exec",
			body:           `{invalid}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "TerminalExec invalid node id returns 400",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleTerminalExec(rr, req)
			},
			method:         http.MethodPost,
			url:            "/api/terminal/exec",
			body:           `{"nodeId":"node!@#$invalid"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "TerminalExec dangerous command returns 403",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleTerminalExec(rr, req)
			},
			method:         http.MethodPost,
			url:            "/api/terminal/exec",
			body:           `{"command":"rm -rf /"}`,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "TerminalExec empty command returns 200 with empty output",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleTerminalExec(rr, req)
			},
			method:         http.MethodPost,
			url:            "/api/terminal/exec",
			body:           `{"command":""}`,
			expectedStatus: http.StatusOK,
		},
		{
			name: "OpenTerminal wrong method returns 405",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleOpenTerminal(rr, req)
			},
			method:         http.MethodGet,
			url:            "/api/servers/open-terminal",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name: "OpenTerminal invalid json returns 400",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleOpenTerminal(rr, req)
			},
			method:         http.MethodPost,
			url:            "/api/servers/open-terminal",
			body:           `{bad_json`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "OpenTerminal missing server returns 404",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleOpenTerminal(rr, req)
			},
			method:         http.MethodPost,
			url:            "/api/servers/open-terminal",
			body:           `{"name":"non-existent-server-xyz"}`,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepo{}
			cfg := &domain.ServerConfig{Name: "local", Host: "localhost", CloudProvider: "local"}
			ws := NewWebServer(repo, cfg)

			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
			} else {
				req = httptest.NewRequest(tc.method, tc.url, nil)
			}

			rr := httptest.NewRecorder()
			tc.handler(ws, rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("[%s] expected status %d, got %d (body: %s)", tc.name, tc.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestWebServer_HandleSSLAndMaintenance_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		handler        func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request)
		method         string
		url            string
		body           string
		expectedStatus int
	}{
		{
			name: "SSL inspect wrong method returns 405",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleSSLInspect(rr, req)
			},
			method:         http.MethodPost,
			url:            "/api/ssl/inspect",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name: "SSL inspect GET returns 200",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleSSLInspect(rr, req)
			},
			method:         http.MethodGet,
			url:            "/api/ssl/inspect",
			expectedStatus: http.StatusOK,
		},
		{
			name: "Maintenance toggle wrong method returns 405",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleMaintenanceToggle(rr, req)
			},
			method:         http.MethodGet,
			url:            "/api/maintenance/toggle",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name: "Maintenance toggle invalid json returns 400",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleMaintenanceToggle(rr, req)
			},
			method:         http.MethodPost,
			url:            "/api/maintenance/toggle",
			body:           `{invalid-json}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Maintenance toggle empty service name returns 400",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleMaintenanceToggle(rr, req)
			},
			method:         http.MethodPost,
			url:            "/api/maintenance/toggle",
			body:           `{"serviceName":""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "SSL reload wrong method returns 405",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleSSLReload(rr, req)
			},
			method:         http.MethodGet,
			url:            "/api/ssl/reload",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepo{}
			cfg := &domain.ServerConfig{Name: "local", Host: "localhost", CloudProvider: "local"}
			ws := NewWebServer(repo, cfg)

			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
			} else {
				req = httptest.NewRequest(tc.method, tc.url, nil)
			}

			rr := httptest.NewRecorder()
			tc.handler(ws, rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("[%s] expected status %d, got %d (body: %s)", tc.name, tc.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestLocalAuthMiddleware_SecurityMatrix(t *testing.T) {
	tests := []struct {
		name           string
		isExposed      bool
		apiKey         string
		remoteAddr     string
		headerKey      string
		queryKey       string
		expectedStatus int
	}{
		{
			name:           "unexposed local dev allows access without key",
			isExposed:      false,
			apiKey:         "",
			remoteAddr:     "127.0.0.1:45678",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "exposed server allows localhost access without key",
			isExposed:      true,
			apiKey:         "",
			remoteAddr:     "127.0.0.1:45678",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "exposed server blocks remote IP when no key configured",
			isExposed:      true,
			apiKey:         "",
			remoteAddr:     "192.168.1.105:45678",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "exposed server rejects remote IP with wrong key",
			isExposed:      true,
			apiKey:         "secret-pass-123",
			remoteAddr:     "192.168.1.105:45678",
			headerKey:      "wrong-key",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "exposed server allows remote IP with valid X-API-Key header",
			isExposed:      true,
			apiKey:         "secret-pass-123",
			remoteAddr:     "192.168.1.105:45678",
			headerKey:      "secret-pass-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "exposed server allows remote IP with valid api_key query param",
			isExposed:      true,
			apiKey:         "secret-pass-123",
			remoteAddr:     "192.168.1.105:45678",
			queryKey:       "secret-pass-123",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := NewWebServer(&mockRepo{}, &domain.ServerConfig{Name: "srv"})
			ws.SetExposed(tt.isExposed)
			ws.SetAPIKey(tt.apiKey)

			dummyHandler := func(rw http.ResponseWriter, req *http.Request) {
				rw.WriteHeader(http.StatusOK)
			}
			wrapped := ws.localAuthMiddleware(dummyHandler)

			targetURL := "/api/critical-endpoint"
			if tt.queryKey != "" {
				targetURL += "?api_key=" + tt.queryKey
			}
			req := httptest.NewRequest(http.MethodPost, targetURL, nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.headerKey != "" {
				req.Header.Set("X-API-Key", tt.headerKey)
			}

			rr := httptest.NewRecorder()
			wrapped(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("[%s] expected status %d, got %d", tt.name, tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestHandleOpenTerminal_SecurityEnforcement(t *testing.T) {
	tests := []struct {
		name           string
		isExposed      bool
		remoteAddr     string
		expectedStatus int
	}{
		{
			name:           "remote IP on exposed server is blocked from spawning GUI terminal",
			isExposed:      true,
			remoteAddr:     "192.168.1.80:54321",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "localhost on exposed server passes IP check",
			isExposed:      true,
			remoteAddr:     "127.0.0.1:54321",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := NewWebServer(&mockRepo{}, &domain.ServerConfig{Name: "srv"})
			ws.SetExposed(tt.isExposed)

			req := httptest.NewRequest(http.MethodPost, "/api/servers/open-terminal", nil)
			req.RemoteAddr = tt.remoteAddr

			rr := httptest.NewRecorder()
			ws.handleOpenTerminal(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("[%s] expected status %d, got %d", tt.name, tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestMutatingEndpoints_ExposedRemoteProtection(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		handler        func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request)
		expectedStatus int
	}{
		{
			name:   "GET services is allowed for remote reader",
			method: http.MethodGet,
			url:    "/api/services",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleServices(rr, req)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "POST services is blocked for remote user without key",
			method: http.MethodPost,
			url:    "/api/services",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleServices(rr, req)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "GET databases is allowed for remote reader",
			method: http.MethodGet,
			url:    "/api/databases",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleDatabases(rr, req)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "POST databases is blocked for remote user without key",
			method: http.MethodPost,
			url:    "/api/databases",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleDatabases(rr, req)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "POST links is blocked for remote user without key",
			method: http.MethodPost,
			url:    "/api/links",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleLinks(rr, req)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "POST servers is blocked for remote user without key",
			method: http.MethodPost,
			url:    "/api/servers",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleServers(rr, req)
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := NewWebServer(&mockRepo{}, &domain.ServerConfig{Name: "srv"})
			ws.SetExposed(true)

			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.RemoteAddr = "192.168.1.200:54321"

			rr := httptest.NewRecorder()
			tt.handler(ws, rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("[%s] expected status %d, got %d", tt.name, tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestContainerStats_CacheAndFallback(t *testing.T) {
	ws := NewWebServer(&mockRepo{}, &domain.ServerConfig{Name: "local", Host: "127.0.0.1"})

	// Pre-populate cache
	cachedData := domain.ContainerStats{
		Container: "my-app",
		CPUPerc:   "2.5%",
		MemUsage:  "120MiB / 1GiB",
	}
	ws.setCache("stats:127.0.0.1:my-app", cachedData, 5*time.Second)

	req := httptest.NewRequest(http.MethodGet, "/api/stats?name=my-app", nil)
	rr := httptest.NewRecorder()
	ws.handleContainerStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var res domain.ContainerStats
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if res.CPUPerc != "2.5%" {
		t.Errorf("expected cached stats CPU 2.5%%, got %s", res.CPUPerc)
	}
}

func TestDBHealth_CacheAndDynamicPassword(t *testing.T) {
	ws := NewWebServer(&mockRepo{}, &domain.ServerConfig{Name: "local", Host: "127.0.0.1"})

	cachedHealth := domain.DBHealthStats{
		Engine:            "postgres",
		ActiveConnections: 7,
		Status:            "Healthy",
	}
	ws.setCache("dbhealth:127.0.0.1:my-db", cachedHealth, 5*time.Second)

	req := httptest.NewRequest(http.MethodGet, "/api/databases/health?name=my-db", nil)
	rr := httptest.NewRecorder()
	ws.handleDBHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var res domain.DBHealthStats
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if res.ActiveConnections != 7 {
		t.Errorf("expected cached active connections 7, got %d", res.ActiveConnections)
	}
}

func TestVultrPlans_Cache(t *testing.T) {
	ws := NewWebServer(&mockRepo{}, &domain.ServerConfig{Name: "local", VultrAPIToken: "test-token"})

	cachedPlans := []domain.VultrPlan{
		{ID: "vc2-1c-1gb", VCPU: 1, RAM: 1024, MonthlyCost: 5.0},
	}
	ws.setCache("vultr:plans:test-token", cachedPlans, 5*time.Minute)

	req := httptest.NewRequest(http.MethodGet, "/api/vultr/plans", nil)
	rr := httptest.NewRecorder()
	ws.handleVultrPlans(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var res []domain.VultrPlan
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if len(res) != 1 || res[0].ID != "vc2-1c-1gb" {
		t.Errorf("expected cached plan vc2-1c-1gb, got %+v", res)
	}
}

type targetMockRepo struct {
	mockRepo
	servers map[string]*domain.ServerConfig
}

func (m *targetMockRepo) GetServerConfigByName(name string) (*domain.ServerConfig, error) {
	if cfg, ok := m.servers[name]; ok {
		return cfg, nil
	}
	return nil, fmt.Errorf("server %s not found", name)
}

func TestWebServer_ResolveTargetServer_TableDriven(t *testing.T) {
	defaultCfg := &domain.ServerConfig{Name: "default-vps", Host: "10.0.0.1"}
	vps2Cfg := &domain.ServerConfig{Name: "vps-secondary", Host: "10.0.0.2"}

	repo := &targetMockRepo{
		servers: map[string]*domain.ServerConfig{
			"vps-secondary": vps2Cfg,
		},
	}

	ws := NewWebServer(repo, defaultCfg)

	tests := []struct {
		name       string
		targetName string
		expectedIP string
	}{
		{
			name:       "empty target returns default server",
			targetName: "",
			expectedIP: "10.0.0.1",
		},
		{
			name:       "unknown target returns default server",
			targetName: "unknown-vps",
			expectedIP: "10.0.0.1",
		},
		{
			name:       "existing target returns requested server",
			targetName: "vps-secondary",
			expectedIP: "10.0.0.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := ws.resolveTargetServer(tt.targetName)
			if res == nil {
				t.Fatalf("expected non-nil server config")
			}
			if res.Host != tt.expectedIP {
				t.Errorf("expected host %s, got %s", tt.expectedIP, res.Host)
			}
		})
	}
}

func TestHandleServices_TargetServerAndStream_TableDriven(t *testing.T) {
	defaultCfg := &domain.ServerConfig{Name: "default-vps", Host: "10.0.0.1"}
	vps2Cfg := &domain.ServerConfig{Name: "vps-secondary", Host: "10.0.0.2"}

	repo := &targetMockRepo{
		servers: map[string]*domain.ServerConfig{
			"vps-secondary": vps2Cfg,
		},
	}

	ws := NewWebServer(repo, defaultCfg)

	tests := []struct {
		name           string
		method         string
		url            string
		body           string
		expectedStatus int
	}{
		{
			name:           "POST invalid json returns 400",
			method:         http.MethodPost,
			url:            "/api/deploy-service",
			body:           `{invalid}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "POST invalid service name returns 400",
			method:         http.MethodPost,
			url:            "/api/deploy-service",
			body:           `{"name":"invalid name with spaces!","imageSource":"nginx"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "DELETE missing name returns 400",
			method:         http.MethodDelete,
			url:            "/api/deploy-service",
			body:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "DELETE with target server resolves without error",
			method:         http.MethodDelete,
			url:            "/api/deploy-service?name=valid-svc&server=vps-secondary",
			body:           "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyReader io.Reader
			if tt.body != "" {
				bodyReader = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.url, bodyReader)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			ws.handleServices(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestHandleDatabases_TargetServerAndStream_TableDriven(t *testing.T) {
	defaultCfg := &domain.ServerConfig{Name: "default-vps", Host: "10.0.0.1"}
	vps2Cfg := &domain.ServerConfig{Name: "vps-secondary", Host: "10.0.0.2"}

	repo := &targetMockRepo{
		servers: map[string]*domain.ServerConfig{
			"vps-secondary": vps2Cfg,
		},
	}

	ws := NewWebServer(repo, defaultCfg)

	tests := []struct {
		name           string
		method         string
		url            string
		body           string
		expectedStatus int
	}{
		{
			name:           "POST invalid json returns 400",
			method:         http.MethodPost,
			url:            "/api/deploy-db",
			body:           `{invalid}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "POST invalid db name returns 400",
			method:         http.MethodPost,
			url:            "/api/deploy-db",
			body:           `{"name":"invalid db name!","engine":"postgres"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "DELETE missing db name returns 400",
			method:         http.MethodDelete,
			url:            "/api/deploy-db",
			body:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "DELETE with target server resolves without error",
			method:         http.MethodDelete,
			url:            "/api/deploy-db?name=valid-db&server=vps-secondary",
			body:           "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyReader io.Reader
			if tt.body != "" {
				bodyReader = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.url, bodyReader)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			ws.handleDatabases(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestHandleNodes_TargetServer_TableDriven(t *testing.T) {
	defaultCfg := &domain.ServerConfig{Name: "default-vps", Host: "10.0.0.1"}
	vps2Cfg := &domain.ServerConfig{Name: "vps-secondary", Host: "10.0.0.2"}

	repo := &targetMockRepo{
		servers: map[string]*domain.ServerConfig{
			"vps-secondary": vps2Cfg,
		},
	}

	ws := NewWebServer(repo, defaultCfg)

	tests := []struct {
		name           string
		handler        func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request)
		method         string
		url            string
		body           string
		expectedStatus int
	}{
		{
			name: "handleNodes DELETE missing id returns 400",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleNodes(rr, req)
			},
			method:         http.MethodDelete,
			url:            "/api/nodes",
			body:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "handleNodes DELETE invalid id returns 400",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleNodes(rr, req)
			},
			method:         http.MethodDelete,
			url:            "/api/nodes?id=invalid!id",
			body:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "handleNodes DELETE unconfigured server returns 400",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				emptyWS := NewWebServer(repo, nil)
				emptyWS.handleNodes(rr, req)
			},
			method:         http.MethodDelete,
			url:            "/api/nodes?id=node-123&server=non-existent",
			body:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "handleNodeJoinToken POST method not allowed returns 405",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleNodeJoinToken(rr, req)
			},
			method:         http.MethodPost,
			url:            "/api/nodes/join-token",
			body:           "",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name: "handleNodeJoinToken unconfigured server returns 400",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				emptyWS := NewWebServer(repo, nil)
				emptyWS.handleNodeJoinToken(rr, req)
			},
			method:         http.MethodGet,
			url:            "/api/nodes/join-token?server=non-existent",
			body:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "handleNodeUpdate GET method not allowed returns 405",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleNodeUpdate(rr, req)
			},
			method:         http.MethodGet,
			url:            "/api/nodes/update",
			body:           "",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name: "handleNodeUpdate invalid json returns 400",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleNodeUpdate(rr, req)
			},
			method:         http.MethodPost,
			url:            "/api/nodes/update",
			body:           "{invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "handleNodeUpdate invalid node id returns 400",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				ws.handleNodeUpdate(rr, req)
			},
			method:         http.MethodPost,
			url:            "/api/nodes/update",
			body:           `{"id":"invalid!id","availability":"drain"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "handleNodeUpdate unconfigured server returns 400",
			handler: func(ws *WebServer, rr *httptest.ResponseRecorder, req *http.Request) {
				emptyWS := NewWebServer(repo, nil)
				emptyWS.handleNodeUpdate(rr, req)
			},
			method:         http.MethodPost,
			url:            "/api/nodes/update?server=non-existent",
			body:           `{"id":"node-123","availability":"drain"}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyReader io.Reader
			if tt.body != "" {
				bodyReader = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.url, bodyReader)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			tt.handler(ws, rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestWebServer_SecurityAndAuth_TableDriven(t *testing.T) {
	repo := &mockRepo{}
	cfg := &domain.ServerConfig{Host: "127.0.0.1"}

	tests := []struct {
		name           string
		apiKey         string
		isExposed      bool
		remoteAddr     string
		url            string
		headerKey      string
		headerVal      string
		expectedStatus int
	}{
		{
			name:           "Local request without API key allowed",
			apiKey:         "",
			isExposed:      false,
			remoteAddr:     "127.0.0.1:12345",
			url:            "/api/protected",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Exposed server with valid X-API-Key header allowed",
			apiKey:         "secret-token-123",
			isExposed:      true,
			remoteAddr:     "198.51.100.1:54321",
			url:            "/api/protected",
			headerKey:      "X-API-Key",
			headerVal:      "secret-token-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Exposed server with valid Authorization Bearer token allowed",
			apiKey:         "secret-token-123",
			isExposed:      true,
			remoteAddr:     "198.51.100.1:54321",
			url:            "/api/protected",
			headerKey:      "Authorization",
			headerVal:      "Bearer secret-token-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Exposed server with valid key query parameter allowed",
			apiKey:         "secret-token-123",
			isExposed:      true,
			remoteAddr:     "198.51.100.1:54321",
			url:            "/api/protected?key=secret-token-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Exposed server with invalid API key returns 401",
			apiKey:         "secret-token-123",
			isExposed:      true,
			remoteAddr:     "198.51.100.1:54321",
			url:            "/api/protected",
			headerKey:      "X-API-Key",
			headerVal:      "wrong-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Exposed server without API key configured blocks external IP with 403",
			apiKey:         "",
			isExposed:      true,
			remoteAddr:     "198.51.100.1:54321",
			url:            "/api/protected",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Exposed server without API key configured permits loopback caller",
			apiKey:         "",
			isExposed:      true,
			remoteAddr:     "127.0.0.1:54321",
			url:            "/api/protected",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := NewWebServer(repo, cfg)
			ws.SetAPIKey(tt.apiKey)
			ws.SetExposed(tt.isExposed)

			dummyHandler := ws.localAuthMiddleware(func(rw http.ResponseWriter, req *http.Request) {
				rw.WriteHeader(http.StatusOK)
				if _, wErr := rw.Write([]byte(`{"status":"ok"}`)); wErr != nil {
					t.Fatalf("failed to write response: %v", wErr)
				}
			})

			req := httptest.NewRequest(http.MethodPost, tt.url, nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.headerKey != "" {
				req.Header.Set(tt.headerKey, tt.headerVal)
			}

			rr := httptest.NewRecorder()
			dummyHandler(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestWebServer_EchoEngineIntegration(t *testing.T) {
	repo := &mockRepo{}
	cfg := &domain.ServerConfig{Host: "127.0.0.1", User: "root"}
	ws := NewWebServer(repo, cfg)

	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
	}{
		{
			name:           "Echo router dispatches status endpoint correctly",
			method:         http.MethodGet,
			url:            "/api/status",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Echo router applies security headers",
			method:         http.MethodGet,
			url:            "/api/status",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.url, nil)
			rr := httptest.NewRecorder()

			// Test ServeHTTP delegation to Echo engine
			ws.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, rr.Code)
			}

			// Validate Echo security headers middleware
			if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Errorf("expected X-Content-Type-Options: nosniff, got: %s", rr.Header().Get("X-Content-Type-Options"))
			}
			if rr.Header().Get("X-Frame-Options") != "DENY" {
				t.Errorf("expected X-Frame-Options: DENY, got: %s", rr.Header().Get("X-Frame-Options"))
			}
		})
	}
}

func TestWebServer_HandleObservabilitySuite(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		setupMock      func(m *mocks.MockConfigRepository)
		body           string
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name:   "GET observability when disabled returns enabled false",
			method: http.MethodGet,
			setupMock: func(m *mocks.MockConfigRepository) {
				m.Observability = nil
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, `"enabled":false`) {
					t.Errorf("expected enabled:false, got: %s", body)
				}
			},
		},
		{
			name:   "GET observability when enabled returns details",
			method: http.MethodGet,
			setupMock: func(m *mocks.MockConfigRepository) {
				m.Observability = &domain.SavedObservability{
					DeployType:      "swarm",
					ExternalURL:     "https://obs.tarhiata.local",
					GrafanaPassword: "admin_pass_secure",
				}
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, `"enabled":true`) || !strings.Contains(body, "obs.tarhiata.local") {
					t.Errorf("expected enabled:true with url, got: %s", body)
				}
			},
		},
		{
			name:           "POST observability with invalid payload returns 400",
			method:         http.MethodPost,
			setupMock:      func(m *mocks.MockConfigRepository) {},
			body:           `{invalid_json`,
			expectedStatus: http.StatusBadRequest,
			checkBody:      nil,
		},
		{
			name:           "PUT observability returns 405 Method Not Allowed",
			method:         http.MethodPut,
			setupMock:      func(m *mocks.MockConfigRepository) {},
			expectedStatus: http.StatusMethodNotAllowed,
			checkBody:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mRepo := mocks.NewMockConfigRepository()
			tt.setupMock(mRepo)
			cfg := &domain.ServerConfig{Host: "127.0.0.1", User: "root"}
			ws := NewWebServer(mRepo, cfg)

			var reqBody io.Reader
			if tt.body != "" {
				reqBody = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(tt.method, "/api/observability", reqBody)
			rr := httptest.NewRecorder()

			ws.handleObservability(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}
			if tt.checkBody != nil {
				tt.checkBody(t, rr.Body.String())
			}
		})
	}
}

func TestWebServer_HandleMigrationsSuite(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		targetURL      string
		body           string
		setupMock      func(m *mocks.MockConfigRepository)
		expectedStatus int
		checkBody      func(t *testing.T, mRepo *mocks.MockConfigRepository, body string)
	}{
		{
			name:      "GET migrations returns empty array when none exist",
			method:    http.MethodGet,
			targetURL: "/api/migrations?db=shop-db",
			setupMock: func(m *mocks.MockConfigRepository) {},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, mRepo *mocks.MockConfigRepository, body string) {
				var files []domain.MigrationFile
				if err := json.Unmarshal([]byte(body), &files); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if len(files) != 0 {
					t.Errorf("expected 0 files, got %d", len(files))
				}
			},
		},
		{
			name:      "POST migration file saves successfully",
			method:    http.MethodPost,
			targetURL: "/api/migrations/file",
			body: `{
				"dbName": "shop-db",
				"filename": "001_create_users.sql",
				"content": "CREATE TABLE users (id SERIAL PRIMARY KEY);",
				"downContent": "DROP TABLE users;"
			}`,
			setupMock:      func(m *mocks.MockConfigRepository) {},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, mRepo *mocks.MockConfigRepository, body string) {
				if len(mRepo.Migrations) != 1 {
					t.Fatalf("expected 1 migration in mock, got %d", len(mRepo.Migrations))
				}
				if mRepo.Migrations[0].Filename != "001_create_users.sql" {
					t.Errorf("unexpected filename: %s", mRepo.Migrations[0].Filename)
				}
			},
		},
		{
			name:           "POST migration file with invalid JSON returns 400",
			method:         http.MethodPost,
			targetURL:      "/api/migrations/file",
			body:           `{bad_json`,
			setupMock:      func(m *mocks.MockConfigRepository) {},
			expectedStatus: http.StatusBadRequest,
			checkBody:      nil,
		},
		{
			name:      "DELETE migration file removes from repository",
			method:    http.MethodDelete,
			targetURL: "/api/migrations/file?db=shop-db&filename=001_create_users.sql",
			setupMock: func(m *mocks.MockConfigRepository) {
				if err := m.SaveMigrationFile(domain.MigrationFile{
					DBName:   "shop-db",
					Filename: "001_create_users.sql",
					Status:   "pending",
				}); err != nil {
					t.Fatalf("failed to setup migration: %v", err)
				}
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, mRepo *mocks.MockConfigRepository, body string) {
				if len(mRepo.Migrations) != 0 {
					t.Errorf("expected migration to be deleted, remaining: %d", len(mRepo.Migrations))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mRepo := mocks.NewMockConfigRepository()
			tt.setupMock(mRepo)
			cfg := &domain.ServerConfig{Host: "127.0.0.1", User: "root"}
			ws := NewWebServer(mRepo, cfg)

			var reqBody io.Reader
			if tt.body != "" {
				reqBody = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.targetURL, reqBody)
			rr := httptest.NewRecorder()

			if strings.HasPrefix(tt.targetURL, "/api/migrations/file") {
				ws.handleMigrationFile(rr, req)
			} else {
				ws.handleMigrations(rr, req)
			}

			if rr.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}
			if tt.checkBody != nil {
				tt.checkBody(t, mRepo, rr.Body.String())
			}
		})
	}
}

func TestWebServer_HandleBackupsSuite(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		targetURL      string
		body           string
		setupMock      func(m *mocks.MockConfigRepository)
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name:      "GET backups returns all items",
			method:    http.MethodGet,
			targetURL: "/api/backups",
			setupMock: func(m *mocks.MockConfigRepository) {
				if err := m.SaveBackup(domain.SavedBackup{
					ID:         1,
					TargetName: "db-main",
					Engine:     "postgres",
					Filename:   "backup_1.sql",
				}); err != nil {
					t.Fatalf("failed to save mock backup: %v", err)
				}
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				var backups []domain.SavedBackup
				if err := json.Unmarshal([]byte(body), &backups); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}
				if len(backups) != 1 || backups[0].TargetName != "db-main" {
					t.Errorf("unexpected backups response: %+v", backups)
				}
			},
		},
		{
			name:      "GET backups with targetName filter filters correctly",
			method:    http.MethodGet,
			targetURL: "/api/backups?targetName=db-redis",
			setupMock: func(m *mocks.MockConfigRepository) {
				if err := m.SaveBackup(domain.SavedBackup{
					ID:         1,
					TargetName: "db-main",
				}); err != nil {
					t.Fatalf("failed to save mock backup: %v", err)
				}
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				var backups []domain.SavedBackup
				if err := json.Unmarshal([]byte(body), &backups); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}
				if len(backups) != 0 {
					t.Errorf("expected 0 backups for filter, got %d", len(backups))
				}
			},
		},
		{
			name:           "POST backup with missing target name returns 400",
			method:         http.MethodPost,
			targetURL:      "/api/backups",
			body:           `{"targetType":"database"}`,
			setupMock:      func(m *mocks.MockConfigRepository) {},
			expectedStatus: http.StatusBadRequest,
			checkBody:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mRepo := mocks.NewMockConfigRepository()
			tt.setupMock(mRepo)
			cfg := &domain.ServerConfig{Host: "127.0.0.1", User: "root"}
			ws := NewWebServer(mRepo, cfg)

			var reqBody io.Reader
			if tt.body != "" {
				reqBody = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.targetURL, reqBody)
			rr := httptest.NewRecorder()

			ws.handleBackups(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}
			if tt.checkBody != nil {
				tt.checkBody(t, rr.Body.String())
			}
		})
	}
}

func TestWebServer_HandleAuditLogsSuite(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		setupMock      func(m *mocks.MockConfigRepository)
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name:   "GET audit logs returns logs list",
			method: http.MethodGet,
			setupMock: func(m *mocks.MockConfigRepository) {
				if err := m.SaveAuditLog(domain.AuditLog{
					ID:           1,
					Action:       "DEPLOY",
					ResourceType: "service",
					ResourceName: "shop-api",
					Details:      "Deployed by developer",
					Timestamp:    time.Now(),
				}); err != nil {
					t.Fatalf("failed to save mock audit log: %v", err)
				}
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				var logs []domain.AuditLog
				if err := json.Unmarshal([]byte(body), &logs); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}
				if len(logs) != 1 || logs[0].Action != "DEPLOY" {
					t.Errorf("unexpected audit logs list: %+v", logs)
				}
			},
		},
		{
			name:           "POST audit logs returns 405 Method Not Allowed",
			method:         http.MethodPost,
			setupMock:      func(m *mocks.MockConfigRepository) {},
			expectedStatus: http.StatusMethodNotAllowed,
			checkBody:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mRepo := mocks.NewMockConfigRepository()
			tt.setupMock(mRepo)
			cfg := &domain.ServerConfig{Host: "127.0.0.1", User: "root"}
			ws := NewWebServer(mRepo, cfg)

			req := httptest.NewRequest(tt.method, "/api/audit-logs", nil)
			rr := httptest.NewRecorder()

			ws.handleAuditLogs(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
			if tt.checkBody != nil {
				tt.checkBody(t, rr.Body.String())
			}
		})
	}
}

func TestWebServer_HandleDNSCheckSuite(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name:           "DNS check with missing domain param returns 400",
			method:         http.MethodGet,
			url:            "/api/dns/check",
			expectedStatus: http.StatusBadRequest,
			checkBody:      nil,
		},
		{
			name:           "DNS check with invalid domain format returns 400",
			method:         http.MethodGet,
			url:            "/api/dns/check?domain=invalid..domain!@#",
			expectedStatus: http.StatusBadRequest,
			checkBody:      nil,
		},
		{
			name:           "DNS check with valid domain format performs lookup",
			method:         http.MethodGet,
			url:            "/api/dns/check?domain=example.com",
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				var res map[string]interface{}
				if err := json.Unmarshal([]byte(body), &res); err != nil {
					t.Fatalf("failed to unmarshal DNS response: %v", err)
				}
				if res["domain"] != "example.com" {
					t.Errorf("expected domain 'example.com', got: %v", res["domain"])
				}
			},
		},
		{
			name:           "DNS check with POST method returns 405",
			method:         http.MethodPost,
			url:            "/api/dns/check?domain=example.com",
			expectedStatus: http.StatusMethodNotAllowed,
			checkBody:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mRepo := mocks.NewMockConfigRepository()
			cfg := &domain.ServerConfig{Host: "127.0.0.1", User: "root"}
			ws := NewWebServer(mRepo, cfg)

			req := httptest.NewRequest(tt.method, tt.url, nil)
			rr := httptest.NewRecorder()

			ws.handleDNSCheck(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
			if tt.checkBody != nil {
				tt.checkBody(t, rr.Body.String())
			}
		})
	}
}

func TestWebServer_HandleRegistriesSuite(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		body           string
		setupMock      func(m *mocks.MockConfigRepository)
		expectedStatus int
		checkBody      func(t *testing.T, mRepo *mocks.MockConfigRepository, body string)
	}{
		{
			name:      "GET registries masks passwords",
			method:    http.MethodGet,
			url:       "/api/registries",
			setupMock: func(m *mocks.MockConfigRepository) {
				if err := m.SaveRegistryCredential(domain.SavedRegistryCredential{
					Server:   "ghcr.io",
					Username: "octocat",
					Password: "supersecretpassword123",
				}); err != nil {
					t.Fatalf("failed to save mock registry: %v", err)
				}
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, mRepo *mocks.MockConfigRepository, body string) {
				var list []domain.SavedRegistryCredential
				if err := json.Unmarshal([]byte(body), &list); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if len(list) != 1 || list[0].Server != "ghcr.io" {
					t.Fatalf("unexpected list: %+v", list)
				}
				if list[0].Password != "••••••••" {
					t.Errorf("expected masked password, got: %s", list[0].Password)
				}
			},
		},
		{
			name:           "POST registry attempts save and validates auth",
			method:         http.MethodPost,
			url:            "/api/registries",
			body:           `{"server":"docker.io","username":"admin","password":"password123"}`,
			setupMock:      func(m *mocks.MockConfigRepository) {},
			expectedStatus: http.StatusBadRequest,
			checkBody:      nil,
		},
		{
			name:      "DELETE registry removes credential",
			method:    http.MethodDelete,
			url:       "/api/registries?server=docker.io",
			setupMock: func(m *mocks.MockConfigRepository) {
				if err := m.SaveRegistryCredential(domain.SavedRegistryCredential{
					Server:   "docker.io",
					Username: "admin",
				}); err != nil {
					t.Fatalf("failed to save mock registry: %v", err)
				}
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, mRepo *mocks.MockConfigRepository, body string) {
				if len(mRepo.Registries) != 0 {
					t.Errorf("expected registry to be deleted, remaining: %d", len(mRepo.Registries))
				}
			},
		},
		{
			name:           "POST registry with invalid json returns 400",
			method:         http.MethodPost,
			url:            "/api/registries",
			body:           `{bad_json`,
			setupMock:      func(m *mocks.MockConfigRepository) {},
			expectedStatus: http.StatusBadRequest,
			checkBody:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mRepo := mocks.NewMockConfigRepository()
			tt.setupMock(mRepo)
			cfg := &domain.ServerConfig{Host: "127.0.0.1", User: "root"}
			ws := NewWebServer(mRepo, cfg)

			var reqBody io.Reader
			if tt.body != "" {
				reqBody = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.url, reqBody)
			rr := httptest.NewRecorder()

			ws.handleRegistries(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}
			if tt.checkBody != nil {
				tt.checkBody(t, mRepo, rr.Body.String())
			}
		})
	}
}

// TestWebServer_HandleSystemReport valida el flujo completo del endpoint: cada usecase
// (Host, Security, Swarm) obtiene su propia conexión SSH independiente y debe completar
// exitosamente sin errores, incluso encadenados uno tras otro en la misma petición
// (antes, la conexión compartida se cerraba tras el primer usecase, dejando el reporte
// de seguridad vacío en silencio).
func TestWebServer_HandleSystemReport(t *testing.T) {
	repo := &mockRepo{}
	cfg := &domain.ServerConfig{
		Name:          "local",
		Host:          "localhost",
		CloudProvider: "local",
	}
	ws := NewWebServer(repo, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/system/report", nil)
	rr := httptest.NewRecorder()
	ws.handleSystemReport(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var res domain.SystemDiagnosticReport
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode SystemDiagnosticReport: %v", err)
	}
	if res.ServerName != "local" {
		t.Errorf("expected serverName 'local', got '%s'", res.ServerName)
	}
	if res.Telemetry.ServerName != "local" {
		t.Errorf("expected Telemetry.ServerName 'local' (InspectHostUseCase debió correr con su propia conexión), got '%s'", res.Telemetry.ServerName)
	}
}

// TestWebServer_HandleWebhookDeploy_SignatureEnforcement valida el flujo completo del fix:
// antes, el "secreto" venía de la misma query string de la request entrante (un atacante
// controlaba ambos lados de la comparación) y, sin firma, el deploy se ejecutaba igual.
// Ahora el secreto debe venir guardado del lado del servidor para ese servicio, y sin
// firma válida el deploy se rechaza.
func TestWebServer_HandleWebhookDeploy_SignatureEnforcement(t *testing.T) {
	body := []byte(`{"service":"web-api","image":"myrepo/web-api:v2"}`)

	sign := func(secret string) string {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		return "sha256=" + hex.EncodeToString(mac.Sum(nil))
	}

	newServer := func(webhookSecret string) *WebServer {
		repo := mocks.NewMockConfigRepository()
		repo.Services = []domain.SavedService{{Name: "web-api", WebhookSecret: webhookSecret, ServerName: "local"}}
		return NewWebServer(repo, &domain.ServerConfig{Name: "local", Host: "localhost", CloudProvider: "local"})
	}

	t.Run("sin secreto configurado se rechaza (antes el ataque de query-string self-referencial pasaba)", func(t *testing.T) {
		ws := newServer("") // servicio sin webhook secret configurado
		req := httptest.NewRequest(http.MethodPost, "/api/webhooks/deploy?service=web-api&token=atacante-controla-esto", bytes.NewReader(body))
		req.Header.Set("X-Hub-Signature-256", sign("atacante-controla-esto"))
		rr := httptest.NewRecorder()
		ws.handleWebhookDeploy(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 (sin secreto configurado), got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("sin header de firma se rechaza aunque el secreto exista", func(t *testing.T) {
		ws := newServer("secreto-real-del-servidor")
		req := httptest.NewRequest(http.MethodPost, "/api/webhooks/deploy?service=web-api", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		ws.handleWebhookDeploy(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 (sin firma), got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("firma calculada con el secreto que manda el atacante ya no sirve", func(t *testing.T) {
		ws := newServer("secreto-real-del-servidor")
		req := httptest.NewRequest(http.MethodPost, "/api/webhooks/deploy?service=web-api&token=secreto-falso-del-atacante", bytes.NewReader(body))
		req.Header.Set("X-Hub-Signature-256", sign("secreto-falso-del-atacante"))
		rr := httptest.NewRecorder()
		ws.handleWebhookDeploy(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 (firma calculada con el secreto equivocado), got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("firma calculada con el secreto real guardado en el servidor pasa la validación de auth", func(t *testing.T) {
		ws := newServer("secreto-real-del-servidor")
		req := httptest.NewRequest(http.MethodPost, "/api/webhooks/deploy?service=web-api", bytes.NewReader(body))
		req.Header.Set("X-Hub-Signature-256", sign("secreto-real-del-servidor"))
		rr := httptest.NewRecorder()
		ws.handleWebhookDeploy(rr, req)
		// No afirmamos 200: el despliegue real depende de que exista un servicio "web-api"
		// en Docker en esta máquina. Lo que valida este caso es que la capa de
		// autenticación (lo que se rompió) ya no bloquea una firma válida.
		if rr.Code == http.StatusUnauthorized || rr.Code == http.StatusForbidden {
			t.Fatalf("una firma válida no debería fallar en la capa de auth, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

// TestWebServer_HandleWebhookDeploy_GitSourceTriggersAsyncBuild valida el flujo completo
// del build-from-source disparado por webhook: para un servicio SourceType=git, el
// handler debe responder rápido (sin esperar al build) con un buildId, y ese build debe
// quedar consultable vía /api/builds/stream.
func TestWebServer_HandleWebhookDeploy_GitSourceTriggersAsyncBuild(t *testing.T) {
	body := []byte(`{"service":"git-app","after":"abc123"}`)
	repo := mocks.NewMockConfigRepository()
	repo.Services = []domain.SavedService{{
		Name:          "git-app",
		WebhookSecret: "s3cret",
		SourceType:    "git",
		GitRepoURL:    "https://github.com/org/repo.git",
		ServerName:    "local",
	}}
	ws := NewWebServer(repo, &domain.ServerConfig{Name: "local", Host: "localhost", CloudProvider: "local"})

	mac := hmac.New(sha256.New, []byte("s3cret"))
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/deploy?service=git-app", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", sig)
	rr := httptest.NewRecorder()
	ws.handleWebhookDeploy(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Status  string `json:"status"`
		BuildID string `json:"buildId"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "building" || resp.BuildID == "" {
		t.Fatalf("respuesta inesperada: %+v", resp)
	}
	if ws.builds.Get(resp.BuildID) == nil {
		t.Fatal("el build disparado no quedó registrado en el BuildRegistry")
	}
}

// TestWebServer_HandleBuildStream_UnknownIDReturns404 valida el caso de un ID de build
// que no existe (ej. el proceso se reinició desde que se disparó el webhook).
func TestWebServer_HandleBuildStream_UnknownIDReturns404(t *testing.T) {
	ws := NewWebServer(&mockRepo{}, &domain.ServerConfig{Name: "local"})
	req := httptest.NewRequest(http.MethodGet, "/api/builds/stream?id=no-existe", nil)
	rr := httptest.NewRecorder()
	ws.handleBuildStream(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestWebServer_HandleServiceRebuild_RejectsNonGitService valida que un servicio que no
// es build-from-source no pueda usar el endpoint de rebuild manual.
func TestWebServer_HandleServiceRebuild_RejectsNonGitService(t *testing.T) {
	repo := mocks.NewMockConfigRepository()
	repo.Services = []domain.SavedService{{Name: "image-app", SourceType: "image", ServerName: "local"}}
	ws := NewWebServer(repo, &domain.ServerConfig{Name: "local", Host: "localhost", CloudProvider: "local"})

	req := httptest.NewRequest(http.MethodPost, "/api/services/rebuild?name=image-app", nil)
	rr := httptest.NewRecorder()
	ws.handleServiceRebuild(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 para un servicio que no es git, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestWebServer_HandleServiceItem_PreservesSecretsOnEmptyUpdate valida el flujo
// completo: un PUT de edición que no manda GitAccessToken/WebhookSecret (porque el
// formulario no los re-muestra) no debe borrar los que ya estaban guardados.
func TestWebServer_HandleServiceItem_PreservesSecretsOnEmptyUpdate(t *testing.T) {
	repo := mocks.NewMockConfigRepository()
	repo.Services = []domain.SavedService{{
		Name:           "git-app",
		SourceType:     "git",
		GitAccessToken: "ghp_existing_token",
		WebhookSecret:  "existing_webhook_secret",
		Domain:         "old.example.com",
		ServerName:     "local",
	}}
	ws := NewWebServer(repo, &domain.ServerConfig{Name: "local"})

	body := []byte(`{"name":"git-app","domain":"new.example.com","expose":true,"sourceType":"git"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/services/git-app", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	ws.handleServiceItem(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	saved, err := repo.GetService("git-app", "local")
	if err != nil || saved == nil {
		t.Fatalf("error leyendo servicio actualizado: %v", err)
	}
	if saved.Domain != "new.example.com" {
		t.Errorf("el campo editado (domain) no se actualizó: got %q", saved.Domain)
	}
	if saved.GitAccessToken != "ghp_existing_token" {
		t.Errorf("el GitAccessToken se borró en vez de conservarse: got %q", saved.GitAccessToken)
	}
	if saved.WebhookSecret != "existing_webhook_secret" {
		t.Errorf("el WebhookSecret se borró en vez de conservarse: got %q", saved.WebhookSecret)
	}
}

// TestWebServer_VolumeReadRoutes_RequireAuthWhenExposed valida el flujo completo a
// través del enrutador real (ws.Echo()), no solo llamando al handler directo: en modo
// expuesto y sin API key, listar/leer/descargar archivos de /opt/data debía funcionar
// sin autenticación (fuga de confidencialidad); ahora deben bloquearse igual que
// escribir/borrar, que ya estaban protegidos.
func TestWebServer_VolumeReadRoutes_RequireAuthWhenExposed(t *testing.T) {
	ws := NewWebServer(&mockRepo{}, &domain.ServerConfig{Name: "srv"})
	ws.SetExposed(true)
	ws.SetAPIKey("")
	handler := ws.Echo()

	for _, route := range []string{"/api/volumes/files", "/api/volumes/read", "/api/volumes/download"} {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, route+"?path=/opt/data", nil)
			req.RemoteAddr = "203.0.113.7:45678"
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusForbidden {
				t.Errorf("expected 403 Forbidden para %s sin autenticación, got %d: %s", route, rr.Code, rr.Body.String())
			}
		})
	}
}












