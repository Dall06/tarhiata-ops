package terraform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"
)

// Runner encapsula la ejecución genérica de Terraform.
type Runner struct {
	workspace string
	execPath  string
}

// NewRunner prepara el entorno de Terraform, descargando el binario si es necesario.
func NewRunner(workspace string) (*Runner, error) {
	fmt.Println("⏳ [Terraform] Preparando binario de Terraform...")
	installer := &releases.ExactVersion{
		Product: product.Terraform,
		Version: version.Must(version.NewVersion("1.5.7")),
	}

	execPath, err := installer.Install(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error instalando terraform: %w", err)
	}

	if err := os.MkdirAll(workspace, 0755); err != nil {
		return nil, fmt.Errorf("error creando workspace de terraform: %w", err)
	}

	return &Runner{
		workspace: workspace,
		execPath:  execPath,
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

	// Convertir el map de variables a tfexec.ApplyOption
	var applyOpts []tfexec.ApplyOption
	for k, v := range vars {
		applyOpts = append(applyOpts, tfexec.Var(fmt.Sprintf("%s=%s", k, v)))
	}

	if err := tf.Apply(context.Background(), applyOpts...); err != nil {
		return nil, fmt.Errorf("error en tf apply: %w", err)
	}

	// Extraer los outputs
	tfOutputs, err := tf.Output(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error leyendo tf outputs: %w", err)
	}

	parsedOutputs := make(map[string]string)
	for k, v := range tfOutputs {
		valStr := string(v.Value)
		// Limpiar las comillas que pone json.RawMessage
		valStr = strings.Trim(valStr, "\"")
		valStr = strings.ReplaceAll(valStr, "\\n", "\n")
		parsedOutputs[k] = valStr
	}

	return parsedOutputs, nil
}
