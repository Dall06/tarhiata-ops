package usecases

import (
	"github.com/Dall06/tarhiata-ops/pkg/secutil"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

// IsLocal determina si una configuración de servidor apunta al equipo local.
func IsLocal(cfg domain.ServerConfig) bool {
	return secutil.IsLocalHost(cfg.Host, cfg.CloudProvider)
}
