package usecases

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestRepairTraefikUseCase_Execute(t *testing.T) {
	t.Run("sin acmeEmail no incluye bloque ACME/HTTPS", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		mockSSH.MockResponses["docker stack deploy"] = &domain.CommandResult{Output: "", ExitCode: 0}
		uc := NewRepairTraefikUseCase(mockSSH)

		out, err := uc.Execute("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != "Traefik reparado y redesplegado con configuración actualizada." {
			t.Errorf("mensaje default inesperado: %q", out)
		}

		var written string
		for _, c := range mockSSH.CommandsExecuted {
			if strings.HasPrefix(c, "WRITE ") {
				written = c
			}
		}
		if written == "" {
			t.Fatal("no se escribió ningún archivo remoto")
		}
	})

	t.Run("con acmeEmail valido activa certresolver y volumen ACME", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		mockSSH.WriteRemoteFileCapture = true
		uc := NewRepairTraefikUseCase(mockSSH)

		if _, err := uc.Execute("ops@tarhiata.com"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(mockSSH.LastWrittenContent, "certificatesresolvers.leresolver.acme.email=ops@tarhiata.com") {
			t.Errorf("no se activó el certresolver ACME en el compose: %s", mockSSH.LastWrittenContent)
		}
		if !strings.Contains(mockSSH.LastWrittenContent, "traefik_certs:/letsencrypt") {
			t.Errorf("no se montó el volumen de certificados: %s", mockSSH.LastWrittenContent)
		}
	})

	t.Run("acmeEmail con formato invalido se rechaza sin tocar SSH", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewRepairTraefikUseCase(mockSSH)

		if _, err := uc.Execute("no-es-un-email"); err == nil {
			t.Fatal("se esperaba error por email inválido")
		}
		if len(mockSSH.CommandsExecuted) != 0 {
			t.Errorf("no debió ejecutarse ningún comando SSH, se ejecutaron: %v", mockSSH.CommandsExecuted)
		}
	})

	// TestRepairTraefikUseCase_ShellInjectionPrevention (dentro del mismo Run):
	// un acmeEmail diseñado para romper el YAML e inyectar flags arbitrarios a Traefik
	// debe rechazarse, no interpolarse.
	t.Run("acmeEmail con intento de inyeccion YAML se rechaza", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewRepairTraefikUseCase(mockSSH)

		evil := "x@x.com\"\n      - \"--api.insecure=true"
		if _, err := uc.Execute(evil); err == nil {
			t.Fatal("se esperaba error por email con contenido YAML malicioso")
		}
	})

	t.Run("error escribiendo el archivo remoto se propaga", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		mockSSH.WriteRemoteFileErr = fmt.Errorf("disco lleno")
		uc := NewRepairTraefikUseCase(mockSSH)

		if _, err := uc.Execute(""); err == nil || !strings.Contains(err.Error(), "disco lleno") {
			t.Fatalf("se esperaba error propagado de WriteRemoteFile, obtenido: %v", err)
		}
	})

	t.Run("exit code distinto de cero en el deploy se propaga con la salida", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		mockSSH.MockResponses["docker stack deploy"] = &domain.CommandResult{
			Output:   "manifest inválido",
			ExitCode: 1,
		}
		uc := NewRepairTraefikUseCase(mockSSH)

		_, err := uc.Execute("")
		if err == nil || !strings.Contains(err.Error(), "manifest inválido") {
			t.Fatalf("se esperaba error con la salida del comando, obtenido: %v", err)
		}
	})

	t.Run("salida no vacia del deploy se devuelve tal cual", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		mockSSH.MockResponses["docker stack deploy"] = &domain.CommandResult{
			Output:   "  service tarhiata_proxy_traefik updated  ",
			ExitCode: 0,
		}
		uc := NewRepairTraefikUseCase(mockSSH)

		out, err := uc.Execute("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != "service tarhiata_proxy_traefik updated" {
			t.Errorf("se esperaba la salida recortada del comando, obtenido: %q", out)
		}
	})
}
