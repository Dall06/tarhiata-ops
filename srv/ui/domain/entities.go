package domain

import (
	"time"

	sysdomain "github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

// CacheItem representa un elemento cacheado en memoria con tiempo de expiración.
type CacheItem struct {
	Data      interface{}
	ExpiresAt time.Time
}

// UIConfig representa las opciones de configuración específicas de la interfaz de usuario.
type UIConfig struct {
	Port      string
	APIKey    string
	IsExposed bool
}

// ServerState representa el estado en vivo de un servidor seleccionado.
type ServerState struct {
	Config    sysdomain.ServerConfig
	IsLocal   bool
	IsOnline  bool
	LastCheck time.Time
}
