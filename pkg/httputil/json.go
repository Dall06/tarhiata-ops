package httputil

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
)

// JSON serializa y escribe una respuesta JSON con el código de estado indicado.
func JSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("httputil: error serializando json", "error", err)
		return err
	}
	return nil
}

// Error escribe un error estructurado en formato JSON {"error": message}.
func Error(w http.ResponseWriter, statusCode int, message string) error {
	return JSON(w, statusCode, map[string]string{"error": message})
}

// SetupStreaming configura los encabezados HTTP para streaming NDJSON y obtiene el flusher.
func SetupStreaming(w http.ResponseWriter) (http.Flusher, error) {
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("httputil: responseWriter no soporta streaming (http.Flusher)")
	}
	return flusher, nil
}

// WriteEvent envía un evento de progreso al cliente en formato NDJSON {"t": eventType, "m": msg}.
func WriteEvent(w http.ResponseWriter, flusher http.Flusher, eventType, msg string) error {
	data, err := json.Marshal(map[string]string{"t": eventType, "m": msg})
	if err != nil {
		slog.Warn("httputil: error serializando evento NDJSON", "error", err)
		return err
	}
	if _, err := fmt.Fprintf(w, "%s\n", data); err != nil {
		return err
	}
	if flusher != nil {
		flusher.Flush()
	}
	return nil
}

// WriteDone envía el evento final de éxito con payload al cliente {"t": "done", "d": result}.
func WriteDone(w http.ResponseWriter, flusher http.Flusher, result map[string]string) error {
	data, err := json.Marshal(map[string]any{"t": "done", "d": result})
	if err != nil {
		slog.Warn("httputil: error serializando evento final NDJSON", "error", err)
		return err
	}
	if _, err := fmt.Fprintf(w, "%s\n", data); err != nil {
		return err
	}
	if flusher != nil {
		flusher.Flush()
	}
	return nil
}

// IsLoopback determina si una dirección IP remota corresponde a loopback local (127.0.0.1 o ::1).
func IsLoopback(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil {
		return host == "localhost"
	}
	return ip.IsLoopback()
}
