package dto

import (
	"testing"
)

func TestDecodeSavedService(t *testing.T) {
	snakeJSON := `{
		"name": "my-app",
		"image_source": "node:18-alpine",
		"enable_ssl": true,
		"pre_deploy_hook": "npm run migrate",
		"healthcheck_cmd": "curl -f http://localhost:3000/health",
		"env_vars": "PORT=3000"
	}`

	svc, err := DecodeSavedService([]byte(snakeJSON))
	if err != nil {
		t.Fatalf("failed to decode snake_case JSON: %v", err)
	}

	if svc.Name != "my-app" {
		t.Errorf("expected Name 'my-app', got '%s'", svc.Name)
	}
	if svc.ImageSource != "node:18-alpine" {
		t.Errorf("expected ImageSource 'node:18-alpine', got '%s'", svc.ImageSource)
	}
	if !svc.EnableSSL {
		t.Errorf("expected EnableSSL true, got false")
	}

	camelJSON := `{
		"name": "my-app-camel",
		"imageSource": "postgres:15",
		"enableSSL": false,
		"preDeployHook": "npx prisma db push"
	}`

	svcCamel, err := DecodeSavedService([]byte(camelJSON))
	if err != nil {
		t.Fatalf("failed to decode camelCase JSON: %v", err)
	}

	if svcCamel.ImageSource != "postgres:15" {
		t.Errorf("expected ImageSource 'postgres:15', got '%s'", svcCamel.ImageSource)
	}
}

func TestDecodeServiceLink(t *testing.T) {
	snakeJSON := `{
		"source_svc": "app-api",
		"target_svc": "db-pg",
		"env_var_name": "DATABASE_URL"
	}`

	link, err := DecodeServiceLink([]byte(snakeJSON))
	if err != nil {
		t.Fatalf("failed to decode ServiceLink: %v", err)
	}

	if link.SourceSvc != "app-api" || link.TargetSvc != "db-pg" || link.EnvVarName != "DATABASE_URL" {
		t.Errorf("decode ServiceLink failed: %+v", link)
	}
}
