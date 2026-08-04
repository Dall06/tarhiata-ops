package domain

import (
	"testing"
)

func TestDomainStructs(t *testing.T) {
	svc := SavedService{
		Name:        "test-app",
		ImageSource: "nginx:latest",
		Port:        80,
	}

	if svc.Name != "test-app" {
		t.Errorf("expected Name 'test-app', got '%s'", svc.Name)
	}

	db := SavedDatabase{
		Name:   "test-db",
		Engine: "postgres",
	}

	if db.Engine != "postgres" {
		t.Errorf("expected Engine 'postgres', got '%s'", db.Engine)
	}
}
