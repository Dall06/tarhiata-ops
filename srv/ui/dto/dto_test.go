package dto

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

func TestDecodeServerConfig(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantErr   bool
		checkFunc func(t *testing.T, srv domain.ServerConfig)
	}{
		{
			name: "snake_case parsing with fallback keys",
			jsonInput: `{
				"name": "prod-vps",
				"host": "192.168.1.50",
				"port": 2222,
				"user": "ubuntu",
				"key_path": "/home/user/.ssh/id_rsa",
				"do_token": "dop_v1_123456",
				"vultr_token": "vultr_sec_789"
			}`,
			wantErr: false,
			checkFunc: func(t *testing.T, srv domain.ServerConfig) {
				if srv.Name != "prod-vps" || srv.Host != "192.168.1.50" || srv.Port != 2222 {
					t.Errorf("unexpected basic fields: %+v", srv)
				}
				if srv.PrivateKey != "/home/user/.ssh/id_rsa" {
					t.Errorf("expected PrivateKey '/home/user/.ssh/id_rsa', got '%s'", srv.PrivateKey)
				}
				if srv.DOAPIToken != "dop_v1_123456" {
					t.Errorf("expected DOAPIToken 'dop_v1_123456', got '%s'", srv.DOAPIToken)
				}
				if srv.VultrAPIToken != "vultr_sec_789" {
					t.Errorf("expected VultrAPIToken 'vultr_sec_789', got '%s'", srv.VultrAPIToken)
				}
			},
		},
		{
			name: "camelCase parsing with token inheritance",
			jsonInput: `{
				"name": "staging-vps",
				"host": "10.0.0.1",
				"privateKey": "-----BEGIN RSA PRIVATE KEY-----",
				"doAPIToken": "token-abc"
			}`,
			wantErr: false,
			checkFunc: func(t *testing.T, srv domain.ServerConfig) {
				if srv.PrivateKey != "-----BEGIN RSA PRIVATE KEY-----" {
					t.Errorf("expected raw private key content, got '%s'", srv.PrivateKey)
				}
				if srv.DOAPIToken != "token-abc" {
					t.Errorf("expected DOAPIToken 'token-abc', got '%s'", srv.DOAPIToken)
				}
				if srv.VultrAPIToken != "token-abc" {
					t.Errorf("expected VultrAPIToken fallback to 'token-abc', got '%s'", srv.VultrAPIToken)
				}
			},
		},
		{
			name: "private_key alias fallback",
			jsonInput: `{
				"name": "dev-box",
				"host": "127.0.0.1",
				"private_key": "/path/to/key"
			}`,
			wantErr: false,
			checkFunc: func(t *testing.T, srv domain.ServerConfig) {
				if srv.PrivateKey != "/path/to/key" {
					t.Errorf("expected PrivateKey '/path/to/key', got '%s'", srv.PrivateKey)
				}
			},
		},
		{
			name:      "invalid json syntax",
			jsonInput: `{invalid_json`,
			wantErr:   true,
			checkFunc: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := DecodeServerConfig([]byte(tt.jsonInput))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for invalid json, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected decode error: %v", err)
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, cfg)
			}
		})
	}
}

func TestDecodeSavedService(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantErr   bool
		check     func(t *testing.T, name, img, envVars, node string, ssl bool)
	}{
		{
			name: "snake_case fields",
			jsonInput: `{
				"name": "my-app",
				"image_source": "node:18-alpine",
				"enable_ssl": true,
				"pre_deploy_hook": "npm run migrate",
				"healthcheck_cmd": "curl -f http://localhost:3000/health",
				"env_vars": "PORT=3000",
				"env_file_path": "/etc/secrets/.env",
				"target_node": "worker-1"
			}`,
			wantErr: false,
			check: func(t *testing.T, name, img, envVars, node string, ssl bool) {
				if name != "my-app" || img != "node:18-alpine" || envVars != "PORT=3000" || node != "worker-1" || !ssl {
					t.Errorf("failed snake_case assertions: name=%s, img=%s, ssl=%v, node=%s", name, img, ssl, node)
				}
			},
		},
		{
			name: "camelCase fields",
			jsonInput: `{
				"name": "my-app-camel",
				"imageSource": "postgres:15",
				"enableSSL": false,
				"preDeployHook": "npx prisma db push",
				"targetNode": "manager-1"
			}`,
			wantErr: false,
			check: func(t *testing.T, name, img, envVars, node string, ssl bool) {
				if name != "my-app-camel" || img != "postgres:15" || node != "manager-1" || ssl {
					t.Errorf("failed camelCase assertions: name=%s, img=%s, ssl=%v, node=%s", name, img, ssl, node)
				}
			},
		},
		{
			name:      "malformed JSON",
			jsonInput: `{"name": `,
			wantErr:   true,
			check:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := DecodeSavedService([]byte(tt.jsonInput))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, svc.Name, svc.ImageSource, svc.EnvVars, svc.TargetNode, svc.EnableSSL)
			}
		})
	}
}

func TestDecodeSavedDatabase(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantErr   bool
		check     func(t *testing.T, name, engine, deployType, url, node string, port int)
	}{
		{
			name: "snake_case fields",
			jsonInput: `{
				"name": "main-pg",
				"engine": "postgres",
				"deploy_type": "swarm",
				"external_url": "postgres.internal",
				"internal_port": 5432,
				"volume_host_path": "/data/pgdata",
				"target_node": "worker-db"
			}`,
			wantErr: false,
			check: func(t *testing.T, name, engine, deployType, url, node string, port int) {
				if name != "main-pg" || engine != "postgres" || deployType != "swarm" || url != "postgres.internal" || port != 5432 || node != "worker-db" {
					t.Errorf("database decode mismatch: %s %s %s %d %s", name, engine, deployType, port, node)
				}
			},
		},
		{
			name: "camelCase fields",
			jsonInput: `{
				"name": "cache-redis",
				"engine": "redis",
				"deployType": "host",
				"externalURL": "redis://127.0.0.1",
				"internalPort": 6379,
				"targetNode": "standalone"
			}`,
			wantErr: false,
			check: func(t *testing.T, name, engine, deployType, url, node string, port int) {
				if name != "cache-redis" || engine != "redis" || deployType != "host" || url != "redis://127.0.0.1" || port != 6379 || node != "standalone" {
					t.Errorf("database camelCase mismatch: %s %s %d", name, engine, port)
				}
			},
		},
		{
			name:      "malformed JSON",
			jsonInput: `{"invalid": `,
			wantErr:   true,
			check:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := DecodeSavedDatabase([]byte(tt.jsonInput))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, db.Name, db.Engine, db.DeployType, db.ExternalURL, db.TargetNode, db.InternalPort)
			}
		})
	}
}

func TestDecodeServiceLink(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantErr   bool
		check     func(t *testing.T, src, target, envVar string)
	}{
		{
			name: "snake_case fields",
			jsonInput: `{
				"source_svc": "app-api",
				"target_svc": "db-pg",
				"env_var_name": "DATABASE_URL"
			}`,
			wantErr: false,
			check: func(t *testing.T, src, target, envVar string) {
				if src != "app-api" || target != "db-pg" || envVar != "DATABASE_URL" {
					t.Errorf("link decode mismatch: %s -> %s via %s", src, target, envVar)
				}
			},
		},
		{
			name: "camelCase fields",
			jsonInput: `{
				"sourceSvc": "frontend-web",
				"targetSvc": "backend-api",
				"envVarName": "API_ENDPOINT"
			}`,
			wantErr: false,
			check: func(t *testing.T, src, target, envVar string) {
				if src != "frontend-web" || target != "backend-api" || envVar != "API_ENDPOINT" {
					t.Errorf("link camelCase mismatch: %s -> %s via %s", src, target, envVar)
				}
			},
		},
		{
			name:      "malformed JSON",
			jsonInput: `{"bad": `,
			wantErr:   true,
			check:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link, err := DecodeServiceLink([]byte(tt.jsonInput))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, link.SourceSvc, link.TargetSvc, link.EnvVarName)
			}
		})
	}
}
