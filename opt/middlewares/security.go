package middlewares

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

var pathTraversalKeywords = []string{
	"../", "..\\",
	"%2e%2e", "%252e",
	"/etc/passwd", "/proc/self",
}

// SecurityPerimeter inspecciona parámetros de consulta y rutas para prevenir ataques comunes como Path Traversal.
func SecurityPerimeter() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			path := strings.ToLower(ctx.Request().URL.Path)
			rawQuery := strings.ToLower(ctx.Request().URL.RawQuery)

			for _, pattern := range pathTraversalKeywords {
				if strings.Contains(path, pattern) || strings.Contains(rawQuery, pattern) {
					slog.WarnContext(ctx.Request().Context(), "bloqueado intento de path traversal en seguridad perimetral",
						slog.String("path", ctx.Path()),
						slog.String("remote_ip", ctx.RealIP()),
						slog.String("pattern", pattern),
					)
					return ctx.JSON(http.StatusBadRequest, map[string]any{
						"success": false,
						"message": "petición bloqueada por reglas de seguridad perimetral",
					})
				}
			}

			return next(ctx)
		}
	}
}
