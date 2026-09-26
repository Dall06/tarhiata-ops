package exs

import (
	"errors"
)

var (
	// ErrNotFound indica que el recurso solicitado no existe (HTTP 404).
	ErrNotFound = errors.New("recurso no encontrado")

	// ErrInternal indica un error no recuperable del servidor o backend (HTTP 500).
	ErrInternal = errors.New("error interno del servidor")

	// ErrBadRequest indica que los parámetros o el cuerpo de la petición son inválidos (HTTP 400).
	ErrBadRequest = errors.New("petición inválida")

	// ErrUnauthorized indica que la petición carece de credenciales válidas (HTTP 401).
	ErrUnauthorized = errors.New("no autorizado")

	// ErrForbidden indica que las credenciales no poseen los permisos necesarios (HTTP 403).
	ErrForbidden = errors.New("acceso denegado")

	// ErrConflict indica un conflicto de estado o duplicidad (HTTP 409).
	ErrConflict = errors.New("conflicto de recurso")

	// ErrServiceUnavailable indica que un servicio dependiente o nodo no está disponible (HTTP 503).
	ErrServiceUnavailable = errors.New("servicio no disponible")
)
