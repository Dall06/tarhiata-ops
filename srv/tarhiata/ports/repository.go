package ports

import "github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"

// ConfigRepository define los métodos para persistir configuraciones locales del CLI.
type ConfigRepository interface {
	// SaveServerConfig guarda la configuración principal del servidor activo.
	SaveServerConfig(config domain.ServerConfig) error
	
	// GetServerConfig obtiene la configuración del servidor activo. Retorna nil si no existe.
	GetServerConfig() (*domain.ServerConfig, error)

	// --- Catálogo de Servicios ---
	SaveService(svc domain.SavedService) error
	GetServices() ([]domain.SavedService, error)
	GetService(name string) (*domain.SavedService, error)
	DeleteService(name string) error
	
	// Close cierra la conexión a la base de datos local.
	Close() error
}
