package handlers

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Dall06/tarhiata-ops/opt/banner"
	"github.com/Dall06/tarhiata-ops/opt/cloud"
	"github.com/Dall06/tarhiata-ops/opt/server"
	"github.com/Dall06/tarhiata-ops/pkg/apiclient"
	"github.com/Dall06/tarhiata-ops/pkg/dockerutil"
	"github.com/Dall06/tarhiata-ops/pkg/exs"
	"github.com/Dall06/tarhiata-ops/pkg/httputil"
	"github.com/Dall06/tarhiata-ops/pkg/jsutil"
	"github.com/Dall06/tarhiata-ops/pkg/modal"
	"github.com/Dall06/tarhiata-ops/pkg/notify"
	"github.com/Dall06/tarhiata-ops/pkg/osterminal"
	"github.com/Dall06/tarhiata-ops/pkg/store"
	"github.com/Dall06/tarhiata-ops/pkg/toast"
	"github.com/Dall06/tarhiata-ops/pkg/validator"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
	"github.com/Dall06/tarhiata-ops/srv/sys/repositories"
	"github.com/Dall06/tarhiata-ops/srv/sys/usecases"
	"github.com/Dall06/tarhiata-ops/srv/ui/components/alerts"
	"github.com/Dall06/tarhiata-ops/srv/ui/components/audit"
	"github.com/Dall06/tarhiata-ops/srv/ui/components/databases"
	"github.com/Dall06/tarhiata-ops/srv/ui/components/env"
	"github.com/Dall06/tarhiata-ops/srv/ui/components/fleet"
	"github.com/Dall06/tarhiata-ops/srv/ui/components/hardware"
	"github.com/Dall06/tarhiata-ops/srv/ui/components/logs"
	"github.com/Dall06/tarhiata-ops/srv/ui/components/nodes"
	"github.com/Dall06/tarhiata-ops/srv/ui/components/services"
	"github.com/Dall06/tarhiata-ops/srv/ui/components/spotlight"
	"github.com/Dall06/tarhiata-ops/srv/ui/components/ssl"
	"github.com/Dall06/tarhiata-ops/srv/ui/components/telemetry"
	"github.com/Dall06/tarhiata-ops/srv/ui/components/terminal"
	"github.com/Dall06/tarhiata-ops/srv/ui/components/topology"
	"github.com/Dall06/tarhiata-ops/srv/ui/components/volumes"
	dto "github.com/Dall06/tarhiata-ops/srv/ui/domain"
	"github.com/Dall06/tarhiata-ops/srv/ui/views/public"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

type cacheItem struct {
	data      interface{}
	expiresAt time.Time
}

type WebServer struct {
	mu        sync.RWMutex
	repo      ports.ConfigRepository
	config    *domain.ServerConfig
	apiKey    string // Clave API opcional para proteger endpoints destructivos
	isExposed bool   // Indica si el servidor escucha en 0.0.0.0
	cacheMu   sync.RWMutex
	cache     map[string]cacheItem
	echo      *echo.Echo
	builds    *usecases.BuildRegistry
}

func isLocalConfig(cfg *domain.ServerConfig) bool {
	if cfg == nil {
		return false
	}
	return usecases.IsLocal(*cfg)
}

func isLocalServer(cfg domain.ServerConfig) bool {
	return usecases.IsLocal(cfg)
}

func (w *WebServer) getConfig() *domain.ServerConfig {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.config
}

func (w *WebServer) setConfig(cfg *domain.ServerConfig) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.config = cfg
}

// resolveTargetServer obtiene la configuración del servidor solicitado por nombre o query param,
// recurriendo a w.getConfig() si no se especifica o no se encuentra en el repositorio.
func (w *WebServer) resolveTargetServer(serverName string) *domain.ServerConfig {
	serverName = strings.TrimSpace(serverName)
	if serverName != "" && w.repo != nil {
		found, err := w.repo.GetServerConfigByName(serverName)
		if err != nil {
			slog.Debug("web_server: servidor no encontrado por nombre, recurriendo a default", "server", serverName, "error", err)
		}
		if found != nil && (found.Host != "" || isLocalConfig(found)) {
			return found
		}
	}
	return w.getConfig()
}


// SetExposed define si el servidor opera en modo de red expuesto (0.0.0.0).
func (w *WebServer) SetExposed(exposed bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.isExposed = exposed
}

// SetAPIKey permite configurar o actualizar la clave de API dinámicamente.
func (w *WebServer) SetAPIKey(key string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.apiKey = key
}

func (w *WebServer) getCache(key string) (interface{}, bool) {
	w.cacheMu.RLock()
	defer w.cacheMu.RUnlock()
	if entry, ok := w.cache[key]; ok {
		if time.Now().Before(entry.expiresAt) {
			return entry.data, true
		}
	}
	return nil, false
}

func (w *WebServer) setCache(key string, data interface{}, ttl time.Duration) {
	w.cacheMu.Lock()
	defer w.cacheMu.Unlock()
	if w.cache == nil {
		w.cache = make(map[string]cacheItem)
	}
	w.cache[key] = cacheItem{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
}

func NewWebServer(repo ports.ConfigRepository, config *domain.ServerConfig) *WebServer {
	apiKey := ""
	if k, ok := os.LookupEnv("TARHIATA_API_KEY"); ok && k != "" {
		apiKey = k
	}
	isExposed := os.Getenv("TARHIATA_EXPOSE") == "true" || os.Getenv("TARHIATA_HOST") == "0.0.0.0"
	return &WebServer{
		repo:      repo,
		config:    config,
		apiKey:    apiKey,
		isExposed: isExposed,
		cache:     make(map[string]cacheItem),
		builds:    usecases.NewBuildRegistry(),
	}
}

// isLoopback determina si una dirección IP o host remoto corresponde a localhost.
func isLoopback(remoteAddr string) bool {
	if remoteAddr == "" {
		return true
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsLoopback()
	}
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// isAuthorized valida si la petición cuenta con credenciales o permisos para operar.
func (w *WebServer) isAuthorized(req *http.Request) bool {
	w.mu.RLock()
	apiKey := w.apiKey
	isExposed := w.isExposed
	w.mu.RUnlock()

	if apiKey != "" {
		reqKey := req.Header.Get("X-API-Key")
		if reqKey == "" {
			reqKey = req.URL.Query().Get("api_key")
		}
		if reqKey == "" {
			reqKey = req.URL.Query().Get("key")
		}
		if reqKey == "" {
			authHeader := req.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				reqKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}
		return subtle.ConstantTimeCompare([]byte(reqKey), []byte(apiKey)) == 1
	}
	if isExposed && !isLoopback(req.RemoteAddr) {
		return false
	}
	return true
}

// localAuthMiddleware verifica que las peticiones a endpoints críticos provengan de localhost
// o cuenten con una API key válida si el servidor está expuesto en la red.
func (w *WebServer) localAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		if !w.isAuthorized(req) {
			w.mu.RLock()
			hasKey := w.apiKey != ""
			w.mu.RUnlock()

			if hasKey {
				http.Error(rw, `{"error":"Unauthorized: API key requerida o inválida"}`, http.StatusUnauthorized)
				return
			}
			http.Error(rw, `{"error":"Forbidden: Este endpoint crítico requiere TARHIATA_API_KEY cuando el servidor está expuesto en la red"}`, http.StatusForbidden)
			return
		}
		next(rw, req)
	}
}

// Echo inicializa y retorna el motor de enrutamiento y middlewares de Echo v4.
func (w *WebServer) Echo() *echo.Echo {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.echo != nil {
		return w.echo
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = exs.EchoHTTPErrorHandler

	// Middlewares globales de Echo
	e.Use(echomw.Recover())
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

	echoAuth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !w.isAuthorized(c.Request()) {
				w.mu.RLock()
				hasKey := w.apiKey != ""
				w.mu.RUnlock()

				if hasKey {
					return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized: API key requerida o inválida"})
				}
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: Este endpoint crítico requiere TARHIATA_API_KEY cuando el servidor está expuesto en la red"})
			}
			return next(c)
		}
	}

	// Mapeo de módulos frontend distribuidos en pkg/ y srv/ui/components/
	embeddedModules := map[string][]byte{
		"/pkg/apiclient/api.js":              apiclient.JSContent,
		"/pkg/store/state.js":                store.JSContent,
		"/pkg/toast/toast.js":                toast.JSContent,
		"/pkg/modal/modal.js":                modal.JSContent,
		"/pkg/notify/notify.js":              notify.JSContent,
		"/pkg/jsutil/utils.js":               jsutil.JSContent,
		"/components/fleet/fleet.js":         fleet.JSContent,
		"/components/alerts/alerts.js":       alerts.JSContent,
		"/components/audit/audit.js":         audit.JSContent,
		"/components/telemetry/telemetry.js": telemetry.JSContent,
		"/components/services/services.js":   services.JSContent,
		"/components/databases/databases.js": databases.JSContent,
		"/components/nodes/nodes.js":         nodes.JSContent,
		"/components/topology/topology.js":   topology.JSContent,
		"/components/hardware/hardware.js":   hardware.JSContent,
		"/components/spotlight/spotlight.js": spotlight.JSContent,
		"/components/terminal/terminal.js":   terminal.JSContent,
		"/components/env/env.js":             env.JSContent,
		"/components/volumes/volumes.js":     volumes.JSContent,
		"/components/logs/logs.js":           logs.JSContent,
		"/components/ssl/ssl.js":             ssl.JSContent,
	}

	serveModule := func(c echo.Context) error {
		reqPath := c.Request().URL.Path
		localPath := strings.TrimPrefix(reqPath, "/")
		if strings.HasPrefix(localPath, "components/") {
			localPath = filepath.Join("srv/ui", localPath)
		}
		mimeType := "application/javascript; charset=utf-8"
		if strings.HasSuffix(reqPath, ".html") {
			mimeType = "text/html; charset=utf-8"
		} else if strings.HasSuffix(reqPath, ".css") {
			mimeType = "text/css; charset=utf-8"
		} else if strings.HasSuffix(reqPath, ".json") {
			mimeType = "application/json; charset=utf-8"
		}
		if data, err := os.ReadFile(localPath); err == nil {
			c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			return c.Blob(http.StatusOK, mimeType, data)
		}
		if data, ok := embeddedModules[reqPath]; ok {
			c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			return c.Blob(http.StatusOK, mimeType, data)
		}
		return c.NoContent(http.StatusNotFound)
	}

	e.GET("/pkg/*", serveModule)
	e.GET("/components/*", serveModule)

	// Archivos estáticos y Single Page Application
	var rootFS http.FileSystem = http.FS(public.FS)
	if fi, err := os.Stat("srv/ui/views/public/index.html"); err == nil && !fi.IsDir() {
		rootFS = http.Dir("srv/ui/views/public")
	}
	fileServer := http.FileServer(rootFS)
	e.GET("/*", func(c echo.Context) error {
		c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Response().Header().Set("Pragma", "no-cache")
		c.Response().Header().Set("Expires", "0")
		fileServer.ServeHTTP(c.Response(), c.Request())
		return nil
	})

	// API REST Controllers vía Echo
	e.Any("/api/status", echo.WrapHandler(http.HandlerFunc(w.handleStatus)))
	e.Any("/api/dashboard", echo.WrapHandler(http.HandlerFunc(w.handleStatus)))
	e.Any("/api/services", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleServices))))
	e.Any("/api/services/rollback", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleServiceRollback))))
	e.Any("/api/services/restart", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleServiceRestart))))
	e.Any("/api/databases/restart", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleServiceRestart))))
	e.Any("/api/services/*", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleServiceItem))))
	e.Any("/api/deploy", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleServices))))
	e.Any("/api/deploy-service", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleServices))))
	e.Any("/api/databases", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleDatabases))))
	e.Any("/api/databases/*", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleDatabaseItem))))
	e.Any("/api/deploy-db", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleDatabases))))
	e.Any("/api/config", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleConfig))))
	e.Any("/api/config/test", echo.WrapHandler(http.HandlerFunc(w.handleConnect)))
	e.Any("/api/connect", echo.WrapHandler(http.HandlerFunc(w.handleConnect)))
	e.Any("/api/connect/all", echo.WrapHandler(http.HandlerFunc(w.handleConnectAll)))
	e.Any("/api/servers", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleServers))))
	e.Any("/api/servers/active", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleSetActiveServer))))
	e.Any("/api/servers/provision", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleProvisionServer))))
	e.Any("/api/servers/open-terminal", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleOpenTerminal))))
	e.Any("/api/servers/terminal", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleTerminalExec))))
	e.Any("/api/host/metrics", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleHostMetrics))))
	e.Any("/api/host/services", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleHostServices))))
	e.Any("/api/host/inspect", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleHostInspect))))
	e.Any("/api/host/devices", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleHostDevices))))
	e.Any("/api/host/security", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleHostSecurity))))
	e.Any("/api/system/report", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleSystemReport))))
	e.Any("/api/swarm/status", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleSwarmStatus))))
	e.Any("/api/bootstrap", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleBootstrap))))
	e.Any("/api/create-vm-bootstrap", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleCreateVMBootstrap))))
	e.Any("/api/workers", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleWorkerProvision))))
	e.Any("/api/provision-worker", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleWorkerProvision))))
	e.Any("/api/swarm/provision-worker", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleWorkerProvision))))
	e.Any("/api/observability", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleObservability))))
	e.Any("/api/update", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleServerUpdate))))
	e.Any("/api/bootstrap-master", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleBootstrapMaster))))
	e.Any("/api/previews", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handlePreviewEnvs))))
	e.Any("/api/registries", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleRegistries))))
	e.Any("/api/migrations", echo.WrapHandler(http.HandlerFunc(w.handleMigrations)))
	e.Any("/api/migrations/file", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleMigrationFile))))
	e.Any("/api/migrations/run", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleRunMigrations))))
	e.Any("/api/observability/metrics", echo.WrapHandler(http.HandlerFunc(w.handleObservabilityMetrics)))
	e.Any("/api/backups", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleBackups))))
	e.Any("/api/databases/backup", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleBackups))))
	e.Any("/api/backups/restore", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleRestoreBackup))))
	e.Any("/api/backups/download", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleDownloadBackup))))
	e.Any("/api/env", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleEnvVars))))
	e.Any("/api/env/export", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleExportEnvVars))))
	e.Any("/api/volumes", echo.WrapHandler(http.HandlerFunc(w.handleVolumes)))
	e.Any("/api/volumes/files", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleVolumeFiles))))
	e.Any("/api/volumes/read", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleVolumeRead))))
	e.Any("/api/volumes/write", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleVolumeWrite))))
	e.Any("/api/volumes/download", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleVolumeDownload))))
	e.Any("/api/volumes/upload", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleVolumeUpload))))
	e.Any("/api/volumes/delete", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleVolumeDelete))))
	e.Any("/api/volumes/mkdir", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleVolumeMkdir))))
	e.Any("/api/ssl/inspect", echo.WrapHandler(http.HandlerFunc(w.handleSSLInspect)))
	e.Any("/api/ssl/reload", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleSSLReload))))
	e.Any("/api/maintenance/toggle", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleMaintenanceToggle))))
	e.Any("/api/domains", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleCustomDomains))))
	e.Any("/api/dns/check", echo.WrapHandler(http.HandlerFunc(w.handleDNSCheck)))
	e.Any("/api/prune", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handlePrune))))
	e.Any("/api/tools/prune", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handlePrune))))
	e.Any("/api/tools/restart-traefik", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleRestartTraefik))))
	e.Any("/api/tools/repair-traefik", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleRepairTraefik))))
	e.Any("/api/topology", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleTopology))))
	e.Any("/api/links", echo.WrapHandler(http.HandlerFunc(w.handleLinks)))
	e.Any("/api/nodes", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleNodes))))
	e.Any("/api/nodes/join-token", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleNodeJoinToken))))
	e.Any("/api/nodes/update", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleNodeUpdate))))
	e.Any("/api/nodes/labels", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleNodeLabels))))
	e.Any("/api/terminal/exec", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleTerminalExec))))
	e.Any("/api/logs", echo.WrapHandler(http.HandlerFunc(w.handleLogs)))
	e.Any("/api/audit-logs", echo.WrapHandler(http.HandlerFunc(w.handleAuditLogs)))
	e.Any("/api/settings/alerts", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleAlertSettings))))
	e.Any("/api/alerts/test", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleAlertTest))))
	e.Any("/api/services/history", echo.WrapHandler(http.HandlerFunc(w.handleServiceHistory)))
	e.Any("/api/services/rollback-version", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleServiceRollbackToVersion))))
	e.Any("/api/ssl/certificates", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleSSLCertificates))))
	e.Any("/api/nodes/drain", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleNodeDrain))))
	e.Any("/api/nodes/activate", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleNodeActivate))))
	e.Any("/api/webhooks/deploy", echo.WrapHandler(http.HandlerFunc(w.handleWebhookDeploy)))
	e.Any("/api/services/rebuild", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleServiceRebuild))))
	e.Any("/api/builds/stream", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleBuildStream))))
	e.Any("/api/stats", echo.WrapHandler(http.HandlerFunc(w.handleContainerStats)))
	e.Any("/api/databases/health", echo.WrapHandler(http.HandlerFunc(w.handleDBHealth)))
	e.Any("/api/vultr/plans", echo.WrapHandler(http.HandlerFunc(w.handleVultrPlans)))
	e.Any("/api/vultr/regions", echo.WrapHandler(http.HandlerFunc(w.handleVultrRegions)))
	e.Any("/api/sync", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleSyncState))))
	e.Any("/api/ssh-keys", echoAuth(echo.WrapHandler(http.HandlerFunc(w.handleSSHKeys))))

	w.echo = e
	return e
}

// ServeHTTP implementa http.Handler enlazándolo directamente al enrutador de Echo.
func (w *WebServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	w.Echo().ServeHTTP(rw, req)
}

func (w *WebServer) Start(port int) error {
	bindHost := "127.0.0.1"
	if os.Getenv("TARHIATA_EXPOSE") == "true" || os.Getenv("TARHIATA_HOST") == "0.0.0.0" {
		bindHost = "0.0.0.0"
		w.SetExposed(true)
		fmt.Println("⚠️  [ADVERTENCIA DE SEGURIDAD] Modo de exposición a la red activo (0.0.0.0).")
		fmt.Println("   Cualquier equipo con acceso a su red podrá acceder a este panel de administración.")
		if w.apiKey == "" {
			fmt.Println("   ℹ️  Recomendación: Configure TARHIATA_API_KEY para proteger endpoints de terminal y mutaciones.")
		}
	}
	if bindHost == "127.0.0.1" {
		w.SetExposed(false)
		fmt.Println("🔒 [SEGURIDAD] Servidor bloqueado para acceso local exclusivo (127.0.0.1).")
		fmt.Println("   Para exponer el panel a la red, inicie con: tarhiata --expose o TARHIATA_EXPOSE=true")
	}

	url := fmt.Sprintf("http://localhost:%d", port)
	banner.PrintServerBanner(port)
	go openBrowser(url)

	return server.StartGraceful(w.Echo(), fmt.Sprintf("%s:%d", bindHost, port), server.DefaultShutdownTimeout)
}

func isDockerServiceMatch(targetName string, liveMap map[string]bool) bool {
	if targetName == "" {
		return false
	}
	targetLower := strings.ToLower(targetName)
	for liveName := range liveMap {
		liveLower := strings.ToLower(liveName)
		if liveLower == targetLower ||
			strings.HasPrefix(liveLower, targetLower+"_") ||
			strings.HasSuffix(liveLower, "_"+targetLower) ||
			strings.HasPrefix(liveLower, "tarhiata_"+targetLower) ||
			strings.HasPrefix(liveLower, "tarhiata-db-"+targetLower) ||
			strings.Contains(liveLower, targetLower) {
			return true
		}
	}
	return false
}

func (w *WebServer) handleStatus(rw http.ResponseWriter, req *http.Request) {
	cfg, cfgErr := w.repo.GetServerConfig()
	if cfgErr != nil {
		slog.Warn("web_server: error obteniendo server config en handleStatus", "error", cfgErr)
	}
	serverName := ""
	if cfg != nil {
		serverName = cfg.Name
	}

	services, errSvc := w.repo.GetServices(serverName)
	databases, errDB := w.repo.GetDatabases(serverName)

	if errSvc != nil {
		http.Error(rw, fmt.Sprintf("Error leyendo servicios: %v", errSvc), http.StatusInternalServerError)
		return
	}
	if errDB != nil {
		http.Error(rw, fmt.Sprintf("Error leyendo bases de datos: %v", errDB), http.StatusInternalServerError)
		return
	}

	if services == nil { services = []domain.SavedService{} }
	if databases == nil { databases = []domain.SavedDatabase{} }

	// Preservar siempre todos los servicios y bases de datos guardados localmente
	isOnline := false
	activeServices := make([]domain.SavedService, len(services))
	copy(activeServices, services)
	activeDatabases := make([]domain.SavedDatabase, len(databases))
	copy(activeDatabases, databases)

	if cfg != nil && cfg.Host != "" {
		sshExec := repositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(*cfg); err == nil {
			isOnline = true

			// Si es una PC nueva o migrada (0 servicios/BDs en local), importar el catálogo y las relaciones del VPS
			if len(services) == 0 && len(databases) == 0 {
				syncUC := usecases.NewSyncClusterStateUseCase(w.repo, sshExec)
				if dump, err := syncUC.ImportStateFromRemote(serverName); err == nil && dump != nil {
					var errSvcReload, errDbReload error
					services, errSvcReload = w.repo.GetServices(serverName)
					if errSvcReload != nil {
						slog.Warn("falló recargar servicios tras importar estado", "error", errSvcReload)
					}
					databases, errDbReload = w.repo.GetDatabases(serverName)
					if errDbReload != nil {
						slog.Warn("falló recargar bases de datos tras importar estado", "error", errDbReload)
					}
					activeServices = append([]domain.SavedService{}, services...)
					activeDatabases = append([]domain.SavedDatabase{}, databases...)
				}
			}

			res, err := sshExec.RunCommand("docker service ls --format '{{.Name}}' && docker ps --format '{{.Names}}'")
			if clErr := sshExec.Close(); clErr != nil {
				slog.Warn("web_server: error cerrando sshExec en handleStatus", "error", clErr)
			}

			if err == nil && res != nil && res.Output != "" {
				liveMap := make(map[string]bool)
				for _, line := range strings.Split(res.Output, "\n") {
					t := strings.TrimSpace(line)
					if t != "" {
						liveMap[t] = true
					}
				}

				// Auto-descubrimiento de servicios creados externamente en Docker Swarm
				for liveName := range liveMap {
					if strings.HasPrefix(liveName, "tarhiata_proxy") || strings.HasPrefix(liveName, "tarhiata_obs") {
						continue
					}
					cleanName := liveName
					if idx := strings.Index(liveName, "_"); idx != -1 {
						cleanName = liveName[:idx]
					}
					alreadyKnown := false
					for _, s := range activeServices {
						if isDockerServiceMatch(s.Name, map[string]bool{liveName: true}) {
							alreadyKnown = true
							break
						}
					}
					for _, d := range activeDatabases {
						if isDockerServiceMatch(d.Name, map[string]bool{liveName: true}) {
							alreadyKnown = true
							break
						}
					}
					if !alreadyKnown {
						discovered := domain.SavedService{
							Name:        cleanName,
							ImageSource: "docker-swarm-live",
							Port:        80,
							Expose:      true,
							TargetNode:  cfg.Host,
						}
						if err := w.repo.SaveService(discovered); err != nil {
							slog.Warn("falló persistencia de servicio descubierto", "service", discovered.Name, "error", err)
						}
						activeServices = append(activeServices, discovered)
					}
				}
			}
		}
	}

	isLocalTarget := false
	if isLocalConfig(cfg) {
		isLocalTarget = true
	}

	servers, srvErr := w.repo.GetAllServerConfigs()
	if srvErr != nil {
		slog.Warn("web_server: error obteniendo lista de servidores en handleStatus", "error", srvErr)
		servers = []domain.ServerConfig{}
	}

	resp := map[string]interface{}{
		"config":    cfg,
		"servers":   servers,
		"services":  activeServices,
		"databases": activeDatabases,
		"isOnline":  isOnline,
		"isLocal":   isLocalTarget,
	}
	jsonResponse(rw, resp)
}

func (w *WebServer) handleConfig(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	cfg, err := dto.DecodeServerConfig(body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	if isLocalServer(cfg) {
		cfg.CloudProvider = "local"
		cfg.Host = "localhost"
	}
	if !isLocalServer(cfg) && cfg.CloudProvider == "" {
		cfg.CloudProvider = "vps-direct"
	}
	if err := w.repo.SaveServerConfig(cfg); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	w.setConfig(&cfg)
	jsonResponse(rw, map[string]string{"status": "ok"})
}

func (w *WebServer) handleConnect(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	var targetCfg domain.ServerConfig
	if len(body) > 0 && strings.TrimSpace(string(body)) != "{}" {
		decoded, err := dto.DecodeServerConfig(body)
		if err == nil {
			targetCfg = decoded
		}
	}

	if targetCfg.Host == "" && !isLocalServer(targetCfg) {
		curr := w.getConfig()
		if curr != nil {
			targetCfg = *curr
		}
	}

	if isLocalServer(targetCfg) {
		targetCfg.CloudProvider = "local"
		if targetCfg.Host == "" {
			targetCfg.Host = "localhost"
		}
	}

	exec := repositories.NewCryptoSSHExecutor()
	defer exec.Close()
	uc := usecases.NewConnectServerUseCase(exec)
	res, err := uc.Execute(targetCfg)
	if err != nil {
		http.Error(rw, fmt.Sprintf("Error ejecutando diagnóstico: %v", err), http.StatusInternalServerError)
		return
	}

	jsonResponse(rw, res)
}

func (w *WebServer) handleServers(rw http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		servers, err := w.repo.GetAllServerConfigs()
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(rw, servers)
		return
	}

	if !w.isAuthorized(req) {
		jsonError(rw, "Operación no autorizada: API key requerida", http.StatusUnauthorized)
		return
	}

	if req.Method == http.MethodPost {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		cfg, err := dto.DecodeServerConfig(body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		if isLocalServer(cfg) {
			cfg.CloudProvider = "local"
			cfg.Host = "localhost"
		}
		if !isLocalServer(cfg) && cfg.CloudProvider == "" {
			cfg.CloudProvider = "vps-direct"
		}
		if err := w.repo.SaveServerConfig(cfg); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		if cfg.IsActive {
			w.setConfig(&cfg)
		}
		jsonResponse(rw, map[string]interface{}{"status": "ok", "config": cfg})
		return
	}

	if req.Method == http.MethodDelete {
		name := req.URL.Query().Get("name")
		if name == "" {
			var bodyData struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(req.Body).Decode(&bodyData); err == nil {
				name = bodyData.Name
			}
		}
		name = strings.TrimSpace(name)
		if name == "" {
			http.Error(rw, "nombre de servidor requerido", http.StatusBadRequest)
			return
		}
		if err := w.repo.DeleteServerConfig(name); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		curr := w.getConfig()
		if curr != nil && curr.Name == name {
			activeCfg, errActive := w.repo.GetServerConfig()
			if errActive != nil {
				slog.Warn("web_server: error obteniendo activeCfg tras borrar servidor", "error", errActive)
			}
			w.setConfig(activeCfg)
		}
		jsonResponse(rw, map[string]string{"status": "ok"})
		return
	}

	http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
}

func (w *WebServer) handleSetActiveServer(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var bodyData struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(req.Body).Decode(&bodyData); err != nil || strings.TrimSpace(bodyData.Name) == "" {
		http.Error(rw, "nombre de servidor inválido", http.StatusBadRequest)
		return
	}
	if err := w.repo.SetActiveServerConfig(bodyData.Name); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	activeCfg, err := w.repo.GetServerConfig()
	if err == nil && activeCfg != nil {
		w.setConfig(activeCfg)
	}
	jsonResponse(rw, map[string]interface{}{"status": "ok", "active": activeCfg})
}

func (w *WebServer) handleProvisionServer(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqData ports.ProvisionCloudRequest
	if err := json.NewDecoder(req.Body).Decode(&reqData); err != nil {
		http.Error(rw, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	if reqData.APIToken == "" {
		http.Error(rw, "Se requiere un Token de API del proveedor de nube (Vultr o DigitalOcean)", http.StatusBadRequest)
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("web_server: error cerrando sshExec en provisionServer", "error", clErr)
		}
	}()
	connectUC := usecases.NewConnectServerUseCase(sshExec)
	provisionUC := usecases.NewProvisionCloudServerUseCase(w.repo, connectUC)

	res, err := provisionUC.Execute(reqData)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if reqData.SetAsActive {
		activeCfg, errByName := w.repo.GetServerConfigByName(reqData.Name)
		if errByName != nil {
			slog.Warn("web_server: error obteniendo servidor por nombre tras provision", "error", errByName)
		}
		if activeCfg != nil {
			w.setConfig(activeCfg)
		}
	}

	jsonResponse(rw, res)
}

func (w *WebServer) handleOpenTerminal(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if w.isExposed && !isLoopback(req.RemoteAddr) {
		jsonError(rw, "La terminal nativa del sistema solo puede abrirse desde la máquina anfitriona (localhost). Para acceso remoto, use SSH directo o la terminal integrada.", http.StatusForbidden)
		return
	}

	var payload struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	targetCfg := w.getConfig()
	if payload.Name != "" {
		found, err := w.repo.GetServerConfigByName(payload.Name)
		if err != nil || found == nil {
			http.Error(rw, fmt.Sprintf("Servidor '%s' no encontrado", payload.Name), http.StatusNotFound)
			return
		}
		targetCfg = found
	}

	if targetCfg == nil || (targetCfg.Host == "" && !isLocalConfig(targetCfg)) {
		http.Error(rw, "Servidor no encontrado o no configurado", http.StatusNotFound)
		return
	}

	cmdStr := osterminal.BuildSSHCommand(targetCfg.User, targetCfg.Host, targetCfg.Port, targetCfg.PrivateKey)
	if isLocalConfig(targetCfg) {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		cmdStr = shell
	}

	launchErr := osterminal.OpenNativeTerminal(cmdStr)
	if launchErr != nil {
		slog.Warn("falló al abrir terminal nativa", "cmd", cmdStr, "error", launchErr)
	}
	jsonResponse(rw, map[string]interface{}{
		"status":   "ok",
		"name":     targetCfg.Name,
		"command":  cmdStr,
		"launched": launchErr == nil,
		"isLocal":  isLocalConfig(targetCfg),
	})
}

func (w *WebServer) getTargetServerConfig(req *http.Request) (*domain.ServerConfig, error) {
	name := strings.TrimSpace(req.URL.Query().Get("server"))
	if name == "" {
		name = strings.TrimSpace(req.URL.Query().Get("name"))
	}
	if name != "" {
		found, err := w.repo.GetServerConfigByName(name)
		if err != nil {
			return nil, err
		}
		if found == nil {
			return nil, fmt.Errorf("servidor '%s' no encontrado", name)
		}
		if found.Host == "" && !isLocalConfig(found) {
			return nil, fmt.Errorf("servidor '%s' no tiene Host configurado", name)
		}
		return found, nil
	}

	cfg := w.getConfig()
	if cfg == nil || (cfg.Host == "" && !isLocalConfig(cfg)) {
		return nil, fmt.Errorf("no hay un servidor activo configurado")
	}
	return cfg, nil
}

func (w *WebServer) handleHostInspect(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, err := w.getTargetServerConfig(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}

	cacheKey := "inspect:" + cfg.Name
	if req.URL.Query().Get("fresh") != "true" && req.URL.Query().Get("force") != "true" {
		if cached, ok := w.getCache(cacheKey); ok {
			jsonResponse(rw, cached)
			return
		}
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("web_server: error cerrando sshExec en host inspect", "error", clErr)
		}
	}()

	uc := usecases.NewInspectHostUseCase(sshExec)
	inspection, err := uc.Execute(*cfg)
	if err != nil {
		http.Error(rw, fmt.Sprintf("Error inspeccionando host: %v", err), http.StatusInternalServerError)
		return
	}

	w.setCache(cacheKey, inspection, 5*time.Second)
	jsonResponse(rw, inspection)
}

func (w *WebServer) handleHostMetrics(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, err := w.getTargetServerConfig(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}

	cacheKey := "metrics:" + cfg.Name
	if req.URL.Query().Get("fresh") != "true" && req.URL.Query().Get("force") != "true" {
		if cached, ok := w.getCache(cacheKey); ok {
			jsonResponse(rw, cached)
			return
		}
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("web_server: error cerrando sshExec en host metrics", "error", clErr)
		}
	}()

	uc := usecases.NewInspectHostUseCase(sshExec)
	metrics, err := uc.ExecuteMetricsOnly(*cfg)
	if err != nil {
		http.Error(rw, fmt.Sprintf("Error obteniendo métricas del host: %v", err), http.StatusInternalServerError)
		return
	}

	w.setCache(cacheKey, metrics, 5*time.Second)
	jsonResponse(rw, metrics)
}

func (w *WebServer) handleHostServices(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, err := w.getTargetServerConfig(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}

	cacheKey := "services:" + cfg.Name
	if req.URL.Query().Get("fresh") != "true" && req.URL.Query().Get("force") != "true" {
		if cached, ok := w.getCache(cacheKey); ok {
			jsonResponse(rw, cached)
			return
		}
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("web_server: error cerrando sshExec en host services", "error", clErr)
		}
	}()

	uc := usecases.NewInspectHostUseCase(sshExec)
	services, err := uc.ExecuteServicesOnly(*cfg)
	if err != nil {
		http.Error(rw, fmt.Sprintf("Error obteniendo servicios del host: %v", err), http.StatusInternalServerError)
		return
	}

	w.setCache(cacheKey, services, 10*time.Second)
	jsonResponse(rw, services)
}

func (w *WebServer) handleHostDevices(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, err := w.getTargetServerConfig(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}

	cacheKey := "devices:" + cfg.Name
	if req.URL.Query().Get("fresh") != "true" && req.URL.Query().Get("force") != "true" {
		if cached, ok := w.getCache(cacheKey); ok {
			jsonResponse(rw, cached)
			return
		}
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("web_server: error cerrando sshExec en host devices", "error", clErr)
		}
	}()

	uc := usecases.NewListDevicesUseCase(sshExec)
	devices, err := uc.Execute(*cfg)
	if err != nil {
		http.Error(rw, fmt.Sprintf("Error obteniendo dispositivos del host: %v", err), http.StatusInternalServerError)
		return
	}

	w.setCache(cacheKey, devices, 15*time.Second)
	jsonResponse(rw, devices)
}

func (w *WebServer) handleHostSecurity(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, err := w.getTargetServerConfig(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}

	cacheKey := "security:" + cfg.Name
	if req.URL.Query().Get("fresh") != "true" && req.URL.Query().Get("force") != "true" {
		if cached, ok := w.getCache(cacheKey); ok {
			jsonResponse(rw, cached)
			return
		}
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("web_server: error cerrando sshExec en host security", "error", clErr)
		}
	}()

	if err := sshExec.Connect(*cfg); err != nil {
		http.Error(rw, fmt.Sprintf("Error conectando SSH al servidor: %v", err), http.StatusInternalServerError)
		return
	}

	uc := usecases.NewInspectSecurityUseCase(sshExec)
	report, err := uc.Execute()
	if err != nil {
		http.Error(rw, fmt.Sprintf("Error inspeccionando seguridad del host: %v", err), http.StatusInternalServerError)
		return
	}

	w.setCache(cacheKey, report, 15*time.Second)
	jsonResponse(rw, report)
}

func (w *WebServer) handleSystemReport(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, err := w.getTargetServerConfig(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}

	// Sondeo temprano: preserva el 500 inmediato si el host no es alcanzable, antes de
	// que cada usecase abra su propia conexión (gracias al pool no implica un dial extra).
	probeExec := repositories.NewCryptoSSHExecutor()
	if err := probeExec.Connect(*cfg); err != nil {
		http.Error(rw, fmt.Sprintf("Error conectando SSH: %v", err), http.StatusInternalServerError)
		return
	}
	if clErr := probeExec.Close(); clErr != nil {
		slog.Warn("web_server: error cerrando sondeo ssh en system report", "error", clErr)
	}

	// Cada usecase administra su propia conexión SSH (Connect/Close), así que se les da un
	// ejecutor independiente en vez de compartir uno: InspectHostUseCase y
	// GetSwarmStatusUseCase cierran su conexión al terminar, lo que dejaba a
	// InspectSecurityUseCase (que no reconecta) operando sobre una conexión ya cerrada.
	inspectUC := usecases.NewInspectHostUseCase(repositories.NewCryptoSSHExecutor())
	telemetry, _ := inspectUC.Execute(*cfg)
	var tel domain.HostInspection
	if telemetry != nil {
		tel = *telemetry
	}

	var secReport domain.SecurityReport
	secExec := repositories.NewCryptoSSHExecutor()
	if err := secExec.Connect(*cfg); err != nil {
		slog.Warn("web_server: error conectando ssh para reporte de seguridad", "error", err)
	} else {
		secUC := usecases.NewInspectSecurityUseCase(secExec)
		secReport, _ = secUC.Execute()
		if clErr := secExec.Close(); clErr != nil {
			slog.Warn("web_server: error cerrando ssh de reporte de seguridad", "error", clErr)
		}
	}

	swarmUC := usecases.NewGetSwarmStatusUseCase(repositories.NewCryptoSSHExecutor())
	swarmStatus, _ := swarmUC.Execute(*cfg)
	var swSt domain.SwarmStatus
	if swarmStatus != nil {
		swSt = *swarmStatus
	}

	dbs, _ := w.repo.GetDatabases(cfg.Name)

	report := domain.SystemDiagnosticReport{
		Timestamp:   time.Now(),
		ServerName:  cfg.Name,
		Host:        cfg.Host,
		Telemetry:   tel,
		Security:    secReport,
		SwarmStatus: swSt,
		Databases:   dbs,
		GeneratedBy: "Tarhiata Cloud Studio",
	}

	if req.URL.Query().Get("format") == "markdown" || req.URL.Query().Get("format") == "md" {
		md := generateMarkdownReport(report)
		rw.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		rw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"tarhiata-diagnostic-%s-%s.md\"", cfg.Name, time.Now().Format("20060102-150405")))
		rw.WriteHeader(http.StatusOK)
		if _, writeErr := rw.Write([]byte(md)); writeErr != nil {
			slog.Warn("web_server: error escribiendo reporte markdown", "error", writeErr)
		}
		return
	}

	jsonResponse(rw, report)
}

func generateMarkdownReport(r domain.SystemDiagnosticReport) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# 📊 Tarhiata-Ops — Reporte de Diagnóstico de Sistema\n\n"))
	sb.WriteString(fmt.Sprintf("- **Servidor:** `%s` (%s)\n", r.ServerName, r.Host))
	sb.WriteString(fmt.Sprintf("- **Fecha:** %s\n", r.Timestamp.Format("2006-01-02 15:04:05 MST")))
	sb.WriteString(fmt.Sprintf("- **Sistema Operativo:** %s\n", r.Telemetry.Metrics.OS))
	sb.WriteString(fmt.Sprintf("- **Hostname:** %s\n", r.Telemetry.Metrics.Hostname))
	sb.WriteString(fmt.Sprintf("- **Docker Version:** %s (Swarm Activo: %v)\n\n", r.SwarmStatus.DockerVersion, r.SwarmStatus.Active))

	sb.WriteString("## ⚡ Telemetría de Recursos\n")
	sb.WriteString(fmt.Sprintf("- **CPU:** %.1f%% uso | Cores: %d | LoadAvg: %s\n", r.Telemetry.Metrics.CPUPercent, r.Telemetry.Metrics.CPUCores, r.Telemetry.Metrics.LoadAvg))
	sb.WriteString(fmt.Sprintf("- **RAM:** %d MB / %d MB (%.1f%% uso)\n", r.Telemetry.Metrics.MemoryUsedMB, r.Telemetry.Metrics.MemoryTotalMB, r.Telemetry.Metrics.MemoryPercent))
	sb.WriteString(fmt.Sprintf("- **Disco:** %.1f GB / %.1f GB (%.1f%% uso)\n", r.Telemetry.Metrics.DiskUsedGB, r.Telemetry.Metrics.DiskTotalGB, r.Telemetry.Metrics.DiskPercent))
	sb.WriteString(fmt.Sprintf("- **Uptime:** %s\n\n", r.Telemetry.Metrics.Uptime))

	sb.WriteString("## 🛡️ Seguridad y Cortafuegos\n")
	sb.WriteString(fmt.Sprintf("- **UFW Activo:** %v\n", r.Security.UFWActive))
	if len(r.Security.UFWRules) > 0 {
		sb.WriteString("| # | Puerto/Destino | Acción | Origen | Protocolo |\n|---|---|---|---|---|\n")
		for _, rule := range r.Security.UFWRules {
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n", rule.Number, rule.To, rule.Action, rule.From, rule.Proto))
		}
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("- **Fail2Ban Activo:** %v | Jaulas: %s | IPs Bloqueadas: %d\n\n", r.Security.Fail2Ban.Active, strings.Join(r.Security.Fail2Ban.Jails, ", "), r.Security.Fail2Ban.TotalBanned))

	sb.WriteString("## 🚀 Contenedores y Servicios Swarm\n")
	if len(r.SwarmStatus.Services) > 0 {
		sb.WriteString("| Servicio | Modo | Réplicas | Imagen | Dominio |\n|---|---|---|---|---|\n")
		for _, s := range r.SwarmStatus.Services {
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n", s.Name, s.Mode, s.Replicas, s.Image, s.Domain))
		}
		sb.WriteString("\n")
	}
	if len(r.SwarmStatus.Services) == 0 {
		sb.WriteString("*No se detectaron servicios en ejecución en el clúster.*\n\n")
	}

	sb.WriteString("## 🗄️ Bases de Datos\n")
	if len(r.Databases) > 0 {
		sb.WriteString("| Nombre | Motor | Tipo | Puerto Interno | Nodo |\n|---|---|---|---|---|\n")
		for _, db := range r.Databases {
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %s |\n", db.Name, db.Engine, db.DeployType, db.InternalPort, db.TargetNode))
		}
		sb.WriteString("\n")
	}
	if len(r.Databases) == 0 {
		sb.WriteString("*Sin bases de datos persistentes registradas.*\n\n")
	}

	return sb.String()
}

func (w *WebServer) handleSwarmStatus(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, err := w.getTargetServerConfig(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}

	cacheKey := "swarm:" + cfg.Name
	if req.URL.Query().Get("fresh") != "true" && req.URL.Query().Get("force") != "true" {
		if cached, ok := w.getCache(cacheKey); ok {
			jsonResponse(rw, cached)
			return
		}
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("web_server: error cerrando sshExec en swarm status", "error", clErr)
		}
	}()

	uc := usecases.NewGetSwarmStatusUseCase(sshExec)
	status, err := uc.Execute(*cfg)
	if err != nil {
		http.Error(rw, fmt.Sprintf("Error obteniendo estado de Swarm: %v", err), http.StatusInternalServerError)
		return
	}

	// Incorporar información de bases de datos registradas
	dbs, dbsErr := w.repo.GetDatabases(cfg.Name)
	if dbsErr != nil {
		slog.Warn("web_server: error obteniendo databases para swarm status", "error", dbsErr)
		dbs = []domain.SavedDatabase{}
	}
	liveServicesMap := make(map[string]bool)
	for _, s := range status.Services {
		liveServicesMap[s.Name] = true
	}

	status.Databases = make([]domain.SwarmDBInfo, 0, len(dbs))
	for _, db := range dbs {
		dbStatus := "offline"
		svcName := fmt.Sprintf("tarhiata-db-%s", db.Name)
		if db.DeployType == "external" {
			dbStatus = "external"
		} else if liveServicesMap[svcName] || liveServicesMap[db.Name] {
			dbStatus = "running"
		}

		internalDns := fmt.Sprintf("%s:%d", svcName, db.InternalPort)
		if db.DeployType == "external" && db.ExternalURL != "" {
			internalDns = db.ExternalURL
		}

		status.Databases = append(status.Databases, domain.SwarmDBInfo{
			ID:           db.ID,
			Name:         db.Name,
			Engine:       db.Engine,
			DeployType:   db.DeployType,
			InternalPort: db.InternalPort,
			ExternalURL:  db.ExternalURL,
			TargetNode:   db.TargetNode,
			Status:       dbStatus,
			InternalDNS:  internalDns,
		})
	}

	// Incorporar información de servicios registrados en SQLite para que no desaparezcan si están detenidos
	savedSvcs, svcsErr := w.repo.GetServices(cfg.Name)
	if svcsErr != nil {
		slog.Warn("web_server: error obteniendo servicios para swarm status", "error", svcsErr)
	} else {
		for _, s := range savedSvcs {
			sNameLower := strings.ToLower(strings.TrimSpace(s.Name))
			alreadyRunning := false
			for _, live := range status.Services {
				liveLower := strings.ToLower(strings.TrimSpace(live.Name))
				if liveLower == sNameLower ||
					liveLower == "tarhiata-app-"+sNameLower ||
					strings.HasPrefix(liveLower, sNameLower+"_") ||
					strings.HasPrefix(liveLower, sNameLower+".") ||
					strings.Contains(liveLower, sNameLower) {
					alreadyRunning = true
					break
				}
			}
			if !alreadyRunning {
				portsStr := fmt.Sprintf("%d", s.Port)
				status.Services = append(status.Services, domain.SwarmServiceInfo{
					ID:       "stopped",
					Name:     s.Name,
					Mode:     "replicated",
					Replicas: "0/1",
					Image:    s.ImageSource,
					Ports:    portsStr,
					Domain:   s.Domain,
					Expose:   s.Expose,
				})
			}
		}
	}

	w.setCache(cacheKey, status, 5*time.Second)
	jsonResponse(rw, status)
}

func (w *WebServer) handleConnectAll(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	servers, err := w.repo.GetAllServerConfigs()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	results := make([]domain.ConnectionResult, len(servers))
	var wg sync.WaitGroup

	for i, s := range servers {
		wg.Add(1)
		go func(idx int, cfg domain.ServerConfig) {
			defer wg.Done()
			exec := repositories.NewCryptoSSHExecutor()
			defer exec.Close()
			uc := usecases.NewConnectServerUseCase(exec)
			res, execErr := uc.Execute(cfg)
			if execErr != nil {
				results[idx] = domain.ConnectionResult{
					Name:       cfg.Name,
					Connected:  false,
					IsLocal:    isLocalServer(cfg),
					TargetHost: cfg.Host,
					Message:    execErr.Error(),
					Errors:     []string{execErr.Error()},
				}
				return
			}
			results[idx] = *res
		}(i, s)
	}

	wg.Wait()
	jsonResponse(rw, results)
}

func (w *WebServer) handleServices(rw http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		cfg := w.resolveTargetServer(req.URL.Query().Get("server"))
		serverName := ""
		if cfg != nil {
			serverName = cfg.Name
		}
		svcs, err := w.repo.GetServices(serverName)
		if err != nil {
			http.Error(rw, fmt.Sprintf("Error leyendo servicios: %v", err), http.StatusInternalServerError)
			return
		}
		if svcs == nil {
			svcs = []domain.SavedService{}
		}

		jsonResponse(rw, svcs)
		return
	}
	if req.Method == http.MethodPost {
		if !w.isAuthorized(req) {
			jsonError(rw, "Operación no autorizada: API key requerida", http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		svc, err := dto.DecodeSavedService(body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		if !isValidNodeID(svc.Name) {
			http.Error(rw, "Nombre de servicio inválido", http.StatusBadRequest)
			return
		}
		isGitSource := strings.TrimSpace(svc.SourceType) == "git"
		if isGitSource {
			if strings.TrimSpace(svc.GitRepoURL) == "" {
				http.Error(rw, "gitRepoUrl requerido para origen 'git'", http.StatusBadRequest)
				return
			}
		} else if strings.TrimSpace(svc.ImageSource) == "" {
			http.Error(rw, "imageSource requerido", http.StatusBadRequest)
			return
		}

		flusher, isStreaming := setupStreaming(rw)
		send := func(t, m string) {}
		if isStreaming {
			send = func(t, m string) { streamJSON(rw, flusher, t, m) }
		}

		send("step", fmt.Sprintf("🚀 Desplegando Servicio '%s'...", svc.Name))
		if isGitSource {
			send("log", fmt.Sprintf("🔧 Origen: repo Git %s", svc.GitRepoURL))
		} else {
			send("log", fmt.Sprintf("📦 Imagen: %s", svc.ImageSource))
		}
		send("log", fmt.Sprintf("🔌 Puerto: %d", svc.Port))

		serverTarget := req.URL.Query().Get("server")
		if serverTarget == "" && svc.TargetNode != "" {
			serverTarget = svc.TargetNode
		}
		cfg := w.resolveTargetServer(serverTarget)
		if cfg != nil {
			svc.ServerName = cfg.Name
		}

		if isGitSource {
			if cfg == nil || (cfg.Host == "" && !isLocalConfig(cfg)) {
				msg := "Se requiere un servidor VPS configurado para build-from-source"
				if isStreaming {
					send("error", "❌ "+msg)
					return
				}
				http.Error(rw, msg, http.StatusBadRequest)
				return
			}
			send("step", "🔧 Construyendo imagen desde repo Git...")
			buildUC := usecases.NewBuildFromSourceUseCase(repositories.NewCryptoSSHExecutor())
			tag, errBuild := buildUC.Execute(svc, "", *cfg, func(line string) { send("log", line) })
			if errBuild != nil {
				msg := fmt.Sprintf("Error en build desde Git: %v", errBuild)
				if isStreaming {
					send("error", "❌ "+msg)
					return
				}
				http.Error(rw, msg, http.StatusInternalServerError)
				return
			}
			svc.ImageSource = tag
		}

		if err := w.repo.SaveService(svc); err != nil {
			if isStreaming {
				send("error", fmt.Sprintf("❌ Error guardando servicio: %v", err))
				return
			}
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		send("log", "💾 Registro del Servicio guardado en catálogo local")

		if cfg != nil && (cfg.Host != "" || isLocalConfig(cfg)) {
			send("step", fmt.Sprintf("🔗 Conectando por SSH a %s...", cfg.Host))
			sshExec := repositories.NewCryptoSSHExecutor()
			if err := sshExec.Connect(*cfg); err != nil {
				if isStreaming {
					send("error", fmt.Sprintf("❌ Falló conexión SSH: %v", err))
					return
				}
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
			defer sshExec.Close()

			loggingExec := NewLoggingSSHExecutor(sshExec, send)
			deployConfig := domain.DeployConfig{
				ImageSource:    svc.ImageSource,
				Port:           svc.Port,
				Domain:         svc.Domain,
				Expose:         svc.Expose,
				EnableSSL:      svc.EnableSSL,
				HealthcheckCmd: svc.HealthcheckCmd,
				TargetNode:     svc.TargetNode,
				ServerName:     cfg.Name,
			}
			customSvc := domain.CustomService{
				Name:          svc.Name,
				PreDeployHook: svc.PreDeployHook,
			}
			if svc.EnvVars != "" {
				workDir := fmt.Sprintf("/opt/tarhiata/services/%s", svc.Name)
				if _, err := loggingExec.RunCommand(fmt.Sprintf("mkdir -p %s", workDir)); err != nil {
					slog.Warn("falló al crear workdir", "error", err)
				}
				envEncoded := base64.StdEncoding.EncodeToString([]byte(svc.EnvVars))
				if _, err := loggingExec.RunCommand(fmt.Sprintf("echo '%s' | base64 -d > %s/.env", envEncoded, workDir)); err != nil {
					slog.Warn("falló al decodificar .env", "error", err)
				}
			}
			send("step", "🛠️  Ejecutando servicio en Docker Swarm...")
			if err := usecases.NewDeployServiceUseCase(loggingExec).Execute(customSvc, deployConfig); err != nil {
				if isStreaming {
					send("error", fmt.Sprintf("❌ Error al desplegar servicio: %v", err))
					return
				}
				http.Error(rw, fmt.Sprintf("Error al desplegar servicio SSH: %v", err), http.StatusInternalServerError)
				return
			}
		}

		if err := w.repo.SaveAuditLog(domain.AuditLog{
			Action:       "DEPLOY",
			ResourceType: "service",
			ResourceName: svc.Name,
			Details:      fmt.Sprintf("Desplegado servicio '%s' (Imagen: %s, Puerto: %d, Dominio: %s)", svc.Name, svc.ImageSource, svc.Port, svc.Domain),
		}); err != nil {
			slog.Warn("falló al guardar log de auditoría", "error", err)
		}

		send("step", fmt.Sprintf("✅ Servicio '%s' desplegado con éxito!", svc.Name))
		if isStreaming {
			streamDoneJSON(rw, flusher, map[string]string{"status": "created", "name": svc.Name, "message": fmt.Sprintf("¡Servicio '%s' listo!", svc.Name)})
		} else {
			jsonResponse(rw, map[string]string{"status": "created", "name": svc.Name})
		}
		return
	}
	if req.Method == http.MethodDelete {
		name := req.URL.Query().Get("name")
		if name == "" || !isValidNodeID(name) {
			http.Error(rw, "Nombre de servicio inválido o ausente", http.StatusBadRequest)
			return
		}
		serverTarget := req.URL.Query().Get("server")
		cfg := w.resolveTargetServer(serverTarget)
		if cfg != nil && (cfg.Host != "" || isLocalConfig(cfg)) {
			sshExec := repositories.NewCryptoSSHExecutor()
			if err := sshExec.Connect(*cfg); err == nil {
				defer sshExec.Close()

				cleanName := strings.TrimPrefix(name, "tarhiata-db-")
				cleanName = strings.TrimPrefix(cleanName, "tarhiata-")

				dbService1 := fmt.Sprintf("tarhiata-db-%s", cleanName)
				dbService2 := fmt.Sprintf("tarhiata-db-%s", name)

				cmd := fmt.Sprintf("docker service rm %s || docker service rm %s || docker service rm %s || docker rm -f %s || docker rm -f %s || docker rm -f %s",
					dbService1, dbService2, name, dbService1, dbService2, name)
				if _, err := sshExec.RunCommand(cmd); err != nil {
					slog.Debug("aviso al remover contenedor/servicio ssh", "cmd", cmd, "error", err)
				}
			}
		}

		cleanName := strings.TrimPrefix(name, "tarhiata-db-")
		cleanName = strings.TrimPrefix(cleanName, "tarhiata-")
		deleteServerName := ""
		if cfg != nil {
			deleteServerName = cfg.Name
		}

		if err := w.repo.DeleteService(name, deleteServerName); err != nil {
			slog.Debug("aviso al eliminar servicio por nombre", "name", name, "error", err)
		}
		if err := w.repo.DeleteService(cleanName, deleteServerName); err != nil {
			slog.Debug("aviso al eliminar servicio por cleanName", "cleanName", cleanName, "error", err)
		}
		if err := w.repo.DeleteDatabase(name, deleteServerName); err != nil {
			slog.Debug("aviso al eliminar base de datos por nombre", "name", name, "error", err)
		}
		if err := w.repo.DeleteDatabase(cleanName, deleteServerName); err != nil {
			slog.Debug("aviso al eliminar base de datos por cleanName", "cleanName", cleanName, "error", err)
		}

		jsonResponse(rw, map[string]string{"status": "deleted", "name": name})
		return
	}
}

func (w *WebServer) handleServiceItem(rw http.ResponseWriter, req *http.Request) {
	name := strings.TrimPrefix(req.URL.Path, "/api/services/")
	if name != "update" && !isValidNodeID(name) {
		http.Error(rw, "Nombre de servicio inválido", http.StatusBadRequest)
		return
	}
	if req.Method == http.MethodGet {
		cfg := w.resolveTargetServer(req.URL.Query().Get("server"))
		serverName := ""
		if cfg != nil {
			serverName = cfg.Name
		}
		svc, err := w.repo.GetService(name, serverName)
		if err != nil {
			http.Error(rw, fmt.Sprintf("Error leyendo servicio: %v", err), http.StatusInternalServerError)
			return
		}
		if svc == nil {
			http.Error(rw, "servicio no encontrado", http.StatusNotFound)
			return
		}
		jsonResponse(rw, svc)
		return
	}

	if !w.isAuthorized(req) {
		jsonError(rw, "Operación no autorizada: API key requerida", http.StatusUnauthorized)
		return
	}

	if req.Method == http.MethodPut || req.Method == http.MethodPost {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		svc, err := dto.DecodeSavedService(body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		if name == "update" {
			if svc.Name == "" || !isValidNodeID(svc.Name) {
				http.Error(rw, "Nombre de servicio inválido en payload", http.StatusBadRequest)
				return
			}
			name = svc.Name
		}
		if name != "update" {
			svc.Name = name
		}

		serverTarget := req.URL.Query().Get("server")
		if serverTarget == "" && svc.TargetNode != "" {
			serverTarget = svc.TargetNode
		}
		cfg := w.resolveTargetServer(serverTarget)
		if cfg != nil {
			svc.ServerName = cfg.Name
		}

		// Los secretos (token de git, secreto de webhook) no viajan de vuelta al cliente
		// en los formularios de edición; si llegan vacíos, se conserva el valor ya
		// guardado en vez de borrarlo en cada edición de cualquier otro campo.
		if strings.TrimSpace(svc.GitAccessToken) == "" || strings.TrimSpace(svc.WebhookSecret) == "" {
			if existing, errExisting := w.repo.GetService(svc.Name, svc.ServerName); errExisting == nil && existing != nil {
				if strings.TrimSpace(svc.GitAccessToken) == "" {
					svc.GitAccessToken = existing.GitAccessToken
				}
				if strings.TrimSpace(svc.WebhookSecret) == "" {
					svc.WebhookSecret = existing.WebhookSecret
				}
			}
		}
		if err := w.repo.SaveService(svc); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		if cfg != nil && (cfg.Host != "" || isLocalConfig(cfg)) {
			sshExec := repositories.NewCryptoSSHExecutor()
			if err := sshExec.Connect(*cfg); err == nil {
				defer sshExec.Close()
				deployConfig := domain.DeployConfig{
					ImageSource:    svc.ImageSource,
					Port:           svc.Port,
					Domain:         svc.Domain,
					Expose:         svc.Expose,
					EnableSSL:      svc.EnableSSL,
					HealthcheckCmd: svc.HealthcheckCmd,
					TargetNode:     svc.TargetNode,
					ServerName:     cfg.Name,
				}
				customSvc := domain.CustomService{
					Name:          svc.Name,
					PreDeployHook: svc.PreDeployHook,
				}
				if svc.EnvVars != "" {
					workDir := fmt.Sprintf("/opt/tarhiata/services/%s", svc.Name)
					if _, err := sshExec.RunCommand(fmt.Sprintf("mkdir -p %s", workDir)); err != nil {
						slog.Warn("falló al crear workdir", "error", err)
					}
					envEncoded := base64.StdEncoding.EncodeToString([]byte(svc.EnvVars))
					if _, err := sshExec.RunCommand(fmt.Sprintf("echo '%s' | base64 -d > %s/.env", envEncoded, workDir)); err != nil {
						slog.Warn("falló al decodificar .env", "error", err)
					}
				}
				if err := usecases.NewDeployServiceUseCase(sshExec).Execute(customSvc, deployConfig); err != nil {
					http.Error(rw, fmt.Sprintf("Error al actualizar despliegue SSH: %v", err), http.StatusInternalServerError)
					return
				}
			}
		}
		jsonResponse(rw, map[string]string{"status": "updated", "name": name})
		return
	}
	if req.Method == http.MethodDelete {
		serverTarget := req.URL.Query().Get("server")
		cfg := w.resolveTargetServer(serverTarget)
		if cfg != nil && (cfg.Host != "" || isLocalConfig(cfg)) {
			sshExec := repositories.NewCryptoSSHExecutor()
			if err := sshExec.Connect(*cfg); err == nil {
				defer sshExec.Close()
				if _, err := sshExec.RunCommand(fmt.Sprintf("docker service rm %s || docker rm -f %s", name, name)); err != nil {
					slog.Debug("aviso al remover contenedor/servicio ssh", "error", err)
				}
			}
		}
		deleteServerName := ""
		if cfg != nil {
			deleteServerName = cfg.Name
		}
		if err := w.repo.DeleteService(name, deleteServerName); err != nil {
			http.Error(rw, fmt.Sprintf("Error al eliminar servicio: %v", err), http.StatusInternalServerError)
			return
		}
		w.syncStateToRemote(cfg)
		jsonResponse(rw, map[string]string{"status": "deleted", "name": name})
		return
	}
}

func (w *WebServer) handleDatabases(rw http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		cfg := w.resolveTargetServer(req.URL.Query().Get("server"))
		serverName := ""
		if cfg != nil {
			serverName = cfg.Name
		}
		dbs, err := w.repo.GetDatabases(serverName)
		if err != nil {
			http.Error(rw, fmt.Sprintf("Error leyendo bases de datos: %v", err), http.StatusInternalServerError)
			return
		}
		if dbs == nil {
			dbs = []domain.SavedDatabase{}
		}

		for i := range dbs {
			dbs[i].Password = ""
		}
		jsonResponse(rw, dbs)
		return
	}
	if req.Method == http.MethodPost {
		if !w.isAuthorized(req) {
			jsonError(rw, "Operación no autorizada: API key requerida", http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		db, err := dto.DecodeSavedDatabase(body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		if !isValidNodeID(db.Name) {
			http.Error(rw, "Nombre de base de datos inválido", http.StatusBadRequest)
			return
		}
		if db.InternalPort == 0 {
			db.InternalPort = 5432
			if db.Engine == "mongodb" || db.Engine == "mongo" { db.InternalPort = 27017 }
			if db.Engine == "mysql" { db.InternalPort = 3306 }
			if db.Engine == "redis" { db.InternalPort = 6379 }
		}
		if db.DeployType == "" { db.DeployType = "single-node" }

		flusher, isStreaming := setupStreaming(rw)
		send := func(t, m string) {}
		if isStreaming {
			send = func(t, m string) { streamJSON(rw, flusher, t, m) }
		}

		send("step", fmt.Sprintf("🚀 Desplegando Base de Datos '%s' (%s)...", db.Name, strings.ToUpper(db.Engine)))
		send("log", fmt.Sprintf("📦 Motor: %s", db.Engine))
		send("log", fmt.Sprintf("🔌 Puerto Interno: %d", db.InternalPort))
		if db.TargetNode != "" {
			send("log", fmt.Sprintf("🎯 Nodo Target: %s", db.TargetNode))
		}

		serverTarget := req.URL.Query().Get("server")
		if serverTarget == "" && db.TargetNode != "" {
			serverTarget = db.TargetNode
		}
		cfg := w.resolveTargetServer(serverTarget)
		if cfg != nil {
			db.ServerName = cfg.Name
		}

		if err := w.repo.SaveDatabase(db); err != nil {
			if isStreaming {
				send("error", fmt.Sprintf("❌ Error al guardar base de datos: %v", err))
				return
			}
			http.Error(rw, fmt.Sprintf("Error al guardar base de datos: %v", err), http.StatusInternalServerError)
			return
		}
		send("log", "💾 Registro de Base de Datos guardado en catálogo local")

		if cfg != nil && (cfg.Host != "" || isLocalConfig(cfg)) {
			send("step", fmt.Sprintf("🔗 Conectando por SSH a %s...", cfg.Host))
			sshExec := repositories.NewCryptoSSHExecutor()
			if err := sshExec.Connect(*cfg); err != nil {
				if isStreaming {
					send("error", fmt.Sprintf("❌ Falló conexión SSH: %v", err))
					return
				}
				http.Error(rw, fmt.Sprintf("Falló conexión SSH: %v", err), http.StatusInternalServerError)
				return
			}
			defer sshExec.Close()

			loggingExec := NewLoggingSSHExecutor(sshExec, send)
			send("step", "🛠️  Ejecutando despliegue de contenedor y montaje de volumen persistente...")
			if err := usecases.NewDeployDatabaseUseCase(loggingExec).Execute(db, *cfg); err != nil {
				if isStreaming {
					send("error", fmt.Sprintf("❌ Error al desplegar BD SSH: %v", err))
					return
				}
				http.Error(rw, fmt.Sprintf("Error al desplegar BD SSH: %v", err), http.StatusInternalServerError)
				return
			}
		}

		if err := w.repo.SaveAuditLog(domain.AuditLog{
			Action:       "DEPLOY",
			ResourceType: "database",
			ResourceName: db.Name,
			Details:      fmt.Sprintf("Desplegada BD '%s' (Motor: %s, Nodo: %s, Recovery: %v)", db.Name, db.Engine, db.TargetNode, db.ReuseExistingData),
		}); err != nil {
			slog.Warn("falló al guardar log de auditoría para base de datos", "db", db.Name, "error", err)
		}

		send("step", fmt.Sprintf("✅ Base de Datos '%s' desplegada con éxito!", db.Name))
		if isStreaming {
			streamDoneJSON(rw, flusher, map[string]string{"status": "created", "name": db.Name, "message": fmt.Sprintf("¡Base de Datos '%s' lista!", db.Name)})
		} else {
			jsonResponse(rw, map[string]string{"status": "created", "name": db.Name})
		}
		return
	}
	if req.Method == http.MethodDelete {
		name := req.URL.Query().Get("name")
		if name == "" || !isValidNodeID(name) {
			http.Error(rw, "Nombre de base de datos inválido o ausente", http.StatusBadRequest)
			return
		}
		serverTarget := req.URL.Query().Get("server")
		cfg := w.resolveTargetServer(serverTarget)
		deleteServerName := ""
		if cfg != nil {
			deleteServerName = cfg.Name
		}
		if cfg != nil && (cfg.Host != "" || isLocalConfig(cfg)) {
			sshExec := repositories.NewCryptoSSHExecutor()
			if err := sshExec.Connect(*cfg); err == nil {
				defer sshExec.Close()

				// 1. Remover variables de entorno inyectadas en servicios vinculados en Swarm
				links, errLinks := w.repo.GetServiceLinks(deleteServerName)
				if errLinks != nil {
					slog.Warn("falló al obtener enlaces de servicios", "error", errLinks)
				}
				unlinkUC := usecases.NewUnlinkServicesUseCase(w.repo, sshExec)
				for _, l := range links {
					if l.TargetSvc == name || l.SourceSvc == name {
						if err := unlinkUC.Execute(l.SourceSvc, l.TargetSvc, deleteServerName); err != nil {
							slog.Warn("falló al desvincular servicio", "source", l.SourceSvc, "target", l.TargetSvc, "error", err)
						}
					}
				}

				// 2. Probar eliminación de todas las variaciones de nombres posibles en Docker Swarm
				cleanName := strings.TrimPrefix(name, "tarhiata-db-")
				cleanName = strings.TrimPrefix(cleanName, "tarhiata-")

				dbService1 := fmt.Sprintf("tarhiata-db-%s", cleanName)
				dbService2 := fmt.Sprintf("tarhiata-db-%s", name)

				cmd := fmt.Sprintf("docker service rm %s || docker service rm %s || docker service rm %s || docker rm -f %s || docker rm -f %s || docker rm -f %s",
					dbService1, dbService2, name, dbService1, dbService2, name)
				if _, err := sshExec.RunCommand(cmd); err != nil {
					slog.Debug("aviso al remover contenedor/servicio ssh de base de datos", "cmd", cmd, "error", err)
				}
			}
		}

		cleanName := strings.TrimPrefix(name, "tarhiata-db-")
		cleanName = strings.TrimPrefix(cleanName, "tarhiata-")

		if err := w.repo.DeleteDatabase(name, deleteServerName); err != nil {
			slog.Debug("aviso al eliminar base de datos por nombre", "name", name, "error", err)
		}
		if err := w.repo.DeleteDatabase(cleanName, deleteServerName); err != nil {
			slog.Debug("aviso al eliminar base de datos por cleanName", "cleanName", cleanName, "error", err)
		}
		if err := w.repo.DeleteService(name, deleteServerName); err != nil {
			slog.Debug("aviso al eliminar servicio por nombre", "name", name, "error", err)
		}
		if err := w.repo.DeleteService(cleanName, deleteServerName); err != nil {
			slog.Debug("aviso al eliminar servicio por cleanName", "cleanName", cleanName, "error", err)
		}

		jsonResponse(rw, map[string]string{"status": "deleted", "name": name})
		return
	}
}

func (w *WebServer) handleDatabaseItem(rw http.ResponseWriter, req *http.Request) {
	name := strings.TrimPrefix(req.URL.Path, "/api/databases/")
	if !isValidNodeID(name) {
		http.Error(rw, "Nombre de base de datos inválido", http.StatusBadRequest)
		return
	}
	if req.Method == http.MethodGet {
		cfg := w.resolveTargetServer(req.URL.Query().Get("server"))
		serverName := ""
		if cfg != nil {
			serverName = cfg.Name
		}
		db, err := w.repo.GetDatabase(name, serverName)
		if err != nil {
			http.Error(rw, fmt.Sprintf("Error leyendo base de datos: %v", err), http.StatusInternalServerError)
			return
		}
		if db == nil {
			http.Error(rw, "base de datos no encontrada", http.StatusNotFound)
			return
		}
		db.Password = ""
		jsonResponse(rw, db)
		return
	}

	if !w.isAuthorized(req) {
		jsonError(rw, "Operación no autorizada: API key requerida", http.StatusUnauthorized)
		return
	}

	if req.Method == http.MethodPut || req.Method == http.MethodPost {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		db, err := dto.DecodeSavedDatabase(body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		db.Name = name
		cfgSave := w.resolveTargetServer(req.URL.Query().Get("server"))
		if cfgSave != nil {
			db.ServerName = cfgSave.Name
		}
		if err := w.repo.SaveDatabase(db); err != nil {
			http.Error(rw, fmt.Sprintf("Error guardando base de datos: %v", err), http.StatusInternalServerError)
			return
		}
		jsonResponse(rw, map[string]string{"status": "updated", "name": name})
		return
	}
	if req.Method == http.MethodDelete {
		if !w.isAuthorized(req) {
			jsonError(rw, "Operación no autorizada: API key requerida", http.StatusUnauthorized)
			return
		}
		cfg := w.resolveTargetServer(req.URL.Query().Get("server"))
		deleteServerName := ""
		if cfg != nil {
			deleteServerName = cfg.Name
		}
		if cfg != nil && cfg.Host != "" {
			sshExec := repositories.NewCryptoSSHExecutor()
			if err := sshExec.Connect(*cfg); err == nil {
				defer sshExec.Close()

				// 1. Remover variables de entorno inyectadas en servicios vinculados en Swarm
				links, errLinks := w.repo.GetServiceLinks(deleteServerName)
				if errLinks != nil {
					slog.Warn("falló al obtener enlaces de servicios", "error", errLinks)
				}
				unlinkUC := usecases.NewUnlinkServicesUseCase(w.repo, sshExec)
				for _, l := range links {
					if l.TargetSvc == name || l.SourceSvc == name {
						if err := unlinkUC.Execute(l.SourceSvc, l.TargetSvc, deleteServerName); err != nil {
							slog.Warn("falló al desvincular servicio", "source", l.SourceSvc, "target", l.TargetSvc, "error", err)
						}
					}
				}

				// 2. Probar eliminación de todas las variaciones de nombres posibles en Docker Swarm
				cleanName := strings.TrimPrefix(name, "tarhiata-db-")
				cleanName = strings.TrimPrefix(cleanName, "tarhiata-")

				dbService1 := fmt.Sprintf("tarhiata-db-%s", cleanName)
				dbService2 := fmt.Sprintf("tarhiata-db-%s", name)

				cmd := fmt.Sprintf("docker service rm %s || docker service rm %s || docker service rm %s || docker rm -f %s || docker rm -f %s || docker rm -f %s",
					dbService1, dbService2, name, dbService1, dbService2, name)
				if _, err := sshExec.RunCommand(cmd); err != nil {
					slog.Debug("aviso al remover contenedor/servicio ssh de base de datos", "cmd", cmd, "error", err)
				}
			}
		}

		cleanName := strings.TrimPrefix(name, "tarhiata-db-")
		cleanName = strings.TrimPrefix(cleanName, "tarhiata-")

		if err := w.repo.DeleteDatabase(name, deleteServerName); err != nil {
			slog.Debug("aviso al eliminar base de datos por nombre", "name", name, "error", err)
		}
		if err := w.repo.DeleteDatabase(cleanName, deleteServerName); err != nil {
			slog.Debug("aviso al eliminar base de datos por cleanName", "cleanName", cleanName, "error", err)
		}
		if err := w.repo.DeleteService(name, deleteServerName); err != nil {
			slog.Debug("aviso al eliminar servicio por nombre", "name", name, "error", err)
		}
		if err := w.repo.DeleteService(cleanName, deleteServerName); err != nil {
			slog.Debug("aviso al eliminar servicio por cleanName", "cleanName", cleanName, "error", err)
		}

		jsonResponse(rw, map[string]string{"status": "deleted", "name": name})
		return
	}
}

type sshKeyInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (w *WebServer) handleListSSHKeys(rw http.ResponseWriter, req *http.Request) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		jsonResponse(rw, []sshKeyInfo{})
		return
	}
	sshDir := filepath.Join(homeDir, ".ssh")
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		jsonResponse(rw, []sshKeyInfo{})
		return
	}

	var keys []sshKeyInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".pub") || strings.HasSuffix(name, ".old") ||
			name == "known_hosts" || name == "known_hosts.old" || name == "config" ||
			name == ".DS_Store" || strings.HasPrefix(name, "google_compute") {
			continue
		}

		relPath := fmt.Sprintf("~/.ssh/%s", name)
		info, err := entry.Info()
		if err == nil && info.Size() > 0 {
			keys = append(keys, sshKeyInfo{
				Name: name,
				Path: relPath,
			})
		}
	}
	jsonResponse(rw, keys)
}

func (w *WebServer) handleBootstrap(rw http.ResponseWriter, req *http.Request) {
	var reqData struct {
		Host                 string `json:"host"`
		Port                 int    `json:"port"`
		User                 string `json:"user"`
		PrivateKey           string `json:"keyPath"`
		AcmeEmail            string `json:"acmeEmail"`
		InstallObservability bool   `json:"installObservability"`
	}
	if req.Body != nil {
		if err := json.NewDecoder(req.Body).Decode(&reqData); err != nil && err != io.EOF {
			slog.Warn("cuerpo de solicitud bootstrap no es JSON válido", "error", err)
		}
	}

	// Configurar streaming NDJSON
	flusher, ok := setupStreaming(rw)
	if !ok {
		http.Error(rw, "Streaming no soportado", http.StatusInternalServerError)
		return
	}
	send := func(t, m string) {
		streamJSON(rw, flusher, t, m)
	}

	serverTarget := req.URL.Query().Get("server")
	if reqData.Host == "" && serverTarget != "" {
		if found := w.resolveTargetServer(serverTarget); found != nil && found.Host != "" {
			reqData.Host = found.Host
			reqData.Port = found.Port
			reqData.User = found.User
			reqData.PrivateKey = found.PrivateKey
		}
	}

	if reqData.Host != "" {
		if reqData.Port <= 0 {
			reqData.Port = 22
		}
		if reqData.User == "" {
			reqData.User = "root"
		}
		if reqData.PrivateKey == "" {
			reqData.PrivateKey = "~/.ssh/id_rsa"
		}
		cfg := domain.ServerConfig{
			Host:       reqData.Host,
			Port:       reqData.Port,
			User:       reqData.User,
			PrivateKey: reqData.PrivateKey,
		}
		curr := w.getConfig()
		if curr != nil {
			cfg.DOAPIToken = curr.DOAPIToken
			cfg.VultrAPIToken = curr.VultrAPIToken
			cfg.CloudProvider = curr.CloudProvider
		}
		if err := w.repo.SaveServerConfig(cfg); err != nil {
			send("error", fmt.Sprintf("❌ Error guardando configuración: %v", err))
			return
		}
		w.setConfig(&cfg)
		send("log", fmt.Sprintf("💾 Configuración guardada → %s@%s:%d", cfg.User, cfg.Host, cfg.Port))
	}

	cfg := w.getConfig()
	if cfg == nil || cfg.Host == "" {
		send("error", "❌ No hay ningún VPS configurado ni IP especificada")
		return
	}

	send("step", "🔗 [1/3] Conectando por SSH a " + cfg.Host + "...")
	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*cfg); err != nil {
		send("error", fmt.Sprintf("❌ Falló conexión SSH a %s: %v", cfg.Host, err))
		return
	}
	defer sshExec.Close()
	send("log", fmt.Sprintf("✅ Conexión SSH establecida con %s", cfg.Host))

	send("step", "🔧 [2/3] Ejecutando inicialización del framework...")
	loggingExec := NewLoggingSSHExecutor(sshExec, send)
	bootstrapper := usecases.NewInitServerUseCase(loggingExec)
	acme := reqData.AcmeEmail
	if err := bootstrapper.Execute(acme); err != nil {
		send("error", fmt.Sprintf("❌ Falló inicialización del framework: %v", err))
		return
	}
	send("log", "✅ Framework base instalado correctamente")

	if reqData.InstallObservability {
		send("step", "📊 [3/3] Desplegando stack de observabilidad...")
		obsUC := usecases.NewDeployObservabilityUseCase(loggingExec)
		if err := obsUC.Execute(true); err != nil {
			send("log", fmt.Sprintf("⚠️ Despliegue de observabilidad reportó: %v", err))
		} else {
			send("log", "✅ Observabilidad desplegada (Portainer, Dozzle, Grafana)")
		}
	}

	streamDoneJSON(rw, flusher, map[string]string{
		"status":  "bootstrapped",
		"host":    cfg.Host,
		"message": fmt.Sprintf("¡Framework inicializado con éxito en %s!", cfg.Host),
	})
}

func (w *WebServer) handleCreateVMBootstrap(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var reqData struct {
		Provider             string `json:"provider"`
		ApiToken             string `json:"apiToken"`
		NodeName             string `json:"nodeName"`
		Region               string `json:"region"`
		AcmeEmail            string `json:"acmeEmail"`
		InstallObservability bool   `json:"installObservability"`
	}
	if err := json.NewDecoder(req.Body).Decode(&reqData); err != nil {
		http.Error(rw, "JSON inválido", http.StatusBadRequest)
		return
	}
	if reqData.ApiToken == "" {
		http.Error(rw, "Se requiere un Token de API de Nube (Vultr o DigitalOcean)", http.StatusBadRequest)
		return
	}

	// Configurar streaming NDJSON
	flusher, ok := setupStreaming(rw)
	if !ok {
		http.Error(rw, "Streaming no soportado", http.StatusInternalServerError)
		return
	}
	send := func(t, m string) {
		streamJSON(rw, flusher, t, m)
	}

	// Defaults
	if reqData.NodeName == "" {
		reqData.NodeName = "master-1"
	}
	if reqData.Provider == "" {
		reqData.Provider = "digitalocean"
	}
	if reqData.Region == "" {
		if reqData.Provider == "digitalocean" {
			reqData.Region = "nyc1"
		} else {
			reqData.Region = "ewr"
		}
	}

	send("step", "🚀 Iniciando aprovisionamiento de VM en la nube...")
	send("log", fmt.Sprintf("☁️  Proveedor: %s", strings.ToUpper(reqData.Provider)))
	send("log", fmt.Sprintf("📍 Región: %s", reqData.Region))
	send("log", fmt.Sprintf("🏷️  Nodo: %s", reqData.NodeName))
	if reqData.AcmeEmail != "" {
		send("log", fmt.Sprintf("🔐 SSL ACME Email: %s", reqData.AcmeEmail))
	} else {
		send("log", "🔓 SSL: Deshabilitado (modo HTTP)")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	workspace := filepath.Join(homeDir, ".config", "tarhiata", "terraform", reqData.Provider+"_"+reqData.NodeName)

	providerName := "vultr"
	if reqData.Provider == "digitalocean" {
		providerName = "digitalocean"
	}
	provisioner := cloud.NewProvisioner(providerName, workspace)

	send("step", "⏳ [1/5] Aprovisionando VM con Terraform (1-3 minutos)...")
	send("log", "📦 Descargando providers y preparando infraestructura...")
	provRes, err := provisioner.ProvisionNode(reqData.ApiToken, reqData.NodeName, reqData.Region, "")
	if err != nil {
		send("error", fmt.Sprintf("❌ Falló aprovisionamiento de la VM: %v", err))
		return
	}
	newIP := provRes.PublicIP
	privKeyContent := provRes.PrivateKey
	send("log", fmt.Sprintf("✅ VM creada exitosamente — IP pública: %s", newIP))

	// Guardar llave privada localmente
	keyDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		slog.Warn("falló al crear directorio de llaves ssh", "dir", keyDir, "error", err)
	}
	keyPath := filepath.Join(keyDir, "tarhiata_master_"+reqData.NodeName+".pem")
	if privKeyContent != "" {
		if err := os.WriteFile(keyPath, []byte(privKeyContent), 0600); err != nil {
			slog.Warn("falló al guardar llave ssh privada", "path", keyPath, "error", err)
		} else {
			send("log", fmt.Sprintf("🔑 Llave SSH guardada → %s", keyPath))
		}
	}

	// Guardar Configuración
	cfg := domain.ServerConfig{
		Host:          newIP,
		Port:          22,
		User:          "root",
		PrivateKey:    keyPath,
		CloudProvider: reqData.Provider,
	}
	if reqData.Provider == "digitalocean" {
		cfg.DOAPIToken = reqData.ApiToken
	} else {
		cfg.VultrAPIToken = reqData.ApiToken
	}
	if err := w.repo.SaveServerConfig(cfg); err != nil {
		send("error", fmt.Sprintf("❌ Error guardando configuración: %v", err))
		return
	}
	w.setConfig(&cfg)
	send("log", "💾 Configuración del servidor guardada localmente")

	// Conectar SSH con reintentos
	send("step", "⏳ [2/5] Conectando por SSH (esperando que la VM arranque)...")
	sshExec := repositories.NewCryptoSSHExecutor()
	var connected bool
	for i := 0; i < 18; i++ {
		if err := sshExec.Connect(cfg); err == nil {
			connected = true
			break
		}
		send("log", fmt.Sprintf("🔄 Reintento SSH %d/18 — VM arrancando...", i+1))
		time.Sleep(10 * time.Second)
	}
	if !connected {
		send("error", fmt.Sprintf("❌ SSH no respondió tras 3 minutos en %s — verificar que la VM esté encendida", newIP))
		return
	}
	defer sshExec.Close()
	send("log", fmt.Sprintf("✅ Conexión SSH establecida con %s", newIP))

	// Bootstrap con logging de absolutamente todo
	send("step", "⏳ [3/5] Instalando framework (Docker, Swarm, Traefik, UFW)...")
	loggingExec := NewLoggingSSHExecutor(sshExec, send)
	bootstrapper := usecases.NewInitServerUseCase(loggingExec)
	acme := reqData.AcmeEmail
	if err := bootstrapper.Execute(acme); err != nil {
		send("error", fmt.Sprintf("❌ Bootstrap falló: %v", err))
		return
	}
	send("log", "✅ Framework completo instalado correctamente")

	if reqData.InstallObservability {
		send("step", "⏳ [4/5] Desplegando observabilidad (Portainer, Dozzle, Grafana)...")
		obsUC := usecases.NewDeployObservabilityUseCase(loggingExec)
		if err := obsUC.Execute(true); err != nil {
			send("log", fmt.Sprintf("⚠️ Despliegue de observabilidad reportó: %v", err))
		} else {
			send("log", "✅ Stack de observabilidad desplegado")
		}
	}

	send("step", "✅ [5/5] ¡Proceso completado exitosamente!")

	streamDoneJSON(rw, flusher, map[string]string{
		"status":  "vm_created_and_bootstrapped",
		"host":    newIP,
		"node":    reqData.NodeName,
		"region":  reqData.Region,
		"message": fmt.Sprintf("¡VM '%s' en %s (%s) creada y framework instalado con éxito!", reqData.NodeName, newIP, reqData.Region),
	})
}

func (w *WebServer) handleWorkerProvision(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var reqData struct {
		NodeName  string `json:"nodeName"`
		Plan      string `json:"plan"`
		Region    string `json:"region"`
		LabelType string `json:"labelType"`
		APIKey    string `json:"apiKey"`
		Provider  string `json:"provider"`
	}
	if req.Body != nil {
		if err := json.NewDecoder(req.Body).Decode(&reqData); err != nil && err != io.EOF {
			slog.Warn("cuerpo JSON inválido en worker provision", "error", err)
		}
	}

	flusher, ok := setupStreaming(rw)
	if !ok {
		http.Error(rw, "Streaming no soportado", http.StatusInternalServerError)
		return
	}
	send := func(t, m string) {
		streamJSON(rw, flusher, t, m)
	}

	if reqData.NodeName == "" {
		reqData.NodeName = "worker-1"
	}
	if reqData.LabelType == "" {
		reqData.LabelType = "worker"
	}

	serverTarget := req.URL.Query().Get("server")
	cfg := w.resolveTargetServer(serverTarget)
	if cfg == nil || (cfg.Host == "" && !isLocalConfig(cfg)) {
		send("error", "❌ No hay ningún VPS Manager configurado")
		return
	}

	if reqData.Provider != "" {
		cfg.CloudProvider = reqData.Provider
	}
	if reqData.APIKey != "" {
		if cfg.CloudProvider == "digitalocean" {
			cfg.DOAPIToken = reqData.APIKey
		}
		if cfg.CloudProvider != "digitalocean" {
			cfg.VultrAPIToken = reqData.APIKey
		}
		if err := w.repo.SaveServerConfig(*cfg); err != nil {
			send("log", fmt.Sprintf("⚠️ No se pudo persistir API Key en repositorio: %v", err))
		}
	}

	token := cfg.VultrAPIToken
	if token == "" {
		token = cfg.DOAPIToken
	}
	if token == "" {
		send("error", "❌ Se requiere una API Key / Token Cloud (Vultr o DigitalOcean). Por favor ingrésala en el formulario.")
		return
	}

	if reqData.Region == "" {
		reqData.Region = "mex"
		if cfg.CloudProvider == "digitalocean" {
			reqData.Region = "nyc1"
		}
	}

	send("step", fmt.Sprintf("🚀 Iniciando aprovisionamiento del Nodo Worker '%s'...", reqData.NodeName))
	send("log", fmt.Sprintf("🏷️  Tipo de Nodo: %s", reqData.LabelType))
	send("log", fmt.Sprintf("📍 Región: %s", reqData.Region))
	if reqData.Plan != "" {
		send("log", fmt.Sprintf("💵 Plan Vultr: %s", reqData.Plan))
	}

	send("step", "🔗 [1/6] Conectando por SSH al Manager...")
	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*cfg); err != nil {
		send("error", fmt.Sprintf("❌ Falló conexión SSH al Manager: %v", err))
		return
	}
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("web_server: error cerrando sshExec en provisionWorker", "error", clErr)
		}
	}()

	loggingExec := NewLoggingSSHExecutor(sshExec, send)
	workerUseCase := usecases.NewProvisionWorkerUseCase(loggingExec)

	send("step", "🏗️  [2/6] Aprovisionando VM Worker con Terraform y uniendo al clúster...")
	nodeIP, err := workerUseCase.ExecuteWithPlanAndRegion(*cfg, reqData.NodeName, reqData.LabelType, reqData.Plan, reqData.Region)
	if err != nil {
		send("error", fmt.Sprintf("❌ Falló aprovisionamiento del Worker: %v", err))
		return
	}

	// Registrar el nuevo VPS Worker en el catálogo de servidores
	homeDir, errHome := os.UserHomeDir()
	if errHome != nil {
		slog.Warn("web_server: error obteniendo homeDir para workerKeyPath", "error", errHome)
		homeDir = "."
	}
	workerKeyPath := filepath.Join(homeDir, ".ssh", "tarhiata_worker_"+reqData.NodeName+".pem")
	workerServer := domain.ServerConfig{
		Name:          reqData.NodeName,
		Host:          nodeIP,
		Port:          22,
		User:          "root",
		PrivateKey:    workerKeyPath,
		CloudProvider: cfg.CloudProvider,
		VultrAPIToken: cfg.VultrAPIToken,
		DOAPIToken:    cfg.DOAPIToken,
		IsActive:      false,
	}
	if errSave := w.repo.SaveServerConfig(workerServer); errSave != nil {
		send("log", fmt.Sprintf("⚠️ No se pudo registrar Worker en catálogo de servidores: %v", errSave))
	}
	send("log", fmt.Sprintf("💾 Nuevo VPS Worker '%s' (%s) registrado en el catálogo de servidores", reqData.NodeName, nodeIP))

	send("step", "✅ [6/6] ¡Nodo Worker aprovisionado y unido al clúster exitosamente!")
	streamDoneJSON(rw, flusher, map[string]string{
		"status":  "worker_provisioned",
		"nodeIp":  nodeIP,
		"region":  reqData.Region,
		"message": fmt.Sprintf("¡Nodo Worker '%s' en %s (%s) añadido al clúster y catálogo con éxito!", reqData.NodeName, nodeIP, reqData.Region),
	})
}

func (w *WebServer) handleObservability(rw http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		obs, err := w.repo.GetObservability()
		if err != nil || obs == nil {
			jsonResponse(rw, map[string]interface{}{"enabled": false})
			return
		}
		jsonResponse(rw, map[string]interface{}{
			"enabled":          true,
			"deploy_type":      obs.DeployType,
			"external_url":     obs.ExternalURL,
			"grafana_password": obs.GrafanaPassword,
		})
		return
	}

	if req.Method != http.MethodPost {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var reqData struct {
		Action          string `json:"action"`          // "deploy" or "delete"
		Enabled         *bool  `json:"enabled"`         // true/false
		ExposePublic    bool   `json:"exposePublic"`    // public or internal VPN
		DeployType      string `json:"deployType"`      // "single-node" or "multi-node"
		GrafanaPassword string `json:"grafanaPassword"` // Grafana/Portainer password
		VolumePath      string `json:"volumePath"`      // Custom external VM volume path (e.g., /opt/data/obs)
	}
	if err := json.NewDecoder(req.Body).Decode(&reqData); err != nil {
		jsonError(rw, "payload json inválido", http.StatusBadRequest)
		return
	}

	if reqData.VolumePath == "" {
		reqData.VolumePath = "/opt/data/obs"
	}
	if reqData.GrafanaPassword == "" {
		reqData.GrafanaPassword = "admin"
	}
	if reqData.DeployType == "" {
		reqData.DeployType = "single-node"
	}

	shouldDisable := (reqData.Action == "delete") || (reqData.Enabled != nil && !*reqData.Enabled)

	if w.config == nil || w.config.Host == "" {
		jsonError(rw, "VPS no configurado", http.StatusBadRequest)
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*w.config); err != nil {
		jsonError(rw, fmt.Sprintf("Error SSH: %v", err), http.StatusInternalServerError)
		return
	}
	defer sshExec.Close()

	if shouldDisable {
		if _, err := sshExec.RunCommand("docker stack rm tarhiata_obs"); err != nil {
			slog.Warn("aviso al remover stack tarhiata_obs", "error", err)
		}
		if err := w.repo.DeleteObservability(); err != nil {
			slog.Warn("aviso al eliminar observabilidad local", "error", err)
		}
		jsonResponse(rw, map[string]string{"status": "observability_disabled"})
		return
	}

	obsUC := usecases.NewDeployObservabilityUseCase(sshExec)
	err := obsUC.ExecutePersistentWithVolume(reqData.ExposePublic, reqData.DeployType, reqData.GrafanaPassword, reqData.VolumePath)
	if err != nil {
		jsonError(rw, fmt.Sprintf("Error al desplegar observabilidad: %v", err), http.StatusInternalServerError)
		return
	}

	obsRecord := domain.SavedObservability{
		DeployType:      reqData.DeployType,
		ExternalURL:     reqData.VolumePath,
		GrafanaPassword: reqData.GrafanaPassword,
	}
	if err := w.repo.SaveObservability(obsRecord); err != nil {
		slog.Warn("falló al guardar registro de observabilidad", "error", err)
	}

	jsonResponse(rw, map[string]string{
		"status":     "observability_deployed",
		"volumePath": reqData.VolumePath,
	})
}

func (w *WebServer) handleServiceRollback(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var reqData struct {
		Name   string `json:"name"`
		Server string `json:"server,omitempty"`
	}
	err := json.NewDecoder(req.Body).Decode(&reqData)
	if err != nil || strings.TrimSpace(reqData.Name) == "" {
		jsonError(rw, "Parámetro 'name' requerido", http.StatusBadRequest)
		return
	}

	serviceName := strings.TrimSpace(reqData.Name)
	if !isValidNodeID(serviceName) {
		jsonError(rw, "Nombre de servicio inválido", http.StatusBadRequest)
		return
	}

	serverTarget := req.URL.Query().Get("server")
	if serverTarget == "" && reqData.Server != "" {
		serverTarget = reqData.Server
	}
	cfg := w.resolveTargetServer(serverTarget)
	if cfg == nil || (cfg.Host == "" && !isLocalConfig(cfg)) {
		jsonError(rw, "VPS no configurado", http.StatusBadRequest)
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*cfg); err != nil {
		jsonError(rw, fmt.Sprintf("Error SSH: %v", err), http.StatusInternalServerError)
		return
	}
	defer sshExec.Close()

	rollbackCmd := fmt.Sprintf("docker service rollback %s || docker service rollback %s_%s || docker service rollback tarhiata-app-%s || docker service rollback tarhiata_%s",
		serviceName, serviceName, serviceName, serviceName, serviceName)
	res, err := sshExec.RunCommand(rollbackCmd)
	if err != nil || res == nil || res.ExitCode != 0 {
		out := ""
		if res != nil {
			out = res.Output
		}
		if out == "" && err != nil {
			out = err.Error()
		}
		jsonError(rw, fmt.Sprintf("Error al realizar rollback: %s", out), http.StatusInternalServerError)
		return
	}

	outStr := ""
	if res != nil {
		outStr = res.Output
	}
	jsonResponse(rw, map[string]string{
		"status":  "rolled_back",
		"service": serviceName,
		"output":  outStr,
	})
}

func (w *WebServer) handleServiceRestart(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	name := strings.TrimSpace(req.URL.Query().Get("name"))
	serverTarget := strings.TrimSpace(req.URL.Query().Get("server"))
	if req.Body != nil {
		var reqData struct {
			Name   string `json:"name"`
			Server string `json:"server,omitempty"`
		}
		if err := json.NewDecoder(req.Body).Decode(&reqData); err != nil {
			slog.Debug("handleServiceRestart: json decode omitido o inválido", "error", err)
		} else {
			if name == "" {
				name = strings.TrimSpace(reqData.Name)
			}
			if serverTarget == "" {
				serverTarget = strings.TrimSpace(reqData.Server)
			}
		}
	}

	if name == "" {
		jsonError(rw, "Parámetro 'name' requerido", http.StatusBadRequest)
		return
	}

	validNameRegex := regexp.MustCompile(`^[a-zA-Z0-9_\.\-]+$`)
	if !validNameRegex.MatchString(name) {
		jsonError(rw, "Nombre de servicio inválido", http.StatusBadRequest)
		return
	}

	cfg := w.resolveTargetServer(serverTarget)
	if cfg == nil || (cfg.Host == "" && !isLocalConfig(cfg)) {
		jsonError(rw, "VPS no configurado", http.StatusBadRequest)
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*cfg); err != nil {
		jsonError(rw, fmt.Sprintf("Error SSH: %v", err), http.StatusInternalServerError)
		return
	}
	defer sshExec.Close()

	candidates := []string{
		name,
		"tarhiata-db-" + name,
		"tarhiata-app-" + name,
		"tarhiata_" + name,
	}

	for _, cand := range candidates {
		// 1. Intentar docker service update --force (Docker Swarm)
		resSwarm, errSwarm := sshExec.RunCommand(fmt.Sprintf("docker service update --force %s 2>&1", cand))
		if errSwarm == nil && resSwarm.ExitCode == 0 && !isDockerError(resSwarm.Output) {
			jsonResponse(rw, map[string]string{
				"status":  "restarted",
				"type":    "swarm_service",
				"service": cand,
				"output":  strings.TrimSpace(resSwarm.Output),
			})
			return
		}

		// 2. Intentar docker restart directo
		resRestart, errRestart := sshExec.RunCommand(fmt.Sprintf("docker restart %s 2>&1", cand))
		if errRestart == nil && resRestart.ExitCode == 0 && !isDockerError(resRestart.Output) {
			jsonResponse(rw, map[string]string{
				"status":  "restarted",
				"type":    "docker_container",
				"service": cand,
				"output":  strings.TrimSpace(resRestart.Output),
			})
			return
		}

		// 3. Intentar por filtro de contenedor ID
		resPs, errPs := sshExec.RunCommand(fmt.Sprintf("docker ps -a -q --filter name=%s | head -n 1", cand))
		if errPs == nil && strings.TrimSpace(resPs.Output) != "" {
			cid := strings.TrimSpace(resPs.Output)
			resCid, errCid := sshExec.RunCommand(fmt.Sprintf("docker restart %s 2>&1", cid))
			if errCid == nil && resCid.ExitCode == 0 && !isDockerError(resCid.Output) {
				jsonResponse(rw, map[string]string{
					"status":  "restarted",
					"type":    "docker_container_by_id",
					"service": cand,
					"output":  strings.TrimSpace(resCid.Output),
				})
				return
			}
		}
	}

	// 4. Intentar systemctl restart si es una unidad del host
	if strings.HasSuffix(name, ".service") || !strings.Contains(name, "-") {
		resSys, errSys := sshExec.RunCommand(fmt.Sprintf("systemctl restart %s 2>&1", name))
		if errSys == nil && resSys.ExitCode == 0 {
			jsonResponse(rw, map[string]string{
				"status":  "restarted",
				"type":    "systemd_service",
				"service": name,
				"output":  strings.TrimSpace(resSys.Output),
			})
			return
		}
	}

	jsonError(rw, fmt.Sprintf("No se pudo reiniciar el servicio o contenedor '%s'. Verifique que esté en ejecución.", name), http.StatusInternalServerError)
}



func (w *WebServer) handleServerUpdate(rw http.ResponseWriter, req *http.Request) {
	serverTarget := req.URL.Query().Get("server")
	cfg := w.resolveTargetServer(serverTarget)
	if cfg != nil && cfg.Host != "" {
		sshExec := repositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(*cfg); err == nil {
			defer sshExec.Close()
			if err := usecases.NewUpdateServerUseCase(sshExec).Execute(); err != nil {
				slog.Warn("falló actualización del servidor", "error", err)
			}
		}
	}
	jsonResponse(rw, map[string]string{"status": "server_updated"})
}

func (w *WebServer) handlePrune(rw http.ResponseWriter, req *http.Request) {
	serverTarget := req.URL.Query().Get("server")
	cfg := w.resolveTargetServer(serverTarget)
	if (cfg == nil || cfg.Host == "") && serverTarget == "" {
		if loaded, err := w.repo.GetServerConfig(); err == nil && loaded != nil && loaded.Host != "" {
			w.setConfig(loaded)
			cfg = loaded
		}
	}

	if cfg == nil || cfg.Host == "" {
		http.Error(rw, "Error: VPS Master no configurado. Ingresa la IP del servidor y la llave SSH en Configuración VPS.", http.StatusBadRequest)
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*cfg); err != nil {
		http.Error(rw, fmt.Sprintf("Error de conexión SSH con host %s: %v", cfg.Host, err), http.StatusInternalServerError)
		return
	}
	defer sshExec.Close()

	uc := usecases.NewPruneSystemUseCase(sshExec)
	output, err := uc.Execute()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(rw, map[string]string{
		"status": "ok",
		"output": output,
	})
}

func (w *WebServer) handleRestartTraefik(rw http.ResponseWriter, req *http.Request) {
	serverTarget := req.URL.Query().Get("server")
	cfg := w.resolveTargetServer(serverTarget)
	if (cfg == nil || cfg.Host == "") && serverTarget == "" {
		if loaded, err := w.repo.GetServerConfig(); err == nil && loaded != nil && loaded.Host != "" {
			w.setConfig(loaded)
			cfg = loaded
		}
	}

	if cfg == nil || cfg.Host == "" {
		http.Error(rw, "Error: VPS Master no configurado.", http.StatusBadRequest)
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*cfg); err != nil {
		http.Error(rw, fmt.Sprintf("Error de conexión SSH con host %s: %v", cfg.Host, err), http.StatusInternalServerError)
		return
	}
	defer sshExec.Close()

	uc := usecases.NewRestartTraefikUseCase(sshExec)
	output, err := uc.Execute()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(rw, map[string]string{
		"status": "ok",
		"output": output,
	})
}

func (w *WebServer) handleRepairTraefik(rw http.ResponseWriter, req *http.Request) {
	var reqData struct {
		AcmeEmail string `json:"acmeEmail"`
	}
	if req.Body != nil {
		if err := json.NewDecoder(req.Body).Decode(&reqData); err != nil && err != io.EOF {
			slog.Warn("cuerpo de solicitud repair-traefik no es JSON válido", "error", err)
		}
	}

	serverTarget := req.URL.Query().Get("server")
	cfg := w.resolveTargetServer(serverTarget)
	if (cfg == nil || cfg.Host == "") && serverTarget == "" {
		if loaded, err := w.repo.GetServerConfig(); err == nil && loaded != nil && loaded.Host != "" {
			w.setConfig(loaded)
			cfg = loaded
		}
	}

	if cfg == nil || cfg.Host == "" {
		http.Error(rw, "Error: VPS Master no configurado.", http.StatusBadRequest)
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*cfg); err != nil {
		http.Error(rw, fmt.Sprintf("Error de conexión SSH con host %s: %v", cfg.Host, err), http.StatusInternalServerError)
		return
	}
	defer sshExec.Close()

	uc := usecases.NewRepairTraefikUseCase(sshExec)
	output, err := uc.Execute(reqData.AcmeEmail)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(rw, map[string]string{
		"status": "ok",
		"output": output,
	})
}

func (w *WebServer) handleTopology(rw http.ResponseWriter, req *http.Request) {
	cfg := w.resolveTargetServer(req.URL.Query().Get("server"))
	serverName := ""
	if cfg != nil {
		serverName = cfg.Name
	}
	services, errSvc := w.repo.GetServices(serverName)
	if errSvc != nil {
		slog.Warn("web_server: error leyendo servicios en handleTopology", "error", errSvc)
	}
	databases, errDB := w.repo.GetDatabases(serverName)
	if errDB != nil {
		slog.Warn("web_server: error leyendo bases de datos en handleTopology", "error", errDB)
	}
	links, errLinks := w.repo.GetServiceLinks(serverName)
	if errLinks != nil {
		slog.Warn("web_server: error leyendo service links en handleTopology", "error", errLinks)
	}

	if services == nil { services = []domain.SavedService{} }
	if databases == nil { databases = []domain.SavedDatabase{} }
	if links == nil { links = []domain.ServiceLink{} }

	if strings.Contains(req.Header.Get("Accept"), "application/json") || req.URL.Query().Get("format") == "json" {
		jsonResponse(rw, map[string]interface{}{
			"services":  services,
			"databases": databases,
			"links":     links,
		})
		return
	}

	var sb strings.Builder
	sb.WriteString("========================================================\n")
	sb.WriteString("      🗺️   T A R H I A T A   T O P O L O G Y   M A P    \n")
	sb.WriteString("========================================================\n\n")

	for _, s := range services {
		sb.WriteString(fmt.Sprintf("🚀 SERVICIO: %s\n", s.Name))
		sb.WriteString(fmt.Sprintf(" ├─ 🔌 DNS Interno : http://%s:%d\n", s.Name, s.Port))
		if s.Expose {
			proto := "http"
			if s.EnableSSL { proto = "https" }
			sb.WriteString(fmt.Sprintf(" ├─ 🌐 Red Pública : %s://%s\n", proto, s.Domain))
		} else {
			sb.WriteString(" ├─ 🔒 Red Pública : [ACCESO DENEGADO - Privado]\n")
		}
		sb.WriteString("\n")
	}

	for _, db := range databases {
		pass := db.Password
		if pass == "" {
			pass = "******"
		}
		sb.WriteString(fmt.Sprintf("🗄️ BASE DE DATOS: %s (%s)\n", db.Name, db.Engine))
		sb.WriteString(fmt.Sprintf(" ├─ 🔌 DNS Interno : %s://user:%s@tarhiata-db-%s:%d/%s\n\n", db.Engine, pass, db.Name, db.InternalPort, db.Name))
	}

	rw.Header().Set("Content-Type", "text/plain")
	rw.Write([]byte(sb.String()))
}

func (w *WebServer) handleLinks(rw http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		cfg := w.resolveTargetServer(req.URL.Query().Get("server"))
		serverName := ""
		if cfg != nil {
			serverName = cfg.Name
		}
		links, err := w.repo.GetServiceLinks(serverName)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		if links == nil { links = []domain.ServiceLink{} }
		jsonResponse(rw, links)
		return
	}

	if !w.isAuthorized(req) {
		jsonError(rw, "Operación no autorizada: API key requerida", http.StatusUnauthorized)
		return
	}

	if req.Method == "POST" {
		var reqData struct {
			SourceSvc       string `json:"sourceSvc"`
			TargetSvc       string `json:"targetSvc"`
			EnvVarName      string `json:"envVarName"`
			SourceSvcSnake  string `json:"source_svc"`
			TargetSvcSnake  string `json:"target_svc"`
			EnvVarNameSnake string `json:"env_var_name"`
		}
		if err := json.NewDecoder(req.Body).Decode(&reqData); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		if reqData.SourceSvc == "" { reqData.SourceSvc = reqData.SourceSvcSnake }
		if reqData.TargetSvc == "" { reqData.TargetSvc = reqData.TargetSvcSnake }
		if reqData.EnvVarName == "" { reqData.EnvVarName = reqData.EnvVarNameSnake }

		cfg := w.resolveTargetServer(req.URL.Query().Get("server"))

		var sshExec ports.SSHExecutor
		if cfg != nil && cfg.Host != "" {
			se := repositories.NewCryptoSSHExecutor()
			if err := se.Connect(*cfg); err == nil {
				sshExec = se
				defer se.Close()
			}
		}

		serverName := ""
		if cfg != nil {
			serverName = cfg.Name
		}
		linkUseCase := usecases.NewLinkServicesUseCase(w.repo, sshExec)
		link, err := linkUseCase.Execute(reqData.SourceSvc, reqData.TargetSvc, reqData.EnvVarName, serverName)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(rw, link)
		return
	}

	if req.Method == "DELETE" {
		sourceSvc := req.URL.Query().Get("source_svc")
		targetSvc := req.URL.Query().Get("target_svc")
		if sourceSvc == "" || targetSvc == "" {
			http.Error(rw, "se requieren parámetros source_svc y target_svc", http.StatusBadRequest)
			return
		}

		cfg := w.resolveTargetServer(req.URL.Query().Get("server"))

		var sshExec ports.SSHExecutor
		if cfg != nil && cfg.Host != "" {
			se := repositories.NewCryptoSSHExecutor()
			if err := se.Connect(*cfg); err == nil {
				sshExec = se
				defer se.Close()
			}
		}

		serverName := ""
		if cfg != nil {
			serverName = cfg.Name
		}
		unlinkUseCase := usecases.NewUnlinkServicesUseCase(w.repo, sshExec)
		if err := unlinkUseCase.Execute(sourceSvc, targetSvc, serverName); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(rw, map[string]string{"status": "deleted"})
		return
	}
}

func jsonResponse(rw http.ResponseWriter, data interface{}) {
	if err := httputil.JSON(rw, http.StatusOK, data); err != nil {
		slog.Error("web_server: error serializando jsonResponse", "error", err)
	}
}

func jsonError(rw http.ResponseWriter, message string, statusCode int) {
	if err := httputil.Error(rw, statusCode, message); err != nil {
		slog.Error("web_server: error serializando jsonError", "error", err)
	}
}

// streamJSON envía un evento de progreso al cliente en formato NDJSON.
func streamJSON(rw http.ResponseWriter, flusher http.Flusher, eventType, msg string) {
	if err := httputil.WriteEvent(rw, flusher, eventType, msg); err != nil {
		slog.Warn("web_server: error serializando evento streamJSON", "error", err)
	}
}

// streamDoneJSON envía el evento final de éxito con datos al cliente.
func streamDoneJSON(rw http.ResponseWriter, flusher http.Flusher, result map[string]string) {
	if err := httputil.WriteDone(rw, flusher, result); err != nil {
		slog.Warn("web_server: error serializando evento streamDoneJSON", "error", err)
	}
}

// setupStreaming configura los headers HTTP para streaming NDJSON y retorna el flusher.
func setupStreaming(rw http.ResponseWriter) (http.Flusher, bool) {
	flusher, err := httputil.SetupStreaming(rw)
	return flusher, err == nil
}

func openBrowser(rawURL string) {
	if err := osterminal.OpenBrowser(rawURL); err != nil {
		fmt.Printf("⚠️ Advertencia: No se pudo abrir el navegador automáticamente para %s: %v\n", rawURL, err)
	}
}

func isValidNodeID(id string) bool {
	return validator.IsNodeID(id)
}

func isAllowedTerminalCommand(cmd string) bool {
	return validator.IsSafeCommand(cmd)
}

func (w *WebServer) handleNodes(rw http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodDelete {
		nodeID := req.URL.Query().Get("id")
		if !isValidNodeID(nodeID) {
			jsonError(rw, "Parámetro 'id' inválido o no proporcionado", http.StatusBadRequest)
			return
		}
		serverTarget := req.URL.Query().Get("server")
		targetCfg := w.resolveTargetServer(serverTarget)
		if targetCfg == nil || (targetCfg.Host == "" && !isLocalConfig(targetCfg)) {
			jsonError(rw, "VPS no configurado", http.StatusBadRequest)
			return
		}
		sshExec := repositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(*targetCfg); err != nil {
			jsonError(rw, fmt.Sprintf("Error SSH: %v", err), http.StatusInternalServerError)
			return
		}
		defer func() {
			if clErr := sshExec.Close(); clErr != nil {
				slog.Warn("web_server: error cerrando sshExec en node delete", "error", clErr)
			}
		}()

		// 1. Obtener el hostname del nodo
		hostname := ""
		resHost, errHost := sshExec.RunCommand(fmt.Sprintf("docker node inspect %s --format '{{.Description.Hostname}}'", nodeID))
		if errHost != nil {
			slog.Warn("web_server: error inspeccionando hostname del nodo", "nodeID", nodeID, "error", errHost)
		} else if resHost != nil {
			hostname = strings.TrimSpace(resHost.Output)
		}

		// 2. Remover cualquier servicio de base de datos asociado a este nodo para liberar el candado de Swarm
		resServices, errServices := sshExec.RunCommand("docker service ls --format '{{.Name}}'")
		if errServices != nil {
			slog.Debug("aviso al listar servicios para remover nodo", "error", errServices)
		}
		if resServices != nil && resServices.Output != "" {
			svcs := strings.Split(strings.TrimSpace(resServices.Output), "\n")
			for _, svc := range svcs {
				svc = strings.TrimSpace(svc)
				if svc == "" {
					continue
				}
				if (hostname != "" && strings.Contains(svc, hostname)) || strings.Contains(svc, nodeID) {
					if _, err := sshExec.RunCommand(fmt.Sprintf("docker service rm %s", svc)); err != nil {
						slog.Debug("aviso al remover servicio asociado a nodo", "service", svc, "error", err)
					}
				}
			}
		}

		// 3. Cambiar disponibilidad a drain y forzar remoción del clúster Swarm
		if _, err := sshExec.RunCommand(fmt.Sprintf("docker node update --availability drain %s", nodeID)); err != nil {
			slog.Debug("aviso al poner nodo en drain", "node", nodeID, "error", err)
		}
		res, err := sshExec.RunCommand(fmt.Sprintf("docker node rm --force %s", nodeID))
		if (err != nil || res == nil || res.ExitCode != 0) && hostname != "" {
			// Intentar remover por hostname como alternativa
			res, err = sshExec.RunCommand(fmt.Sprintf("docker node rm --force %s", hostname))
		}

		if err != nil || res == nil || res.ExitCode != 0 {
			out := ""
			if res != nil {
				out = res.Output
			}
			if out == "" && err != nil {
				out = err.Error()
			}
			jsonError(rw, fmt.Sprintf("Error al remover nodo de Swarm: %s", out), http.StatusInternalServerError)
			return
		}
		jsonResponse(rw, map[string]string{"status": "node_removed", "id": nodeID})
		return
	}

	if req.Method != http.MethodGet {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// GET: List all nodes
	serverTarget := req.URL.Query().Get("server")
	targetCfg := w.resolveTargetServer(serverTarget)

	managerHost := "Local / Master VPS"
	if targetCfg != nil && targetCfg.Host != "" {
		managerHost = targetCfg.Host
	}

	nodes := []map[string]interface{}{
		{
			"id":             "manager",
			"hostname":       "manager-node",
			"name":           "Manager Node (Swarm Master)",
			"ip":             managerHost,
			"role":           "manager",
			"status":         "Ready",
			"availability":   "active",
			"is_leader":      true,
			"engine_version": "docker-swarm",
		},
	}

	if targetCfg != nil && targetCfg.Host != "" {
		sshExec := repositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(*targetCfg); err == nil {
			defer sshExec.Close()
			res, err := sshExec.RunCommand("docker node ls --format '{{.ID}}|{{.Hostname}}|{{.Status}}|{{.Availability}}|{{.ManagerStatus}}|{{.EngineVersion}}'")
			if err == nil && res.Output != "" {
				lines := strings.Split(strings.TrimSpace(res.Output), "\n")
				var realNodes []map[string]interface{}
				for _, line := range lines {
					parts := strings.Split(line, "|")
					if len(parts) >= 4 {
						id := parts[0]
						hostname := parts[1]
						status := parts[2]
						availability := parts[3]
						mgrStatus := ""
						if len(parts) >= 5 {
							mgrStatus = parts[4]
						}
						engineVer := ""
						if len(parts) >= 6 {
							engineVer = parts[5]
						}

						role := "worker"
						isLeader := false
						if strings.Contains(strings.ToLower(mgrStatus), "leader") {
							role = "manager"
							isLeader = true
						} else if strings.Contains(strings.ToLower(mgrStatus), "reachable") {
							role = "manager"
						}

						realNodes = append(realNodes, map[string]interface{}{
							"id":             id,
							"hostname":       hostname,
							"name":           fmt.Sprintf("%s (%s)", hostname, role),
							"ip":             managerHost,
							"role":           role,
							"status":         status,
							"availability":   availability,
							"is_leader":      isLeader,
							"engine_version": engineVer,
						})
					}
				}
				if len(realNodes) > 0 {
					jsonResponse(rw, realNodes)
					return
				}
			}
		}
	}

	jsonResponse(rw, nodes)
}

func (w *WebServer) handleNodeJoinToken(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	serverTarget := req.URL.Query().Get("server")
	targetCfg := w.resolveTargetServer(serverTarget)
	if targetCfg == nil || (targetCfg.Host == "" && !isLocalConfig(targetCfg)) {
		jsonError(rw, "VPS no configurado", http.StatusBadRequest)
		return
	}
	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*targetCfg); err != nil {
		jsonError(rw, fmt.Sprintf("Error SSH: %v", err), http.StatusInternalServerError)
		return
	}
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("web_server: error cerrando sshExec en node join-token", "error", clErr)
		}
	}()

	resWorker, errW := sshExec.RunCommand("docker swarm join-token worker -q")
	if errW != nil {
		slog.Warn("web_server: error obteniendo worker join-token", "error", errW)
	}
	resMgr, errM := sshExec.RunCommand("docker swarm join-token manager -q")
	if errM != nil {
		slog.Warn("web_server: error obteniendo manager join-token", "error", errM)
	}

	workerToken := ""
	if resWorker != nil {
		workerToken = strings.TrimSpace(resWorker.Output)
	}
	mgrToken := ""
	if resMgr != nil {
		mgrToken = strings.TrimSpace(resMgr.Output)
	}

	host := targetCfg.Host
	if isLocalConfig(targetCfg) {
		host = "127.0.0.1"
	}
	workerCmd := fmt.Sprintf("docker swarm join --token %s %s:2377", workerToken, host)
	mgrCmd := fmt.Sprintf("docker swarm join --token %s %s:2377", mgrToken, host)

	jsonResponse(rw, map[string]string{
		"worker_token": workerToken,
		"manager_token": mgrToken,
		"worker_cmd":   workerCmd,
		"manager_cmd":   mgrCmd,
	})
}

func (w *WebServer) handleNodeUpdate(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var input struct {
		ID           string `json:"id"`
		Availability string `json:"availability"`
		Role         string `json:"role"`
	}
	if err := json.NewDecoder(req.Body).Decode(&input); err != nil || !isValidNodeID(input.ID) {
		jsonError(rw, "Payload o ID de nodo inválido", http.StatusBadRequest)
		return
	}

	serverTarget := req.URL.Query().Get("server")
	targetCfg := w.resolveTargetServer(serverTarget)
	if targetCfg == nil || (targetCfg.Host == "" && !isLocalConfig(targetCfg)) {
		jsonError(rw, "VPS no configurado", http.StatusBadRequest)
		return
	}
	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*targetCfg); err != nil {
		jsonError(rw, fmt.Sprintf("Error SSH: %v", err), http.StatusInternalServerError)
		return
	}
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("web_server: error cerrando sshExec en node update", "error", clErr)
		}
	}()

	if input.Availability != "" {
		avail := strings.ToLower(input.Availability)
		if avail != "active" && avail != "drain" && avail != "pause" {
			jsonError(rw, "Disponibilidad inválida. Opciones: active, drain, pause", http.StatusBadRequest)
			return
		}
		res, err := sshExec.RunCommand(fmt.Sprintf("docker node update --availability %s %s", avail, input.ID))
		if err != nil || res == nil || res.ExitCode != 0 {
			out := ""
			if res != nil {
				out = res.Output
			}
			if out == "" && err != nil {
				out = err.Error()
			}
			jsonError(rw, fmt.Sprintf("Error al actualizar disponibilidad: %s", out), http.StatusInternalServerError)
			return
		}
	}

	if input.Role == "manager" {
		res, err := sshExec.RunCommand(fmt.Sprintf("docker node promote %s", input.ID))
		if err != nil || res == nil || res.ExitCode != 0 {
			out := ""
			if res != nil {
				out = res.Output
			}
			if out == "" && err != nil {
				out = err.Error()
			}
			jsonError(rw, fmt.Sprintf("Error al promover nodo a manager: %s", out), http.StatusInternalServerError)
			return
		}
	}
	if input.Role == "worker" {
		res, err := sshExec.RunCommand(fmt.Sprintf("docker node demote %s", input.ID))
		if err != nil || res == nil || res.ExitCode != 0 {
			out := ""
			if res != nil {
				out = res.Output
			}
			if out == "" && err != nil {
				out = err.Error()
			}
			jsonError(rw, fmt.Sprintf("Error al demoler nodo a worker: %s", out), http.StatusInternalServerError)
			return
		}
	}

	jsonResponse(rw, map[string]string{"status": "node_updated", "id": input.ID})
}

func (w *WebServer) handleNodeLabels(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var input struct {
		NodeID string `json:"nodeId"`
		Key    string `json:"key"`
		Value  string `json:"value"`
		Action string `json:"action"` // "add" o "remove"
	}
	if err := json.NewDecoder(req.Body).Decode(&input); err != nil || !isValidNodeID(input.NodeID) {
		jsonError(rw, "Payload o ID de nodo inválido", http.StatusBadRequest)
		return
	}

	serverTarget := req.URL.Query().Get("server")
	targetCfg := w.resolveTargetServer(serverTarget)
	if targetCfg == nil || (targetCfg.Host == "" && !isLocalConfig(targetCfg)) {
		jsonError(rw, "VPS no configurado", http.StatusBadRequest)
		return
	}
	cfg := *targetCfg
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageNodesUseCase(w.repo, sshExec)

	if input.Action == "remove" {
		if err := uc.RemoveNodeLabel(input.NodeID, input.Key, cfg); err != nil {
			jsonError(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(rw, map[string]string{"status": "success"})
		return
	}

	if err := uc.AddNodeLabel(input.NodeID, input.Key, input.Value, cfg); err != nil {
		jsonError(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(rw, map[string]string{"status": "success"})
}

func (w *WebServer) handleTerminalExec(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Name      string `json:"name"`
		Server    string `json:"server"`
		Container string `json:"container"`
		NodeID    string `json:"nodeId"`
		Command   string `json:"command"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	if payload.NodeID != "" && !isValidNodeID(payload.NodeID) {
		http.Error(rw, "nodeId inválido", http.StatusBadRequest)
		return
	}

	serverTarget := payload.Name
	if serverTarget == "" {
		serverTarget = payload.Server
	}
	if serverTarget == "" {
		serverTarget = req.URL.Query().Get("server")
	}

	cmdStr := strings.TrimSpace(payload.Command)
	if cmdStr == "" {
		jsonResponse(rw, map[string]interface{}{"output": "", "exitCode": 0, "connected": true})
		return
	}

	if !isAllowedTerminalCommand(cmdStr) {
		jsonError(rw, "Comando bloqueado por motivos de seguridad", http.StatusForbidden)
		return
	}

	if payload.Container != "" {
		sanitizedContainer := strings.TrimSpace(payload.Container)
		if !validator.IsIdentifier(sanitizedContainer) {
			jsonError(rw, "Nombre de contenedor inválido", http.StatusBadRequest)
			return
		}
		cmdStr = fmt.Sprintf("docker exec %s sh -c %q 2>&1 || docker exec %s %s", sanitizedContainer, cmdStr, sanitizedContainer, cmdStr)
	}

	targetCfg := w.resolveTargetServer(serverTarget)

	if targetCfg != nil && (targetCfg.Host != "" || isLocalConfig(targetCfg)) {
		sshExec := repositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(*targetCfg); err != nil {
			jsonResponse(rw, map[string]interface{}{
				"output":    fmt.Sprintf("Error conectando a %s: %v", targetCfg.Host, err),
				"exitCode":  1,
				"connected": false,
			})
			return
		}
		defer sshExec.Close()

		res, err := sshExec.RunCommand(cmdStr)
		if err != nil {
			errMsg := fmt.Sprintf("Error ejecutando comando: %v", err)
			if res != nil && res.Output != "" {
				errMsg += "\n" + res.Output
			}
			jsonResponse(rw, map[string]interface{}{
				"output":    errMsg,
				"exitCode":  1,
				"connected": true,
			})
			return
		}
		jsonResponse(rw, map[string]interface{}{
			"output":    res.Output,
			"exitCode":  res.ExitCode,
			"connected": true,
		})
		return
	}

	jsonResponse(rw, map[string]interface{}{
		"output":    "Error: no hay servidor configurado para ejecutar el comando",
		"exitCode":  1,
		"connected": false,
	})
}

func (w *WebServer) handleLogs(rw http.ResponseWriter, req *http.Request) {
	serviceName := strings.TrimSpace(req.URL.Query().Get("service"))
	if serviceName == "" {
		serviceName = strings.TrimSpace(req.URL.Query().Get("name"))
	}
	if serviceName == "" {
		serviceName = "api-backend"
	}

	// Sanitización estricta: solo alfanuméricos, guiones, puntos y guiones bajos
	validNameRegex := regexp.MustCompile(`^[a-zA-Z0-9_\.\-]+$`)
	if !validNameRegex.MatchString(serviceName) {
		http.Error(rw, "Nombre de servicio inválido", http.StatusBadRequest)
		return
	}

	linesParam := strings.TrimSpace(req.URL.Query().Get("lines"))
	linesInt, err := strconv.Atoi(linesParam)
	if err != nil || linesInt <= 0 || linesInt > 5000 {
		linesInt = 50
	}
	lines := strconv.Itoa(linesInt)

	serverTarget := req.URL.Query().Get("server")
	cfg := w.resolveTargetServer(serverTarget)
	if cfg == nil || cfg.Host == "" {
		jsonError(rw, "Servidor no configurado o sin Host; no se pueden obtener logs reales", http.StatusBadRequest)
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*cfg); err != nil {
		jsonError(rw, fmt.Sprintf("No se pudo conectar por SSH a '%s': %v", cfg.Name, err), http.StatusBadGateway)
		return
	}
	defer sshExec.Close()

	candidates := []string{
		serviceName,
		"tarhiata-db-" + serviceName,
		"tarhiata-app-" + serviceName,
	}

	for _, cand := range candidates {
		// 1. Intentar docker service logs
		res, err := sshExec.RunCommand(fmt.Sprintf("docker service logs --tail %s %s 2>&1", lines, cand))
		if err == nil && res.ExitCode == 0 && res.Output != "" && !isDockerError(res.Output) {
			jsonResponse(rw, map[string]string{"service": serviceName, "logs": res.Output})
			return
		}

		// 2. Intentar docker logs directo por filtro de contenedor
		resPs, errPs := sshExec.RunCommand(fmt.Sprintf("docker ps -a -q --filter name=%s | head -n 1", cand))
		if errPs == nil && strings.TrimSpace(resPs.Output) != "" {
			cid := strings.TrimSpace(resPs.Output)
			resLogs, errLogs := sshExec.RunCommand(fmt.Sprintf("docker logs --tail %s %s 2>&1", lines, cid))
			if errLogs == nil && resLogs.Output != "" && !isDockerError(resLogs.Output) {
				jsonResponse(rw, map[string]string{"service": serviceName, "logs": resLogs.Output})
				return
			}
		}
	}

	jsonError(rw, fmt.Sprintf("No se encontró ningún contenedor/servicio Docker para '%s' en '%s'", serviceName, cfg.Name), http.StatusNotFound)
}

func isDockerError(out string) bool {
	return dockerutil.IsDockerError(out)
}

func (w *WebServer) handleBootstrapMaster(rw http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		http.Error(rw, "método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var input ports.BootstrapMasterInput
	if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	var sshExec ports.SSHExecutor
	var config domain.ServerConfig
	serverTarget := req.URL.Query().Get("server")
	if serverTarget == "" && input.TargetNode != "" {
		serverTarget = input.TargetNode
	}
	cfg := w.resolveTargetServer(serverTarget)
	if cfg == nil || (cfg.Host == "" && !isLocalConfig(cfg)) {
		http.Error(rw, "VPS no configurado", http.StatusBadRequest)
		return
	}
	config = *cfg
	se := repositories.NewCryptoSSHExecutor()
	if err := se.Connect(config); err != nil {
		slog.Warn("web_server: falló conexión SSH en handleBootstrapMaster", "host", config.Host, "error", err)
		http.Error(rw, fmt.Sprintf("Error SSH con VPS: %v", err), http.StatusInternalServerError)
		return
	}
	sshExec = se
	defer func() {
		if clErr := se.Close(); clErr != nil {
			slog.Warn("web_server: error cerrando SSH en handleBootstrapMaster", "error", clErr)
		}
	}()

	linkUC := usecases.NewLinkServicesUseCase(w.repo, sshExec)
	unlinkUC := usecases.NewUnlinkServicesUseCase(w.repo, sshExec)
	dbUC := usecases.NewDeployDatabaseUseCase(sshExec)
	svcUC := usecases.NewDeployServiceUseCase(sshExec)

	bootstrapUC := usecases.NewBootstrapMasterServiceUseCase(w.repo, sshExec, linkUC, unlinkUC, dbUC, svcUC)
	result, err := bootstrapUC.Execute(input, config)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(rw, result)
}

func (w *WebServer) handlePreviewEnvs(rw http.ResponseWriter, req *http.Request) {
	var sshExec ports.SSHExecutor
	var config domain.ServerConfig
	if w.config != nil && w.config.Host != "" {
		config = *w.config
		se := repositories.NewCryptoSSHExecutor()
		if err := se.Connect(config); err == nil {
			sshExec = se
			defer se.Close()
		}
	}

	uc := usecases.NewManagePreviewEnvUseCase(w.repo, sshExec)

	if req.Method == "GET" {
		list, err := uc.List()
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(rw, list)
		return
	}

	if req.Method == "POST" {
		var input ports.CreatePreviewEnvInput
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		res, err := uc.Create(input, config)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(rw, res)
		return
	}

	if req.Method == "DELETE" {
		name := req.URL.Query().Get("name")
		if name == "" {
			http.Error(rw, "parámetro 'name' es requerido", http.StatusBadRequest)
			return
		}

		if err := uc.Destroy(name, config); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(rw, map[string]string{"message": fmt.Sprintf("entorno preview '%s' destruido exitosamente", name)})
		return
	}

	http.Error(rw, "método no permitido", http.StatusMethodNotAllowed)
}

func (w *WebServer) handleRegistries(rw http.ResponseWriter, req *http.Request) {
	var sshExec ports.SSHExecutor
	var cfg domain.ServerConfig
	if w.config != nil && w.config.Host != "" {
		cfg = *w.config
		se := repositories.NewCryptoSSHExecutor()
		if err := se.Connect(cfg); err == nil {
			sshExec = se
			defer se.Close()
		}
	}

	uc := usecases.NewManageRegistryAuthUseCase(w.repo, sshExec)

	switch req.Method {
	case http.MethodGet:
		creds, err := uc.List()
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		safeList := make([]domain.SavedRegistryCredential, len(creds))
		for i, c := range creds {
			safeList[i] = c
			if len(safeList[i].Password) > 4 {
				safeList[i].Password = "••••••••"
			}
		}
		jsonResponse(rw, safeList)

	case http.MethodPost:
		var cred domain.SavedRegistryCredential
		if err := json.NewDecoder(req.Body).Decode(&cred); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		if err := uc.Save(cred, cfg); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		jsonResponse(rw, map[string]string{"status": "saved", "server": cred.Server})

	case http.MethodDelete:
		server := req.URL.Query().Get("server")
		if server == "" {
			var body struct {
				Server string `json:"server"`
			}
			json.NewDecoder(req.Body).Decode(&body)
			server = body.Server
		}
		if err := uc.Delete(server, cfg); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		jsonResponse(rw, map[string]string{"status": "deleted", "server": server})

	default:
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (w *WebServer) handleMigrations(rw http.ResponseWriter, req *http.Request) {
	dbName := req.URL.Query().Get("db")
	cfg := w.resolveTargetServer(req.URL.Query().Get("server"))
	serverName := ""
	if cfg != nil {
		serverName = cfg.Name
	}
	files, err := w.repo.GetMigrationFiles(dbName, serverName)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(rw, files)
}

func (w *WebServer) handleMigrationFile(rw http.ResponseWriter, req *http.Request) {
	cfg := w.resolveTargetServer(req.URL.Query().Get("server"))
	serverName := ""
	if cfg != nil {
		serverName = cfg.Name
	}
	switch req.Method {
	case http.MethodPost:
		var body struct {
			DBName      string `json:"dbName"`
			Filename    string `json:"filename"`
			Content     string `json:"content"`
			DownContent string `json:"downContent"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		mf := domain.MigrationFile{
			DBName:      body.DBName,
			Filename:    body.Filename,
			Content:     body.Content,
			DownContent: body.DownContent,
			Status:      "pending",
			ServerName:  serverName,
		}
		if err := w.repo.SaveMigrationFile(mf); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(rw, map[string]string{"status": "saved", "filename": body.Filename})

	case http.MethodDelete:
		dbName := req.URL.Query().Get("db")
		filename := req.URL.Query().Get("filename")
		if err := w.repo.DeleteMigrationFile(dbName, filename, serverName); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(rw, map[string]string{"status": "deleted", "filename": filename})

	default:
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (w *WebServer) handleRunMigrations(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var migrReq domain.DatabaseMigrationRequest
	if err := json.NewDecoder(req.Body).Decode(&migrReq); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	serverTarget := req.URL.Query().Get("server")
	targetCfg := w.resolveTargetServer(serverTarget)
	cfg := domain.ServerConfig{}
	if targetCfg != nil {
		cfg = *targetCfg
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageDBMigrationsUseCase(w.repo, sshExec)
	res, err := uc.Execute(migrReq, cfg)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(rw, res)
}

func (w *WebServer) handleObservabilityMetrics(rw http.ResponseWriter, req *http.Request) {
	service := req.URL.Query().Get("service")
	timeRange := req.URL.Query().Get("range")
	if service == "" {
		service = "all"
	}

	serverTarget := req.URL.Query().Get("server")
	targetCfg := w.resolveTargetServer(serverTarget)
	cfg := domain.ServerConfig{}
	if targetCfg != nil {
		cfg = *targetCfg
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewGetServiceMetricsUseCase(w.repo, sshExec)
	metrics, err := uc.Execute(service, timeRange, cfg)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(rw, metrics)
}

func (w *WebServer) handleBackups(rw http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		cfg := w.resolveTargetServer(req.URL.Query().Get("server"))
		serverName := ""
		if cfg != nil {
			serverName = cfg.Name
		}
		backups, err := w.repo.GetBackups(serverName)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		if backups == nil {
			backups = []domain.SavedBackup{}
		}
		targetFilter := strings.TrimSpace(req.URL.Query().Get("targetName"))
		if targetFilter != "" {
			var filtered []domain.SavedBackup
			for _, b := range backups {
				if strings.EqualFold(b.TargetName, targetFilter) {
					filtered = append(filtered, b)
				}
			}
			if filtered == nil {
				filtered = []domain.SavedBackup{}
			}
			jsonResponse(rw, filtered)
			return
		}
		jsonResponse(rw, backups)
		return
	}

	if req.Method == http.MethodPost {
		var bReq domain.BackupRequest
		if err := json.NewDecoder(req.Body).Decode(&bReq); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		if bReq.TargetType == "" {
			bReq.TargetType = "database"
		}
		if bReq.TargetName == "" {
			bReq.TargetName = strings.TrimSpace(req.URL.Query().Get("name"))
		}
		if bReq.TargetName == "" {
			http.Error(rw, "Parámetro 'targetName' o 'name' requerido", http.StatusBadRequest)
			return
		}
		serverTarget := req.URL.Query().Get("server")
		if serverTarget == "" && bReq.Server != "" {
			serverTarget = bReq.Server
		}
		cfg := w.resolveTargetServer(serverTarget)
		if cfg == nil || (cfg.Host == "" && !isLocalConfig(cfg)) {
			http.Error(rw, "VPS no configurado", http.StatusBadRequest)
			return
		}
		sshExec := repositories.NewCryptoSSHExecutor()
		uc := usecases.NewManageBackupsUseCase(w.repo, sshExec)
		backup, err := uc.CreateSnapshot(bReq, *cfg)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(rw, backup)
		return
	}

	if req.Method == http.MethodDelete {
		idStr := req.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(rw, "ID inválido", http.StatusBadRequest)
			return
		}
		cfg := w.resolveTargetServer(req.URL.Query().Get("server"))
		serverName := ""
		if cfg != nil {
			serverName = cfg.Name
		}
		if err := w.repo.DeleteBackup(id, serverName); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(rw, map[string]string{"status": "deleted"})
		return
	}
}

func (w *WebServer) handleRestoreBackup(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var bReq domain.BackupRequest
	if err := json.NewDecoder(req.Body).Decode(&bReq); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	serverTarget := req.URL.Query().Get("server")
	if serverTarget == "" && bReq.Server != "" {
		serverTarget = bReq.Server
	}
	cfg := w.resolveTargetServer(serverTarget)
	if cfg == nil || (cfg.Host == "" && !isLocalConfig(cfg)) {
		http.Error(rw, "VPS no configurado", http.StatusBadRequest)
		return
	}
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageBackupsUseCase(w.repo, sshExec)
	if err := uc.RestoreSnapshot(bReq.BackupID, *cfg); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(rw, map[string]string{"status": "restored"})
}

func (w *WebServer) handleDownloadBackup(rw http.ResponseWriter, req *http.Request) {
	idStr := req.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(rw, "ID de backup inválido", http.StatusBadRequest)
		return
	}
	serverTarget := req.URL.Query().Get("server")
	cfg := w.resolveTargetServer(serverTarget)
	if cfg == nil || (cfg.Host == "" && !isLocalConfig(cfg)) {
		http.Error(rw, "VPS no configurado", http.StatusBadRequest)
		return
	}
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageBackupsUseCase(w.repo, sshExec)
	res, err := uc.DownloadSnapshot(id, *cfg)
	if err != nil || res == nil {
		msg := "error al descargar backup"
		if err != nil {
			msg = err.Error()
		}
		http.Error(rw, msg, http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", res.Filename))
	rw.Header().Set("Content-Type", "application/octet-stream")
	rw.Header().Set("Content-Length", strconv.Itoa(len(res.Data)))
	if _, errWrite := rw.Write(res.Data); errWrite != nil {
		slog.Warn("falló escritura de backup en http response", "error", errWrite)
	}
}

func (w *WebServer) handleEnvVars(rw http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		serviceName := req.URL.Query().Get("service")
		if serviceName == "" {
			http.Error(rw, "Parámetro 'service' es requerido", http.StatusBadRequest)
			return
		}
		cfg := w.resolveTargetServer(req.URL.Query().Get("server"))
		serverName := ""
		if cfg != nil {
			serverName = cfg.Name
		}
		sshExec := repositories.NewCryptoSSHExecutor()
		uc := usecases.NewManageEnvVarsUseCase(w.repo, sshExec)
		envData, err := uc.GetEnvVars(serviceName, serverName)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusNotFound)
			return
		}
		jsonResponse(rw, map[string]interface{}{
			"serviceName": serviceName,
			"rawContent":  envData.Raw,
			"envVars":     envData.Map,
		})
		return
	}

	if req.Method == http.MethodPost {
		var body struct {
			ServiceName string `json:"serviceName"`
			RawContent  string `json:"rawContent"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		if body.ServiceName == "" {
			http.Error(rw, "serviceName es requerido", http.StatusBadRequest)
			return
		}
		cfg := w.getConfig()
		if cfg == nil || cfg.Host == "" {
			if loaded, err := w.repo.GetServerConfig(); err == nil && loaded != nil && loaded.Host != "" {
				w.setConfig(loaded)
				cfg = loaded
			}
		}
		serverCfg := domain.ServerConfig{}
		if cfg != nil {
			serverCfg = *cfg
		}
		sshExec := repositories.NewCryptoSSHExecutor()
		uc := usecases.NewManageEnvVarsUseCase(w.repo, sshExec)
		if err := uc.UpdateEnvVars(body.ServiceName, body.RawContent, serverCfg); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(rw, map[string]string{"status": "updated"})
		return
	}
}

func (w *WebServer) handleExportEnvVars(rw http.ResponseWriter, req *http.Request) {
	serviceName := req.URL.Query().Get("service")
	if serviceName == "" {
		http.Error(rw, "Parámetro 'service' es requerido", http.StatusBadRequest)
		return
	}
	cfg := w.resolveTargetServer(req.URL.Query().Get("server"))
	serverName := ""
	if cfg != nil {
		serverName = cfg.Name
	}
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageEnvVarsUseCase(w.repo, sshExec)
	envData, err := uc.GetEnvVars(serviceName, serverName)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}

	filename := fmt.Sprintf("%s.env", serviceName)
	rw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.Header().Set("Content-Length", strconv.Itoa(len(envData.Raw)))
	rw.Write([]byte(envData.Raw))
}

func (w *WebServer) resolveVolumeConfig(req *http.Request) domain.ServerConfig {
	cfg, err := w.getTargetServerConfig(req)
	if err == nil && cfg != nil {
		return *cfg
	}
	curr := w.getConfig()
	if curr != nil {
		return *curr
	}
	return domain.ServerConfig{}
}

func (w *WebServer) handleVolumes(rw http.ResponseWriter, req *http.Request) {
	cfg := w.resolveVolumeConfig(req)
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageVolumesUseCase(w.repo, sshExec)
	vols, err := uc.ListVolumes(cfg)
	if err != nil {
		jsonResponse(rw, []string{})
		return
	}
	jsonResponse(rw, vols)
}

func (w *WebServer) handleVolumeFiles(rw http.ResponseWriter, req *http.Request) {
	path := req.URL.Query().Get("path")
	cfg := w.resolveVolumeConfig(req)
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageVolumesUseCase(w.repo, sshExec)
	files, err := uc.ListVolumeFiles(path, cfg)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	jsonResponse(rw, files)
}

func (w *WebServer) handleVolumeRead(rw http.ResponseWriter, req *http.Request) {
	path := req.URL.Query().Get("path")
	if path == "" {
		http.Error(rw, "path es requerido", http.StatusBadRequest)
		return
	}
	cfg := w.resolveVolumeConfig(req)
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageVolumesUseCase(w.repo, sshExec)
	content, err := uc.ReadFileContent(path, cfg)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(rw, map[string]string{"path": path, "content": content})
}

func (w *WebServer) handleVolumeWrite(rw http.ResponseWriter, req *http.Request) {
	var body struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	if body.Path == "" {
		http.Error(rw, "path es requerido", http.StatusBadRequest)
		return
	}
	cfg := w.resolveVolumeConfig(req)
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageVolumesUseCase(w.repo, sshExec)
	if err := uc.WriteFileContent(body.Path, body.Content, cfg); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(rw, map[string]string{"status": "saved"})
}

func (w *WebServer) handleVolumeDownload(rw http.ResponseWriter, req *http.Request) {
	path := req.URL.Query().Get("path")
	if path == "" {
		http.Error(rw, "path es requerido", http.StatusBadRequest)
		return
	}
	cfg := w.resolveVolumeConfig(req)
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageVolumesUseCase(w.repo, sshExec)
	res, err := uc.DownloadFile(path, cfg)
	if err != nil || res == nil {
		msg := "error al descargar archivo"
		if err != nil {
			msg = err.Error()
		}
		http.Error(rw, msg, http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", res.Filename))
	rw.Header().Set("Content-Type", "application/octet-stream")
	rw.Header().Set("Content-Length", strconv.Itoa(len(res.Data)))
	if _, errWrite := rw.Write(res.Data); errWrite != nil {
		slog.Error("error escribiendo descarga de archivo", "error", errWrite)
	}
}

func (w *WebServer) handleVolumeUpload(rw http.ResponseWriter, req *http.Request) {
	err := req.ParseMultipartForm(100 << 20) // 100MB max
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := req.FormFile("file")
	if err != nil {
		http.Error(rw, "archivo no recibido", http.StatusBadRequest)
		return
	}
	defer file.Close()

	targetPath := req.FormValue("targetPath")
	targetFile := targetPath
	if targetFile == "" {
		dirPath := req.FormValue("dir")
		if dirPath == "" {
			dirPath = "/opt/data"
		}
		targetFile = fmt.Sprintf("%s/%s", strings.TrimRight(dirPath, "/"), header.Filename)
	}

	buf, errRead := io.ReadAll(file)
	if errRead != nil {
		http.Error(rw, "error al leer archivo: "+errRead.Error(), http.StatusBadRequest)
		return
	}

	cfg := w.resolveVolumeConfig(req)
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageVolumesUseCase(w.repo, sshExec)

	if err := uc.WriteFileContent(targetFile, string(buf), cfg); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(rw, map[string]string{"status": "uploaded", "target": targetFile})
}

func (w *WebServer) handleVolumeDelete(rw http.ResponseWriter, req *http.Request) {
	path := req.URL.Query().Get("path")
	if path == "" {
		http.Error(rw, "path es requerido", http.StatusBadRequest)
		return
	}
	cfg := w.resolveVolumeConfig(req)
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageVolumesUseCase(w.repo, sshExec)
	if err := uc.DeleteFile(path, cfg); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(rw, map[string]string{"status": "deleted"})
}

func (w *WebServer) handleVolumeMkdir(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	if body.Path == "" {
		http.Error(rw, "path es requerido", http.StatusBadRequest)
		return
	}
	cfg := w.resolveVolumeConfig(req)
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageVolumesUseCase(w.repo, sshExec)
	if err := uc.CreateDirectory(body.Path, cfg); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(rw, map[string]string{"status": "created", "path": body.Path})
}


func (w *WebServer) handleSSLInspect(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	cfg := w.resolveTargetServer(req.URL.Query().Get("server"))
	serverName := ""
	if cfg != nil {
		serverName = cfg.Name
	}
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageSSLMaintenanceUseCase(w.repo, sshExec)
	items, err := uc.InspectSSL(serverName)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(rw, items)
}

func (w *WebServer) handleMaintenanceToggle(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ServiceName string `json:"serviceName"`
		Enable      bool   `json:"enable"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	if body.ServiceName == "" {
		http.Error(rw, "serviceName es requerido", http.StatusBadRequest)
		return
	}

	cfg := w.resolveVolumeConfig(req)
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageSSLMaintenanceUseCase(w.repo, sshExec)
	if err := uc.ToggleMaintenanceMode(body.ServiceName, body.Enable, cfg); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	statusMsg := "disabled"
	if body.Enable {
		statusMsg = "enabled"
	}
	jsonResponse(rw, map[string]interface{}{
		"status":      "success",
		"mode":        statusMsg,
		"serviceName": body.ServiceName,
	})
}

func (w *WebServer) handleSSLReload(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	cfg := w.resolveVolumeConfig(req)
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageSSLMaintenanceUseCase(w.repo, sshExec)
	if err := uc.ReloadTraefik(cfg); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(rw, map[string]string{
		"status":  "success",
		"message": "Traefik recargado correctamente",
	})
}

func (w *WebServer) handleCustomDomains(rw http.ResponseWriter, req *http.Request) {
	cfg := domain.ServerConfig{}
	if w.config != nil {
		cfg = *w.config
	}
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageDomainsUseCase(w.repo, sshExec)

	switch req.Method {
	case http.MethodGet:
		serviceName := req.URL.Query().Get("service")
		if serviceName == "" {
			http.Error(rw, "service es requerido", http.StatusBadRequest)
			return
		}
		info, err := uc.GetServiceDomains(serviceName, cfg.Name)
		if err != nil || info == nil {
			msg := "servicio no encontrado"
			if err != nil {
				msg = err.Error()
			}
			http.Error(rw, msg, http.StatusNotFound)
			return
		}
		jsonResponse(rw, map[string]interface{}{
			"primaryDomain": info.PrimaryDomain,
			"customRules":   info.Rules,
		})

	case http.MethodPost:
		var body struct {
			ServiceName    string `json:"serviceName"`
			Domain         string `json:"domain"`
			RedirectTarget string `json:"redirectTarget"`
			CertType       string `json:"certType"`
			ForceHTTPS     bool   `json:"forceHTTPS"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		if err := uc.AddCustomDomain(body.ServiceName, body.Domain, body.RedirectTarget, cfg); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(rw, map[string]string{"status": "added", "certType": body.CertType})

	case http.MethodDelete:
		serviceName := req.URL.Query().Get("service")
		customDomain := req.URL.Query().Get("domain")
		if serviceName == "" || customDomain == "" {
			http.Error(rw, "service y domain son requeridos", http.StatusBadRequest)
			return
		}
		if err := uc.RemoveCustomDomain(serviceName, customDomain, cfg); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(rw, map[string]string{"status": "removed"})

	default:
		http.Error(rw, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func (w *WebServer) handleDNSCheck(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	rawDomain := strings.TrimSpace(req.URL.Query().Get("domain"))
	if rawDomain == "" {
		jsonError(rw, "Parámetro 'domain' requerido", http.StatusBadRequest)
		return
	}

	serverIP := ""
	cfg := w.getConfig()
	if cfg == nil || cfg.Host == "" {
		if loaded, err := w.repo.GetServerConfig(); err == nil && loaded != nil && loaded.Host != "" {
			w.setConfig(loaded)
			cfg = loaded
		}
	}
	if cfg != nil {
		serverIP = cfg.Host
	}

	uc := usecases.NewCheckDomainDNSUseCase(repositories.NewNetDNSResolver())
	result, err := uc.Execute(rawDomain, serverIP)
	if err != nil {
		jsonError(rw, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(rw, result)
}

// --- Audit Logs Handler ---
func (w *WebServer) handleAuditLogs(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	logs, err := w.repo.GetAuditLogs(100)
	if err != nil {
		jsonError(rw, fmt.Sprintf("Error leyendo logs de auditoría: %v", err), http.StatusInternalServerError)
		return
	}
	if logs == nil {
		logs = []domain.AuditLog{}
	}
	jsonResponse(rw, logs)
}

// --- Container Live Stats Handler (cgroups / docker stats) ---
func (w *WebServer) handleContainerStats(rw http.ResponseWriter, req *http.Request) {
	name := req.URL.Query().Get("name")
	if name == "" || !isValidNodeID(name) {
		jsonError(rw, "Parámetro 'name' de contenedor/servicio requerido", http.StatusBadRequest)
		return
	}

	serverTarget := req.URL.Query().Get("server")
	cfg := w.resolveTargetServer(serverTarget)
	if cfg == nil || cfg.Host == "" {
		jsonError(rw, "VPS no configurado", http.StatusBadRequest)
		return
	}

	cacheKey := fmt.Sprintf("stats:%s:%s", cfg.Host, name)
	if cached, ok := w.getCache(cacheKey); ok {
		if stats, ok := cached.(domain.ContainerStats); ok {
			jsonResponse(rw, stats)
			return
		}
	}

	uc := usecases.NewGetContainerStatsUseCase(repositories.NewCryptoSSHExecutor())
	stats, err := uc.Execute(name, *cfg)
	if err != nil {
		jsonError(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	w.setCache(cacheKey, stats, 5*time.Second)
	jsonResponse(rw, stats)
}

// --- DB Health Inspection Handler ---
func (w *WebServer) handleDBHealth(rw http.ResponseWriter, req *http.Request) {
	name := req.URL.Query().Get("name")
	if name == "" || !isValidNodeID(name) {
		jsonError(rw, "Parámetro 'name' de base de datos requerido", http.StatusBadRequest)
		return
	}

	serverTarget := req.URL.Query().Get("server")
	cfg := w.resolveTargetServer(serverTarget)
	if cfg == nil || cfg.Host == "" {
		jsonError(rw, "VPS no configurado", http.StatusBadRequest)
		return
	}

	cleanName := strings.TrimPrefix(name, "tarhiata-db-")
	cleanName = strings.TrimPrefix(cleanName, "tarhiata-")

	cacheKey := fmt.Sprintf("dbhealth:%s:%s", cfg.Host, cleanName)
	if cached, ok := w.getCache(cacheKey); ok {
		if health, ok := cached.(domain.DBHealthStats); ok {
			jsonResponse(rw, health)
			return
		}
	}

	uc := usecases.NewGetDBHealthUseCase(w.repo, repositories.NewCryptoSSHExecutor())
	health, err := uc.Execute(name, *cfg)
	if err != nil {
		jsonError(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	w.setCache(cacheKey, health, 5*time.Second)
	jsonResponse(rw, health)
}

// --- Vultr API Plans & Regions Handlers ---
func (w *WebServer) handleVultrPlans(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	cfg := w.getConfig()
	token := ""
	if cfg != nil {
		token = cfg.VultrAPIToken
	}

	cacheKey := fmt.Sprintf("vultr:plans:%s", token)
	if cached, ok := w.getCache(cacheKey); ok {
		if plans, ok := cached.([]domain.VultrPlan); ok {
			jsonResponse(rw, plans)
			return
		}
	}

	uc := usecases.NewListVultrPlansUseCase()
	plans, err := uc.ExecutePlans(token)
	if err != nil {
		jsonError(rw, fmt.Sprintf("Error obteniendo planes de Vultr: %v", err), http.StatusInternalServerError)
		return
	}
	w.setCache(cacheKey, plans, 5*time.Minute)
	jsonResponse(rw, plans)
}

func (w *WebServer) handleVultrRegions(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	cfg := w.getConfig()
	token := ""
	if cfg != nil {
		token = cfg.VultrAPIToken
	}

	cacheKey := fmt.Sprintf("vultr:regions:%s", token)
	if cached, ok := w.getCache(cacheKey); ok {
		if regions, ok := cached.([]domain.VultrRegion); ok {
			jsonResponse(rw, regions)
			return
		}
	}

	uc := usecases.NewListVultrPlansUseCase()
	regions, err := uc.ExecuteRegions(token)
	if err != nil {
		jsonError(rw, fmt.Sprintf("Error obteniendo regiones de Vultr: %v", err), http.StatusInternalServerError)
		return
	}
	w.setCache(cacheKey, regions, 5*time.Minute)
	jsonResponse(rw, regions)
}

// --- Remote VPS State Auto-Sync Handler (Multi-PC Recovery) ---
func (w *WebServer) handleSyncState(rw http.ResponseWriter, req *http.Request) {
	cfg := w.getConfig()
	if cfg == nil || cfg.Host == "" {
		jsonError(rw, "VPS Master no configurado", http.StatusBadRequest)
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*cfg); err != nil {
		jsonError(rw, fmt.Sprintf("Error conectando SSH: %v", err), http.StatusInternalServerError)
		return
	}
	defer sshExec.Close()

	syncUC := usecases.NewSyncClusterStateUseCase(w.repo, sshExec)

	switch req.Method {
	case http.MethodGet, http.MethodPost:
		// Primero intenta importar del VPS
		dump, err := syncUC.ImportStateFromRemote(cfg.Name)
		if err != nil {
			// Si no existe state.json en VPS, exporta el estado local actual
			if exportErr := syncUC.ExportStateToRemote(cfg.Name); exportErr != nil {
				slog.Warn("falló exportar estado a VPS remoto", "error", exportErr)
			}
		}
		jsonResponse(rw, map[string]interface{}{
			"status": "synced",
			"dump":   dump,
		})
	default:
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func (w *WebServer) syncStateToRemote(cfg *domain.ServerConfig) {
	if cfg == nil || cfg.Host == "" {
		return
	}
	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*cfg); err == nil {
		defer sshExec.Close()
		syncUC := usecases.NewSyncClusterStateUseCase(w.repo, sshExec)
		if err := syncUC.ExportStateToRemote(cfg.Name); err != nil {
			slog.Warn("falló sincronización de estado a remoto", "error", err)
		}
	}
}

// --- SSH Keys Management Handler (Team Member Access) ---
func (w *WebServer) handleSSHKeys(rw http.ResponseWriter, req *http.Request) {
	cfg := w.getConfig()
	if cfg == nil || cfg.Host == "" {
		jsonError(rw, "Servidor VPS no configurado", http.StatusBadRequest)
		return
	}

	uc := usecases.NewManageSSHKeysUseCase(nil)

	switch req.Method {
	case http.MethodGet:
		keys, err := uc.ListKeys(*cfg)
		if err != nil {
			jsonError(rw, fmt.Sprintf("Error listando llaves SSH: %v", err), http.StatusInternalServerError)
			return
		}
		jsonResponse(rw, keys)

	case http.MethodPost:
		var body struct {
			PublicKey string `json:"publicKey"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			jsonError(rw, "Cuerpo JSON inválido", http.StatusBadRequest)
			return
		}
		if err := uc.AddKey(*cfg, body.PublicKey); err != nil {
			jsonError(rw, err.Error(), http.StatusBadRequest)
			return
		}
		jsonResponse(rw, map[string]string{"status": "added"})

	case http.MethodDelete:
		fp := req.URL.Query().Get("fp")
		if fp == "" {
			var body struct {
				Fingerprint string `json:"fingerprint"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil && err != io.EOF {
				slog.Debug("aviso al decodificar fingerprint de body", "error", err)
			}
			fp = body.Fingerprint
		}
		if fp == "" {
			jsonError(rw, "Se requiere el parámetro 'fp' (fingerprint) de la llave a eliminar", http.StatusBadRequest)
			return
		}
		if err := uc.DeleteKey(*cfg, fp); err != nil {
			jsonError(rw, err.Error(), http.StatusBadRequest)
			return
		}
		jsonResponse(rw, map[string]string{"status": "deleted", "fingerprint": fp})

	default:
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

// --- Outbound Alerts Handlers ---

func (w *WebServer) handleAlertSettings(rw http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		settings, err := w.repo.GetAlertSettings()
		if err != nil {
			jsonError(rw, fmt.Sprintf("Error obteniendo configuración de alertas: %v", err), http.StatusInternalServerError)
			return
		}
		if settings == nil {
			settings = &domain.AlertSettings{Enabled: false}
		}
		jsonResponse(rw, settings)

	case http.MethodPost:
		var s domain.AlertSettings
		if err := json.NewDecoder(req.Body).Decode(&s); err != nil {
			jsonError(rw, "Cuerpo JSON inválido", http.StatusBadRequest)
			return
		}
		if err := w.repo.SaveAlertSettings(s); err != nil {
			jsonError(rw, fmt.Sprintf("Error guardando alertas: %v", err), http.StatusInternalServerError)
			return
		}
		jsonResponse(rw, map[string]interface{}{"status": "saved", "settings": s})

	default:
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func (w *WebServer) handleAlertTest(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var testReq struct {
		Settings *domain.AlertSettings `json:"settings,omitempty"`
	}
	if err := json.NewDecoder(req.Body).Decode(&testReq); err != nil && err != io.EOF {
		slog.Debug("aviso decodificando test alert body", "error", err)
	}

	settings := testReq.Settings
	if settings == nil {
		var err error
		settings, err = w.repo.GetAlertSettings()
		if err != nil {
			jsonError(rw, fmt.Sprintf("Error leyendo configuración de alertas: %v", err), http.StatusInternalServerError)
			return
		}
	}
	if settings == nil || !settings.Enabled {
		settings = &domain.AlertSettings{Enabled: true}
	}

	sender := notify.NewSender(5 * time.Second)
	ev := notify.AlertEvent{
		Title:       "Alerta de Prueba — Tarhiata Cloud Studio",
		Description: "Conectividad exitosa con el canal de notificaciones configurado.",
		Severity:    notify.SeveritySuccess,
		ServerName:  "tarhiata-control-plane",
		Fields:      map[string]string{"Estado": "Operativo", "Timestamp": time.Now().Format(time.RFC3339)},
		Timestamp:   time.Now(),
	}

	cfg := notify.WebhookConfig{
		DiscordURL:    settings.DiscordURL,
		TelegramToken: settings.TelegramToken,
		TelegramChat:  settings.TelegramChat,
		SlackURL:      settings.SlackURL,
		GenericURL:    settings.GenericURL,
		Enabled:       true,
	}

	errs := sender.Dispatch(req.Context(), cfg, ev)
	if len(errs) > 0 {
		var errMsgs []string
		for _, e := range errs {
			errMsgs = append(errMsgs, e.Error())
		}
		jsonResponse(rw, map[string]interface{}{
			"status":   "partial_failure",
			"errors":   errMsgs,
			"message":  "Se detectaron fallos al despachar la alerta de prueba",
		})
		return
	}

	jsonResponse(rw, map[string]string{"status": "dispatched", "message": "Alerta de prueba enviada exitosamente"})
}

// --- Deployment History & Version Rollback Handlers ---

func (w *WebServer) handleServiceHistory(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	serviceName := req.URL.Query().Get("name")
	if strings.TrimSpace(serviceName) == "" {
		jsonError(rw, "Parámetro 'name' de servicio requerido", http.StatusBadRequest)
		return
	}
	history, err := w.repo.GetDeploymentHistory(serviceName, 30)
	if err != nil {
		jsonError(rw, fmt.Sprintf("Error obteniendo historial: %v", err), http.StatusInternalServerError)
		return
	}
	if history == nil {
		history = []domain.DeploymentRecord{}
	}
	jsonResponse(rw, history)
}

func (w *WebServer) handleServiceRollbackToVersion(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		ServiceName string `json:"serviceName"`
		VersionID   int    `json:"versionId"`
		Server      string `json:"server,omitempty"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		jsonError(rw, "Cuerpo JSON inválido", http.StatusBadRequest)
		return
	}
	if body.ServiceName == "" || body.VersionID <= 0 {
		jsonError(rw, "Parámetros 'serviceName' y 'versionId' válidos requeridos", http.StatusBadRequest)
		return
	}

	cfg := w.resolveTargetServer(body.Server)
	if cfg == nil || (cfg.Host == "" && !isLocalConfig(cfg)) {
		jsonError(rw, "Servidor VPS no configurado", http.StatusBadRequest)
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("web_server: error cerrando sshExec en rollback de versión", "error", clErr)
		}
	}()

	uc := usecases.NewManageDeploymentHistoryUseCase(w.repo, sshExec)
	rec, err := uc.RollbackToVersion(body.ServiceName, body.VersionID, *cfg)
	if err != nil {
		jsonError(rw, fmt.Sprintf("Error revirtiendo a versión %d: %v", body.VersionID, err), http.StatusInternalServerError)
		return
	}

	_ = w.repo.SaveAuditLog(domain.AuditLog{
		Action:       "ROLLBACK",
		ResourceType: "service",
		ResourceName: body.ServiceName,
		Details:      fmt.Sprintf("Revertido a versión ID %d (imagen %s)", body.VersionID, rec.ImageTag),
		Timestamp:    time.Now(),
	})

	jsonResponse(rw, map[string]interface{}{
		"status":  "rolled_back",
		"record":  rec,
		"message": fmt.Sprintf("Servicio revertido a la versión %d (%s)", body.VersionID, rec.ImageTag),
	})
}

// --- ACME SSL Certificates Handler ---

func (w *WebServer) handleSSLCertificates(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	cfg, err := w.getTargetServerConfig(req)
	if err != nil {
		jsonError(rw, err.Error(), http.StatusNotFound)
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("web_server: error cerrando sshExec en ssl certs", "error", clErr)
		}
	}()

	uc := usecases.NewInspectSSLAcmeUseCase(sshExec)
	certs, err := uc.Execute(*cfg)
	if err != nil {
		jsonError(rw, fmt.Sprintf("Error inspeccionando certificados ACME: %v", err), http.StatusInternalServerError)
		return
	}
	if certs == nil {
		certs = []usecases.ACMECertificateSummary{}
	}
	jsonResponse(rw, certs)
}

// --- Node Drain & Maintenance Handlers ---

func (w *WebServer) handleNodeDrain(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		NodeID string `json:"nodeId"`
		Server string `json:"server,omitempty"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil || strings.TrimSpace(body.NodeID) == "" {
		jsonError(rw, "Parámetro 'nodeId' requerido", http.StatusBadRequest)
		return
	}

	cfg := w.resolveTargetServer(body.Server)
	if cfg == nil || (cfg.Host == "" && !isLocalConfig(cfg)) {
		jsonError(rw, "Servidor VPS no configurado", http.StatusBadRequest)
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("web_server: error cerrando sshExec en node drain", "error", clErr)
		}
	}()

	uc := usecases.NewDrainNodeUseCase(sshExec)
	res, err := uc.Execute(body.NodeID, "drain", *cfg)
	if err != nil {
		jsonError(rw, fmt.Sprintf("Error drenando nodo: %v", err), http.StatusInternalServerError)
		return
	}

	_ = w.repo.SaveAuditLog(domain.AuditLog{
		Action:       "DRAIN",
		ResourceType: "node",
		ResourceName: body.NodeID,
		Details:      res.Message,
		Timestamp:    time.Now(),
	})

	jsonResponse(rw, res)
}

func (w *WebServer) handleNodeActivate(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		NodeID string `json:"nodeId"`
		Server string `json:"server,omitempty"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil || strings.TrimSpace(body.NodeID) == "" {
		jsonError(rw, "Parámetro 'nodeId' requerido", http.StatusBadRequest)
		return
	}

	cfg := w.resolveTargetServer(body.Server)
	if cfg == nil || (cfg.Host == "" && !isLocalConfig(cfg)) {
		jsonError(rw, "Servidor VPS no configurado", http.StatusBadRequest)
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("web_server: error cerrando sshExec en node activate", "error", clErr)
		}
	}()

	uc := usecases.NewDrainNodeUseCase(sshExec)
	res, err := uc.Execute(body.NodeID, "active", *cfg)
	if err != nil {
		jsonError(rw, fmt.Sprintf("Error reactivando nodo: %v", err), http.StatusInternalServerError)
		return
	}

	_ = w.repo.SaveAuditLog(domain.AuditLog{
		Action:       "EDIT",
		ResourceType: "node",
		ResourceName: body.NodeID,
		Details:      res.Message,
		Timestamp:    time.Now(),
	})

	jsonResponse(rw, res)
}

// --- Git Webhook Auto-Deploy Handler ---

func (w *WebServer) handleWebhookDeploy(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	serviceName := req.URL.Query().Get("service")
	imageTag := req.URL.Query().Get("image")

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		jsonError(rw, "Error leyendo cuerpo de webhook", http.StatusBadRequest)
		return
	}

	if serviceName == "" {
		var genericPayload struct {
			Service string `json:"service"`
			Image   string `json:"image"`
		}
		if errJSON := json.Unmarshal(bodyBytes, &genericPayload); errJSON == nil {
			if genericPayload.Service != "" {
				serviceName = genericPayload.Service
			}
			if genericPayload.Image != "" && imageTag == "" {
				imageTag = genericPayload.Image
			}
		}
	}

	if strings.TrimSpace(serviceName) == "" {
		jsonError(rw, "Parámetro 'service' requerido en webhook", http.StatusBadRequest)
		return
	}

	// Validación de firma HMAC obligatoria contra el secreto guardado del lado del
	// servidor para este servicio (nunca contra un valor que venga en la propia request:
	// eso le permitiría a quien manda la petición controlar ambos lados de la comparación).
	webhookCfg, errCfg := w.getTargetServerConfig(req)
	if errCfg != nil {
		jsonError(rw, errCfg.Error(), http.StatusNotFound)
		return
	}
	svc, errSvc := w.repo.GetService(serviceName, webhookCfg.Name)
	if errSvc != nil {
		jsonError(rw, fmt.Sprintf("Error obteniendo servicio: %v", errSvc), http.StatusInternalServerError)
		return
	}
	if svc == nil || strings.TrimSpace(svc.WebhookSecret) == "" {
		jsonError(rw, "Este servicio no tiene un secreto de webhook configurado; configúralo antes de usar el auto-despliegue", http.StatusForbidden)
		return
	}
	sigHeader := req.Header.Get("X-Hub-Signature-256")
	if sigHeader == "" {
		jsonError(rw, "Falta el header X-Hub-Signature-256", http.StatusUnauthorized)
		return
	}
	if !usecases.VerifySignature(svc.WebhookSecret, bodyBytes, sigHeader) {
		jsonError(rw, "Firma de webhook inválida (HMAC SHA-256 mismatch)", http.StatusUnauthorized)
		return
	}

	// Reutilizamos webhookCfg (ya resuelto y validado arriba para la firma HMAC) en vez de
	// re-resolver contra el servidor activo global: un servicio puede vivir en un servidor
	// del fleet distinto al que esté marcado como activo en ese momento, y desplegar contra
	// el servidor equivocado sobrescribiría/fallaría el servicio homónimo de otra máquina.
	cfg := webhookCfg

	// Build-from-source: en vez de re-pullear un tag existente, disparamos el build en
	// background (GitHub/GitLab/Gitea no esperan streaming, solo una respuesta rápida)
	// y el dashboard se engancha a verlo vía /api/builds/stream.
	if svc.SourceType == "git" {
		commitSHA := usecases.ExtractCommitSHA(bodyBytes)
		job := w.builds.NewJob(serviceName)
		go func() {
			tag, errBuild := w.runBuildAndDeploy(*svc, commitSHA, *cfg, job.AppendLine)
			if errBuild != nil {
				job.Finish("failed", tag, errBuild.Error())
				return
			}
			_ = w.repo.SaveAuditLog(domain.AuditLog{
				Action:       "DEPLOY",
				ResourceType: "service",
				ResourceName: serviceName,
				Details:      fmt.Sprintf("Auto-despliegue por Git Webhook (build desde fuente, imagen: %s)", tag),
				Timestamp:    time.Now(),
			})
			job.Finish("success", tag, "")
		}()
		jsonResponse(rw, map[string]string{"status": "building", "buildId": job.ID})
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("web_server: error cerrando sshExec en webhook deploy", "error", clErr)
		}
	}()

	uc := usecases.NewTriggerWebhookDeployUseCase(w.repo, sshExec)
	rec, err := uc.Execute(serviceName, imageTag, *cfg)
	if err != nil {
		jsonError(rw, fmt.Sprintf("Error en auto-despliegue de webhook: %v", err), http.StatusInternalServerError)
		return
	}

	_ = w.repo.SaveAuditLog(domain.AuditLog{
		Action:       "DEPLOY",
		ResourceType: "service",
		ResourceName: serviceName,
		Details:      fmt.Sprintf("Auto-despliegue por Git Webhook (imagen: %s)", rec.ImageTag),
		Timestamp:    time.Now(),
	})

	// Enviar alerta si hay canales configurados
	if alertSettings, errAlert := w.repo.GetAlertSettings(); errAlert == nil && alertSettings != nil && alertSettings.Enabled {
		sender := notify.NewSender(5 * time.Second)
		_ = sender.Dispatch(req.Context(), notify.WebhookConfig{
			DiscordURL:    alertSettings.DiscordURL,
			TelegramToken: alertSettings.TelegramToken,
			TelegramChat:  alertSettings.TelegramChat,
			SlackURL:      alertSettings.SlackURL,
			GenericURL:    alertSettings.GenericURL,
			Enabled:       true,
		}, notify.AlertEvent{
			Title:       fmt.Sprintf("Auto-Despliegue: %s", serviceName),
			Description: fmt.Sprintf("Servicio actualizado automáticamente vía Git Webhook con imagen `%s`.", rec.ImageTag),
			Severity:    notify.SeveritySuccess,
			ServerName:  cfg.Name,
			Resource:    serviceName,
			Timestamp:   time.Now(),
		})
	}

	jsonResponse(rw, map[string]interface{}{
		"status":  "deployed",
		"record":  rec,
		"message": fmt.Sprintf("Servicio '%s' actualizado exitosamente a '%s'", serviceName, rec.ImageTag),
	})
}

// runBuildAndDeploy construye la imagen desde el repo del servicio y, si el build sale
// bien, redespliega el servicio con esa imagen (reusa TriggerWebhookDeployUseCase tal
// cual existe, sin duplicar lógica de despliegue). commitRef vacío = HEAD del branch.
func (w *WebServer) runBuildAndDeploy(svc domain.SavedService, commitRef string, config domain.ServerConfig, onLine func(string)) (string, error) {
	buildUC := usecases.NewBuildFromSourceUseCase(repositories.NewCryptoSSHExecutor())
	tag, err := buildUC.Execute(svc, commitRef, config, onLine)
	if err != nil {
		return "", err
	}

	onLine(fmt.Sprintf("▶ Desplegando %s...", tag))
	deployUC := usecases.NewTriggerWebhookDeployUseCase(w.repo, repositories.NewCryptoSSHExecutor())
	rec, err := deployUC.Execute(svc.Name, tag, config)
	if err != nil {
		return tag, fmt.Errorf("build exitoso pero falló el redeploy: %w", err)
	}
	onLine(fmt.Sprintf("✅ Desplegado %s", rec.ImageTag))
	return tag, nil
}

// --- Build-from-source: rebuild manual y streaming de logs ---

func (w *WebServer) handleServiceRebuild(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimSpace(req.URL.Query().Get("name"))
	if name == "" {
		var reqData struct {
			Name string `json:"name"`
		}
		if req.Body != nil {
			if err := json.NewDecoder(req.Body).Decode(&reqData); err == nil {
				name = strings.TrimSpace(reqData.Name)
			}
		}
	}
	if !isValidNodeID(name) {
		jsonError(rw, "Parámetro 'name' de servicio requerido o inválido", http.StatusBadRequest)
		return
	}

	serverTarget := req.URL.Query().Get("server")
	cfg := w.resolveTargetServer(serverTarget)
	if cfg == nil || (cfg.Host == "" && !isLocalConfig(cfg)) {
		jsonError(rw, "Servidor VPS no configurado", http.StatusBadRequest)
		return
	}

	svc, err := w.repo.GetService(name, cfg.Name)
	if err != nil || svc == nil {
		jsonError(rw, fmt.Sprintf("Servicio '%s' no encontrado", name), http.StatusNotFound)
		return
	}
	if svc.SourceType != "git" {
		jsonError(rw, "Este servicio no está configurado con origen 'git' (build-from-source)", http.StatusBadRequest)
		return
	}

	flusher, ok := setupStreaming(rw)
	if !ok {
		http.Error(rw, "Streaming no soportado", http.StatusInternalServerError)
		return
	}
	send := func(t, m string) { streamJSON(rw, flusher, t, m) }

	tag, errBuild := w.runBuildAndDeploy(*svc, "", *cfg, func(line string) { send("log", line) })
	if errBuild != nil {
		send("error", errBuild.Error())
		return
	}

	_ = w.repo.SaveAuditLog(domain.AuditLog{
		Action:       "DEPLOY",
		ResourceType: "service",
		ResourceName: name,
		Details:      fmt.Sprintf("Rebuild manual desde git (imagen: %s)", tag),
		Timestamp:    time.Now(),
	})

	streamDoneJSON(rw, flusher, map[string]string{"imageTag": tag})
}

// handleBuildStream transmite (NDJSON) el log de un build disparado por webhook: las
// líneas ya acumuladas primero, y las nuevas en vivo hasta que el build termina.
func (w *WebServer) handleBuildStream(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonError(rw, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(req.URL.Query().Get("id"))
	if id == "" {
		jsonError(rw, "Parámetro 'id' de build requerido", http.StatusBadRequest)
		return
	}
	job := w.builds.Get(id)
	if job == nil {
		jsonError(rw, "Build no encontrado (puede que el proceso se haya reiniciado)", http.StatusNotFound)
		return
	}

	flusher, ok := setupStreaming(rw)
	if !ok {
		http.Error(rw, "Streaming no soportado", http.StatusInternalServerError)
		return
	}
	send := func(t, m string) { streamJSON(rw, flusher, t, m) }

	linesSoFar, live := job.Subscribe()
	for _, l := range linesSoFar {
		send("log", l)
	}
	if live != nil {
		for l := range live {
			send("log", l)
		}
	}

	if job.Status == "failed" {
		send("error", job.Error)
		return
	}
	streamDoneJSON(rw, flusher, map[string]string{"status": job.Status, "imageTag": job.ImageTag})
}

