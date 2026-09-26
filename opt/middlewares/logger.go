package middlewares

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

// RequestLogger registra de forma estructurada cada petición entrante y su tiempo de respuesta.
func RequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			start := time.Now()

			err := next(ctx)

			duration := time.Since(start)
			req := ctx.Request()
			res := ctx.Response()

			attrs := []any{
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Int("status", res.Status),
				slog.Duration("duration", duration),
				slog.String("remote_ip", ctx.RealIP()),
			}

			reqCtx := req.Context()

			if err != nil {
				attrs = append(attrs, slog.String("error", err.Error()))
				slog.ErrorContext(reqCtx, "petición fallida", attrs...)
				return err
			}

			if res.Status >= 500 {
				slog.ErrorContext(reqCtx, "error interno en respuesta HTTP", attrs...)
				return nil
			}

			if res.Status >= 400 {
				slog.WarnContext(reqCtx, "error de cliente en respuesta HTTP", attrs...)
				return nil
			}

			slog.InfoContext(reqCtx, "petición completada", attrs...)
			return nil
		}
	}
}
