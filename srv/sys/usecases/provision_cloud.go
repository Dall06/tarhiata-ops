package usecases

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Dall06/tarhiata-ops/opt/cloud"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type ProvisionCloudServerUseCase struct {
	configRepo         ports.ConfigRepository
	connectUC          ports.ConnectServerUseCase
	provisionerFactory func(provider, workspace string) ports.Provisioner
	maxSSHRetries      int
	retryDelay         time.Duration
}

func NewProvisionCloudServerUseCase(repo ports.ConfigRepository, connectUC ports.ConnectServerUseCase) *ProvisionCloudServerUseCase {
	return &ProvisionCloudServerUseCase{
		configRepo: repo,
		connectUC:  connectUC,
		provisionerFactory: func(provider, workspace string) ports.Provisioner {
			return cloud.NewProvisioner(provider, workspace)
		},
		maxSSHRetries: 24,
		retryDelay:    5 * time.Second,
	}
}

// WithProvisionerFactory permite inyectar un factory mock para pruebas unitarias.
func (uc *ProvisionCloudServerUseCase) WithProvisionerFactory(f func(provider, workspace string) ports.Provisioner) *ProvisionCloudServerUseCase {
	uc.provisionerFactory = f
	return uc
}

// WithRetryConfig ajusta reintentos para tests rápidos.
func (uc *ProvisionCloudServerUseCase) WithRetryConfig(maxRetries int, delay time.Duration) *ProvisionCloudServerUseCase {
	uc.maxSSHRetries = maxRetries
	uc.retryDelay = delay
	return uc
}

func (uc *ProvisionCloudServerUseCase) Execute(req ports.ProvisionCloudRequest) (*domain.ConnectionResult, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	req.APIToken = strings.TrimSpace(req.APIToken)
	req.Region = strings.TrimSpace(req.Region)
	req.Plan = strings.TrimSpace(req.Plan)

	if req.APIToken == "" {
		return nil, fmt.Errorf("se requiere un Token de API del proveedor de nube")
	}

	if req.Provider == "" {
		req.Provider = "vultr"
	}

	if req.Name == "" {
		req.Name = fmt.Sprintf("%s-%d", req.Provider, time.Now().Unix())
	}

	if req.Region == "" {
		if req.Provider == "digitalocean" {
			req.Region = "nyc1"
		}
		if req.Provider == "vultr" {
			req.Region = "mex"
		}
	}

	// 1. Validar que no exista un servidor con ese nombre
	existing, err := uc.configRepo.GetServerConfigByName(req.Name)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("ya existe un servidor con el nombre '%s' en el catálogo", req.Name)
	}

	// 2. Preparar workspace de IaC
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("no se pudo resolver el directorio home del usuario: %w", err)
	}

	workspace := filepath.Join(homeDir, ".config", "tarhiata", "terraform", req.Provider+"_"+req.Name)
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return nil, fmt.Errorf("error creando workspace de aprovisionamiento: %w", err)
	}

	// 3. Aprovisionar infraestructura con el motor OpenTofu/Terraform
	provisioner := uc.provisionerFactory(req.Provider, workspace)
	provRes, err := provisioner.ProvisionNode(req.APIToken, req.Name, req.Region, req.Plan)
	if err != nil {
		return nil, fmt.Errorf("error aprovisionando nodo en %s: %w", req.Provider, err)
	}

	newIP := strings.TrimSpace(provRes.PublicIP)
	privKeyContent := provRes.PrivateKey
	if newIP == "" {
		return nil, fmt.Errorf("el aprovisionador no devolvió una IP pública válida")
	}

	// 4. Guardar llave privada en ~/.ssh/ si fue autogenerada por el módulo
	keyPath := filepath.Join(homeDir, ".ssh", "tarhiata_"+req.Name+".pem")
	if privKeyContent != "" {
		sshDir := filepath.Join(homeDir, ".ssh")
		if err := os.MkdirAll(sshDir, 0700); err != nil {
			return nil, fmt.Errorf("error asegurando directorio .ssh: %w", err)
		}
		if err := os.WriteFile(keyPath, []byte(privKeyContent), 0600); err != nil {
			return nil, fmt.Errorf("error guardando llave privada SSH: %w", err)
		}
	}

	// 5. Registrar servidor en catálogo
	cfg := domain.ServerConfig{
		Name:          req.Name,
		Host:          newIP,
		Port:          22,
		User:          "root",
		PrivateKey:    keyPath,
		CloudProvider: req.Provider,
		IsActive:      req.SetAsActive,
	}

	if req.Provider == "digitalocean" {
		cfg.DOAPIToken = req.APIToken
	}
	if req.Provider == "vultr" {
		cfg.VultrAPIToken = req.APIToken
	}

	if err := uc.configRepo.SaveServerConfig(cfg); err != nil {
		return nil, fmt.Errorf("error guardando configuración en catálogo: %w", err)
	}

	if req.SetAsActive {
		if err := uc.configRepo.SetActiveServerConfig(cfg.Name); err != nil {
			slog.Warn("falló al establecer nuevo servidor como activo", "server", cfg.Name, "error", err)
		}
	}

	// 6. Probar conexión esperando arranque de la VM
	var lastResult *domain.ConnectionResult
	for i := 0; i < uc.maxSSHRetries; i++ {
		res, err := uc.connectUC.Execute(cfg)
		if err != nil {
			slog.Debug("intento de conexión a nueva VM falló", "retry", i, "error", err)
		}
		if res != nil {
			lastResult = res
			if res.Connected {
				return res, nil
			}
		}
		time.Sleep(uc.retryDelay)
	}

	if lastResult != nil {
		return lastResult, nil
	}

	return &domain.ConnectionResult{
		Name:       cfg.Name,
		Connected:  false,
		TargetHost: cfg.Host,
		Message:    fmt.Sprintf("Servidor '%s' registrado con IP %s, pero aún no responde SSH.", cfg.Name, cfg.Host),
	}, nil
}
