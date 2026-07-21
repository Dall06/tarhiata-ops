package usecases

import (
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
)

func TestGenerateCompose(t *testing.T) {
	uc := &DeployerUseCase{}

	tests := []struct {
		name        string
		service     domain.CustomService
		config      DeployConfig
		expectItems []string
	}{
		{
			name: "Servicio básico sin exposición pública",
			service: domain.CustomService{
				Name: "api-interna",
			},
			config: DeployConfig{
				ImageSource: "nginx:latest",
				Expose:      false,
			},
			expectItems: []string{
				"image: nginx:latest",
				"api-interna:",
				"networks:",
				"- tarhiata_public",
			},
		},
		{
			name: "Servicio expuesto con dominio y SSL",
			service: domain.CustomService{
				Name: "api-publica",
			},
			config: DeployConfig{
				ImageSource: "api:v1",
				Expose:      true,
				Domain:      "api.gymbro.com",
				Port:        3000,
				EnableSSL:   true,
			},
			expectItems: []string{
				"traefik.enable=true",
				"Host(`api.gymbro.com`)",
				"loadbalancer.server.port=3000",
				"tls.certresolver=leresolver",
			},
		},
		{
			name: "Servicio expuesto por PathPrefix con Stripprefix",
			service: domain.CustomService{
				Name: "test-api",
			},
			config: DeployConfig{
				ImageSource: "helloworld",
				Expose:      true,
				Port:        8080,
			},
			expectItems: []string{
				"PathPrefix(`/test-api`)",
				"test-api-strip.stripprefix.prefixes=/test-api",
				"loadbalancer.server.port=8080",
			},
		},
		{
			name: "Servicio con Mounts y Healthcheck",
			service: domain.CustomService{
				Name: "db-local",
				Mounts: []domain.ServiceMount{
					{LocalPath: "/tmp/config.json", DestPath: "/app/config.json"},
				},
			},
			config: DeployConfig{
				ImageSource:    "postgres:15",
				HealthcheckCmd: "pg_isready -U user",
			},
			expectItems: []string{
				"healthcheck:",
				"- \"pg_isready -U user\"",
				"volumes:",
				"- ./configs/config.json:/app/config.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.generateCompose(tt.service, tt.config)
			for _, item := range tt.expectItems {
				if !strings.Contains(result, item) {
					t.Errorf("Resultado esperado '%s' no encontrado en el yaml generado:\n%s", item, result)
				}
			}
		})
	}
}
