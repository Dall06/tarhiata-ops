package ports

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// HTTPHandler define el contrato estándar para handlers HTTP de la interfaz.
type HTTPHandler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// Router define el contrato para registrar rutas en el servidor web.
type Router interface {
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	Any(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) []*echo.Route
}
