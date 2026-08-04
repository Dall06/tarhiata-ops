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
	installerOnce sync.Once
	cachedExecPath string
	installerErr   error
)

// Runner encapsula la ejecución genérica de Terraform.
type Runner struct {
	workspace string
	execPath  string
}

// NewRunner prepara el entorno de Terraform, descargando el binario si es necesario.
func NewRunner(workspace string) (*Runner, error) {
	installerOnce.Do(func() {
		// 1. Si ya existe 'terraform' instalado en el sistema (Homebrew, etc.), usarlo directamente
		if path, err := exec.LookPath("terraform"); err == nil && path != "" {
			cachedExecPath = path
			return
		}

		// 2. Si ya existe binario descargado persistentemente en ~/.config/tarhiata/bin/terraform
		homeDir, _ := os.UserHomeDir()
		binDir := filepath.Join(homeDir, ".config", "tarhiata", "bin")
		targetBin := filepath.Join(binDir, "terraform")
		if _, err := os.Stat(targetBin); err == nil {
			cachedExecPath = targetBin
			return
		}

		_ = os.MkdirAll(binDir, 0755)

		// 3. Descargar a directorio persistente ~/.config/tarhiata/bin para evitar rases con /var/folders/
		fmt.Println("⏳ [Terraform] Preparando binario de Terraform...")
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
		return nil, fmt.Errorf("error instalando terraform: %w", err)
	}

	if err := os.MkdirAll(workspace, 0755); err != nil {
		return nil, fmt.Errorf("error creando workspace de terraform: %w", err)
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

	fmt.Println("🚀 [Terraform] Inicializando módulos...")
	if err := tf.Init(context.Background(), tfexec.Upgrade(true)); err != nil {
		return nil, fmt.Errorf("error en tf init: %w", err)
	}

	fmt.Println("🏗️  [Terraform] Aprovisionando infraestructura (esto puede tomar 1-2 minutos)...")

	// Escribir variables limpias y seguras en terraform.tfvars.json
	if len(vars) > 0 {
		varData, err := json.Marshal(vars)
		if err == nil {
			_ = os.WriteFile(filepath.Join(r.workspace, "terraform.tfvars.json"), varData, 0644)
		}
	}

	if err := tf.Apply(context.Background()); err != nil {
		return nil, fmt.Errorf("error en tf apply: %w", err)
	}

	// Extraer los outputs
	tfOutputs, err := tf.Output(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error leyendo tf outputs: %w", err)
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
