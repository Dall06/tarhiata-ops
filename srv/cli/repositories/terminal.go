package repositories

import (
	"fmt"
	"io"
	"os"

	"github.com/Dall06/tarhiata-ops/srv/cli/domain"
)

// TerminalPresenter maneja el formateo y salida hacia el terminal.
type TerminalPresenter struct {
	out io.Writer
}

// NewTerminalPresenter inicializa el presentador con la salida estándar por defecto.
func NewTerminalPresenter() *TerminalPresenter {
	return &TerminalPresenter{out: os.Stdout}
}

// NewTerminalPresenterWithWriter inicializa el presentador con un writer personalizado para tests.
func NewTerminalPresenterWithWriter(w io.Writer) *TerminalPresenter {
	return &TerminalPresenter{out: w}
}

// PrintCard imprime una tarjeta estilizada en el writer configurado.
func (p *TerminalPresenter) PrintCard(card domain.DashboardCard) error {
	_, err := fmt.Fprintf(p.out, "[%s] %s: %s %s\n", card.Color, card.Title, card.Value, card.Unit)
	return err
}
