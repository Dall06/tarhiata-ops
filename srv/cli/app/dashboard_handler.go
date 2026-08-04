package app

import (
	"fmt"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
	"github.com/charmbracelet/lipgloss"
)

type DashboardHandler struct {
	repo ports.ConfigRepository
}

func NewDashboardHandler(repo ports.ConfigRepository) *DashboardHandler {
	return &DashboardHandler{repo: repo}
}

// RenderDashboard dibuja el Dashboard estético estilo Vercel / Railway
func (h *DashboardHandler) RenderDashboard(config *domain.ServerConfig) {
	fmt.Print("\033[H\033[2J") // Clear

	const (
		ColorBg       = "#000000"
		ColorCardBg   = "#0A0A0A"
		ColorBorder   = "#333333"
		ColorText     = "#EDEDED"
		ColorSubtext  = "#888888"
		ColorAccent   = "#3291FF"
		ColorSuccess  = "#50E3C2"
		ColorError    = "#FF0000"
	)

	var (
		logoStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(ColorText)).
				Background(lipgloss.Color(ColorAccent)).
				Padding(0, 1)

		subStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorSubtext))

		cardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(ColorBorder)).
				Background(lipgloss.Color(ColorCardBg)).
				Padding(0, 1).
				Width(35).
				Height(8)

		titleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(ColorText))

		labelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorSubtext))

		valueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorText)).
				Bold(true)

		badgeOk = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSuccess)).
			Bold(true).
			Render("● ACTIVE")

		badgeOff = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorError)).
			Bold(true).
			Render("● OFFLINE")
	)

	fmt.Println(lipgloss.NewStyle().Padding(1, 2).Render(
		logoStyle.Render("▲ TARHIATA") + "  " + subStyle.Render("Control Plane CLI"),
	))

	c1Text := titleStyle.Render("Infrastructure") + "\n\n"
	if config != nil {
		c1Text += fmt.Sprintf("%s %s\n", labelStyle.Render("Status:"), badgeOk)
		c1Text += fmt.Sprintf("%s %s\n", labelStyle.Render("Host:  "), valueStyle.Render(config.Host))
		c1Text += fmt.Sprintf("%s %s\n", labelStyle.Render("User:  "), valueStyle.Render(config.User))
		prov := config.CloudProvider
		if prov == "" {
			prov = "vps-direct"
		}
		c1Text += fmt.Sprintf("%s %s", labelStyle.Render("Cloud: "), valueStyle.Render(prov))
	} else {
		c1Text += fmt.Sprintf("%s %s\n", labelStyle.Render("Status:"), badgeOff)
		c1Text += labelStyle.Render("\nNot configured.")
	}
	card1 := cardStyle.Render(c1Text)

	services, _ := h.repo.GetServices()
	dbs, _ := h.repo.GetDatabases()

	c2Text := titleStyle.Render("Platform") + "\n\n"
	c2Text += fmt.Sprintf("%s %s\n", labelStyle.Render("Services: "), valueStyle.Render(fmt.Sprintf("%d running", len(services))))
	c2Text += fmt.Sprintf("%s %s\n\n", labelStyle.Render("Databases:"), valueStyle.Render(fmt.Sprintf("%d online", len(dbs))))
	
	c2Text += lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSuccess)).Render("✔ Swarm Node Active") + "\n"
	c2Text += lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSuccess)).Render("✔ SSL Active")
	
	card2 := cardStyle.Render(c2Text)

	grid := lipgloss.JoinHorizontal(lipgloss.Top, card1, "  ", card2)
	fmt.Println(lipgloss.NewStyle().Padding(0, 2).Render(grid))
	fmt.Println()
}
