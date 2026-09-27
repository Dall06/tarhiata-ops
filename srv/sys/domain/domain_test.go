package domain

import (
	"testing"
)

func TestDomainEntities(t *testing.T) {
	tests := []struct {
		name      string
		testCheck func(t *testing.T)
	}{
		{
			name: "SavedService initialization and fields",
			testCheck: func(t *testing.T) {
				svc := SavedService{
					Name:        "test-app",
					ImageSource: "nginx:latest",
					Port:        80,
				}
				if svc.Name != "test-app" || svc.Port != 80 {
					t.Errorf("unexpected service fields: %+v", svc)
				}
			},
		},
		{
			name: "SavedDatabase initialization and fields",
			testCheck: func(t *testing.T) {
				db := SavedDatabase{
					Name:   "test-db",
					Engine: "postgres",
				}
				if db.Engine != "postgres" || db.Name != "test-db" {
					t.Errorf("unexpected database fields: %+v", db)
				}
			},
		},
		{
			name: "ServiceLink initialization and fields",
			testCheck: func(t *testing.T) {
				link := ServiceLink{
					SourceSvc:  "app-web",
					TargetSvc:  "app-db",
					EnvVarName: "DATABASE_URL",
				}
				if link.SourceSvc != "app-web" || link.TargetSvc != "app-db" {
					t.Errorf("unexpected link fields: %+v", link)
				}
			},
		},
		{
			name: "ServerConfig struct fields",
			testCheck: func(t *testing.T) {
				cfg := ServerConfig{
					Name: "prod",
					Host: "1.2.3.4",
					Port: 22,
					User: "root",
				}
				if cfg.Name != "prod" || cfg.Host != "1.2.3.4" {
					t.Errorf("unexpected ServerConfig fields: %+v", cfg)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testCheck(t)
		})
	}
}
