package usecases

import (
	"sort"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
	"github.com/Dall06/tarhiata-ops/srv/sys/repositories"
)

type ListVultrPlansUseCase struct {
	vultrClient ports.VultrClient
}

func NewListVultrPlansUseCase() *ListVultrPlansUseCase {
	return &ListVultrPlansUseCase{
		vultrClient: repositories.NewVultrHTTPClient(),
	}
}

// WithVultrClient permite inyectar un cliente Vultr falso para pruebas unitarias.
func (uc *ListVultrPlansUseCase) WithVultrClient(c ports.VultrClient) *ListVultrPlansUseCase {
	uc.vultrClient = c
	return uc
}

// ExecutePlans obtiene los planes de Vultr filtrados y ordenados por costo mensual.
func (uc *ListVultrPlansUseCase) ExecutePlans(apiKey string) ([]domain.VultrPlan, error) {
	plans, err := uc.vultrClient.GetPlans(apiKey)
	if err != nil {
		return nil, err
	}

	var filtered []domain.VultrPlan
	for _, p := range plans {
		// Filtrar planes activos estándar (vc2, vhc, vdc, vc2g)
		if p.MonthlyCost > 0 {
			filtered = append(filtered, p)
		}
	}

	// Ordenar por costo mensual ascendente ($2.50, $3.50, $5.00, $6.00...)
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].MonthlyCost == filtered[j].MonthlyCost {
			return filtered[i].RAM < filtered[j].RAM
		}
		return filtered[i].MonthlyCost < filtered[j].MonthlyCost
	})

	return filtered, nil
}

// ExecuteRegions obtiene las regiones geográficas disponibles de Vultr.
func (uc *ListVultrPlansUseCase) ExecuteRegions(apiKey string) ([]domain.VultrRegion, error) {
	return uc.vultrClient.GetRegions(apiKey)
}
