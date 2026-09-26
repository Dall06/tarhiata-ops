package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"
)

var (
	installerOnce  sync.Once
	cachedExecPath string
	installerErr   error
)

// Runner encapsula la ejecución de IaC usando OpenTofu o Terraform.
type Runner struct {
	workspace string
	execPath  string
}

// EngineName retorna el nombre del motor detectado ("OpenTofu" o "Terraform").
func (r *Runner) EngineName() string {
	if strings.Contains(strings.ToLower(filepath.Base(r.execPath)), "tofu") {
		return "OpenTofu"
	}
	return "Terraform"
}

// ExecPath retorna la ruta del binario en uso.
func (r *Runner) ExecPath() string {
	return r.execPath
}

// DetectEngine busca en el sistema el ejecutable de OpenTofu ('tofu') o Terraform ('terraform').
func DetectEngine() string {
	// 1. Priorizar OpenTofu si está instalado en el PATH
	if path, err := exec.LookPath("tofu"); err == nil && path != "" {
		return path
	}

	// 2. Fallback a Terraform si está instalado en el PATH
	if path, err := exec.LookPath("terraform"); err == nil && path != "" {
		return path
	}

	// 3. Revisar binario persistente en ~/.config/tarhiata/bin
	homeDir, err := os.UserHomeDir()
	if err == nil {
		binDir := filepath.Join(homeDir, ".config", "tarhiata", "bin")
		targetTofu := filepath.Join(binDir, "tofu")
		if _, err := os.Stat(targetTofu); err == nil {
			return targetTofu
		}
		targetTf := filepath.Join(binDir, "terraform")
		if _, err := os.Stat(targetTf); err == nil {
			return targetTf
		}
	}

	return ""
}

// NewRunner prepara el entorno de IaC (OpenTofu / Terraform), detectando o descargando el binario si es necesario.
func NewRunner(workspace string) (*Runner, error) {
	installerOnce.Do(func() {
		detected := DetectEngine()
		if detected != "" {
			cachedExecPath = detected
			return
		}

		homeDir, err := os.UserHomeDir()
		if err != nil {
			installerErr = fmt.Errorf("no se pudo obtener homeDir: %w", err)
			return
		}

		binDir := filepath.Join(homeDir, ".config", "tarhiata", "bin")
		if err := os.MkdirAll(binDir, 0755); err != nil {
			installerErr = fmt.Errorf("error creando directorio de binarios: %w", err)
			return
		}

		// Descargar la última versión Open Source (1.5.7) como motor embebido
		fmt.Println("⏳ [IaC Engine] Preparando binario open source de Terraform/OpenTofu...")
		installer := &releases.ExactVersion{
			Product:    product.Terraform,
			Version:    version.Must(version.NewVersion("1.5.7")),
			InstallDir: binDir,
		}
		cachedExecPath, installerErr = installer.Install(context.Background())
	})

	if installerErr != nil {
		installerOnce = sync.Once{} // Permitir reintento si falló
		err := installerErr
		installerErr = nil
		return nil, fmt.Errorf("error preparando motor IaC: %w", err)
	}

	if err := os.MkdirAll(workspace, 0755); err != nil {
		return nil, fmt.Errorf("error creando workspace de IaC: %w", err)
	}

	return &Runner{
		workspace: workspace,
		execPath:  cachedExecPath,
	}, nil
}

// Apply escribe el archivo HCL en el workspace, lo inicializa y aplica los cambios.
// Retorna un mapa con las salidas (outputs) extraídas limpiamente.
func (r *Runner) Apply(tfContent string, vars map[string]string) (map[string]string, error) {
	tfFilePath := filepath.Join(r.workspace, "main.tf")
	if err := os.WriteFile(tfFilePath, []byte(tfContent), 0644); err != nil {
		return nil, fmt.Errorf("error escribiendo main.tf: %w", err)
	}

	tf, err := tfexec.NewTerraform(r.workspace, r.execPath)
	if err != nil {
		return nil, fmt.Errorf("error creando instancia tfexec: %w", err)
	}

	fmt.Printf("🚀 [%s] Inicializando módulos...\n", r.EngineName())
	if err := tf.Init(context.Background(), tfexec.Upgrade(true)); err != nil {
		return nil, fmt.Errorf("error en init (%s): %w", r.EngineName(), err)
	}

	fmt.Printf("🏗️  [%s] Aprovisionando infraestructura en la nube...\n", r.EngineName())

	// Escribir variables limpias y seguras en terraform.tfvars.json
	if len(vars) > 0 {
		varData, err := json.Marshal(vars)
		if err != nil {
			return nil, fmt.Errorf("error serializando variables: %w", err)
		}
		if err := os.WriteFile(filepath.Join(r.workspace, "terraform.tfvars.json"), varData, 0644); err != nil {
			return nil, fmt.Errorf("error guardando terraform.tfvars.json: %w", err)
		}
	}

	if err := tf.Apply(context.Background()); err != nil {
		return nil, fmt.Errorf("error en apply (%s): %w", r.EngineName(), err)
	}

	// Extraer los outputs
	tfOutputs, err := tf.Output(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error leyendo outputs (%s): %w", r.EngineName(), err)
	}

	parsedOutputs := make(map[string]string)
	for k, v := range tfOutputs {
		var strVal string
		if err := json.Unmarshal(v.Value, &strVal); err == nil {
			parsedOutputs[k] = strVal
		} else {
			parsedOutputs[k] = strings.Trim(string(v.Value), "\"")
		}
	}

	return parsedOutputs, nil
}
