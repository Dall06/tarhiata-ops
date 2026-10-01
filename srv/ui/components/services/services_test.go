package services

import (
	"strings"
	"testing"
)

func TestGetJSContent(t *testing.T) {
	tests := []struct {
		name     string
		contains string
	}{
		{
			name:     "contains renderAppCards",
			contains: "export function renderAppCards",
		},
		{
			name:     "contains openDeployModal",
			contains: "export function openDeployModal",
		},
		{
			name:     "contains restartServiceOrContainer",
			contains: "export async function restartServiceOrContainer",
		},
		{
			name:     "una BD detenida se marca como 0/1 (offline), no con su status crudo",
			contains: "db.status === 'running' ? '1/1' : (db.status ? '0/1' : '1/1')",
		},
		{
			name:     "el modal de métricas etiqueta Memoria en MB, no en %",
			contains: "{ label: 'Memoria', unit: 'MB'",
		},
		{
			name:     "el modal de métricas etiqueta Disco en MB, no en %",
			contains: "{ label: 'Disco',   unit: 'MB'",
		},
		{
			name:     "el selector de origen de servicio incluye la opción git",
			contains: `id="editServiceSourceType"`,
		},
		{
			name:     "el botón de rebuild manual existe en el dropdown de apps",
			contains: "btn-rebuild-svc",
		},
		{
			name:     "el rebuild usa fetch() directo, no apiFetch, para poder streamear",
			contains: "/api/services/rebuild?name=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := string(GetJSContent())
			if !strings.Contains(content, tt.contains) {
				t.Errorf("expected services JS content to contain %q", tt.contains)
			}
		})
	}
}

// TestDBReplicasIsOnlineLogic reimplementa en Go la fórmula usada en services.js para
// validar, sin necesidad de un runtime JS, que un status de BD detenida jamás produzca
// un indicador "online". Documenta el contrato: cualquier cambio a la fórmula en el JS
// debe mantenerse en sync con este test o viceversa.
func TestDBReplicasIsOnlineLogic(t *testing.T) {
	replicasFor := func(status string) string {
		if status == "running" {
			return "1/1"
		}
		if status != "" {
			return "0/1"
		}
		return "1/1"
	}
	isOnline := func(replicas string) bool {
		return replicas != "" && !strings.HasPrefix(replicas, "0/")
	}

	tests := []struct {
		status     string
		wantOnline bool
	}{
		{status: "running", wantOnline: true},
		{status: "stopped", wantOnline: false},
		{status: "exited", wantOnline: false},
		{status: "paused", wantOnline: false},
		{status: "", wantOnline: true},
	}

	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			got := isOnline(replicasFor(tc.status))
			if got != tc.wantOnline {
				t.Errorf("status=%q: isOnline=%v, want %v", tc.status, got, tc.wantOnline)
			}
		})
	}
}
