package exs

import (
	"errors"
	"fmt"
)

// CustomError envuelve un centinela semántico con un mensaje descriptivo y un error opcional subyacente.
type CustomError struct {
	sentinel error
	msg      string
	err      error
	data     map[string]any
}

// Data devuelve la carga estructurada adjunta al error, o nil si no posee datos.
func (e *CustomError) Data() map[string]any {
	return e.data
}

// DataOf extrae el mapa de datos adjuntos al error si implementa CustomError.
func DataOf(err error) map[string]any {
	var ce *CustomError
	if errors.As(err, &ce) {
		return ce.data
	}
	return nil
}

// Error devuelve el mensaje formateado del error.
func (e *CustomError) Error() string {
	if e.msg != "" {
		return e.msg
	}
	if e.sentinel != nil {
		return e.sentinel.Error()
	}
	if e.err != nil {
		return e.err.Error()
	}
	return "error desconocido"
}

// Unwrap permite la introspección con errors.Is y errors.As.
func (e *CustomError) Unwrap() error {
	if e.err != nil {
		return e.err
	}
	return e.sentinel
}

// Is verifica coincidencia contra el error centinela o el error interno.
func (e *CustomError) Is(target error) bool {
	if errors.Is(e.sentinel, target) {
		return true
	}
	if e.err != nil && errors.Is(e.err, target) {
		return true
	}
	return false
}

func extractErr(args []any) error {
	for _, arg := range args {
		if e, ok := arg.(error); ok {
			return e
		}
	}
	return nil
}

func newCustomError(sentinel error, msg string, args ...any) *CustomError {
	formatted := msg
	if len(args) > 0 {
		formatted = fmt.Sprintf(msg, args...)
	}
	return &CustomError{
		sentinel: sentinel,
		msg:      formatted,
		err:      extractErr(args),
	}
}

// NotFound crea un error 404 estructurado.
func NotFound(msg string, args ...any) error {
	return newCustomError(ErrNotFound, msg, args...)
}

// BadRequest crea un error 400 estructurado.
func BadRequest(msg string, args ...any) error {
	return newCustomError(ErrBadRequest, msg, args...)
}

// Unauthorized crea un error 401 estructurado.
func Unauthorized(msg string, args ...any) error {
	return newCustomError(ErrUnauthorized, msg, args...)
}

// Forbidden crea un error 403 estructurado.
func Forbidden(msg string, args ...any) error {
	return newCustomError(ErrForbidden, msg, args...)
}

// Conflict crea un error 409 estructurado.
func Conflict(msg string, args ...any) error {
	return newCustomError(ErrConflict, msg, args...)
}

// Internal crea un error 500 estructurado.
func Internal(msg string, args ...any) error {
	return newCustomError(ErrInternal, msg, args...)
}

// ServiceUnavailable crea un error 503 estructurado.
func ServiceUnavailable(msg string, args ...any) error {
	return newCustomError(ErrServiceUnavailable, msg, args...)
}

// Wrap envuelve un error existente con un centinela semántico y un mensaje.
func Wrap(sentinel error, err error, msg string) error {
	return &CustomError{
		sentinel: sentinel,
		msg:      msg,
		err:      err,
	}
}
