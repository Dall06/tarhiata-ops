package exs

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

// HTTPErrorResponse modela la estructura JSON estándar de error emitida al cliente.
type HTTPErrorResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

const internalServerMessage = "error interno del servidor"

// ResolveStatusAndMessage resuelve el código de estado HTTP y mensaje correspondiente al error.
func ResolveStatusAndMessage(err error) (int, string) {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound, err.Error()

	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized, err.Error()

	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden, err.Error()

	case errors.Is(err, ErrConflict):
		return http.StatusConflict, err.Error()

	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest, err.Error()

	case errors.Is(err, ErrServiceUnavailable):
		return http.StatusServiceUnavailable, err.Error()

	case errors.Is(err, ErrInternal):
		return http.StatusInternalServerError, internalServerMessage

	default:
		return http.StatusInternalServerError, internalServerMessage
	}
}

// EchoHTTPErrorHandler intercepta automáticamente los errores retornados por los handlers de Echo.
func EchoHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code := http.StatusInternalServerError
	msg := internalServerMessage

	var he *echo.HTTPError
	if errors.As(err, &he) {
		code = he.Code
		m, ok := he.Message.(string)
		if ok {
			msg = m
		}
		if !ok {
			msg = http.StatusText(code)
		}
		if code >= http.StatusInternalServerError {
			slog.ErrorContext(c.Request().Context(), "error del servidor en ruta echo",
				slog.String("path", c.Path()),
				slog.String("method", c.Request().Method),
				slog.String("error", err.Error()),
			)
		}
		if errJSON := c.JSON(code, HTTPErrorResponse{Success: false, Message: msg}); errJSON != nil {
			slog.ErrorContext(c.Request().Context(), "falló serialización de respuesta de error echo", "error", errJSON)
		}
		return
	}

	code, msg = ResolveStatusAndMessage(err)
	if code >= http.StatusInternalServerError {
		slog.ErrorContext(c.Request().Context(), "error no capturado en controlador echo",
			slog.String("path", c.Path()),
			slog.String("method", c.Request().Method),
			slog.String("error", err.Error()),
		)
	}

	if errJSON := c.JSON(code, HTTPErrorResponse{Success: false, Message: msg, Data: DataOf(err)}); errJSON != nil {
		slog.ErrorContext(c.Request().Context(), "falló serialización de respuesta de error semántico", "error", errJSON)
	}
}
