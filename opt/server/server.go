package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/Dall06/tarhiata-ops/opt/middlewares"
	"github.com/Dall06/tarhiata-ops/pkg/exs"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

// Timeouts predeterminados para protección contra slowloris y agotamiento de descriptores.
const (
	DefaultReadTimeout     = 15 * time.Second
	DefaultWriteTimeout    = 15 * time.Second
	DefaultIdleTimeout     = 60 * time.Second
	DefaultShutdownTimeout = 10 * time.Second
)

// Config define la configuración del servidor HTTP y tiempos de espera.
type Config struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// New crea y configura una instancia de Echo con timeouts endurecidos, CORS y manejador semántico de errores.
func New(cfg Config) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = exs.EchoHTTPErrorHandler

	readTimeout := cfg.ReadTimeout
	if readTimeout <= 0 {
		readTimeout = DefaultReadTimeout
	}
	writeTimeout := cfg.WriteTimeout
	if writeTimeout <= 0 {
		writeTimeout = DefaultWriteTimeout
	}
	idleTimeout := cfg.IdleTimeout
	if idleTimeout <= 0 {
		idleTimeout = DefaultIdleTimeout
	}

	e.Server.ReadTimeout = readTimeout
	e.Server.WriteTimeout = writeTimeout
	e.Server.IdleTimeout = idleTimeout

	// Middlewares perimetrales estándar
	e.Use(echomw.Recover())
	e.Use(middlewares.RequestLogger())
	e.Use(middlewares.SecurityPerimeter())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{"*"},
	}))
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			res := c.Response()
			res.Header().Set("X-Content-Type-Options", "nosniff")
			res.Header().Set("X-Frame-Options", "DENY")
			res.Header().Set("X-XSS-Protection", "1; mode=block")
			res.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			return next(c)
		}
	})

	return e
}

// StartGracefulWithContext ejecuta el servidor Echo y bloquea hasta que el contexto es cancelado,
// ejecutando un apagado ordenado dentro del tiempo límite establecido.
func StartGracefulWithContext(ctx context.Context, e *echo.Echo, addr string, shutdownTimeout time.Duration) error {
	if shutdownTimeout <= 0 {
		shutdownTimeout = DefaultShutdownTimeout
	}

	slog.Info("iniciando servidor HTTP con apagado ordenado", "addr", addr)

	errChan := make(chan error, 1)
	go func() {
		err := e.Start(addr)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
			return
		}
		errChan <- nil
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("falló inicio del servidor: %w", err)
		}
		return nil
	case <-ctx.Done():
		slog.Info("contexto cancelado, cerrando servidor de forma ordenada")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("falló apagado ordenado del servidor: %w", err)
	}

	slog.Info("servidor detenido exitosamente sin interrumpir conexiones")
	return nil
}

// StartGraceful ejecuta el servidor Echo en una goroutine y bloquea hasta recibir una señal
// de terminación del sistema operativo (SIGINT o SIGTERM), ejecutando un apagado ordenado.
func StartGraceful(e *echo.Echo, addr string, shutdownTimeout time.Duration) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	return StartGracefulWithContext(ctx, e, addr, shutdownTimeout)
}

