package usecases

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

type ListVultrPlansUseCase struct {
	client *http.Client
}

func NewListVultrPlansUseCase() *ListVultrPlansUseCase {
	return &ListVultrPlansUseCase{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type vultrPlansResponse struct {
	Plans []domain.VultrPlan `json:"plans"`
}

type vultrRegionsResponse struct {
	Regions []domain.VultrRegion `json:"regions"`
}

// ExecutePlans obtiene los planes de Vultr filtrados y ordenados por costo mensual.
func (uc *ListVultrPlansUseCase) ExecutePlans(apiKey string) ([]domain.VultrPlan, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.vultr.com/v2/plans", nil)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := uc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error al conectar con Vultr API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vultr api devolvió status HTTP %d", resp.StatusCode)
	}

	var data vultrPlansResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("error al decodificar planes de Vultr: %w", err)
	}

	var filtered []domain.VultrPlan
	for _, p := range data.Plans {
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
	req, err := http.NewRequest(http.MethodGet, "https://api.vultr.com/v2/regions", nil)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := uc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error al conectar con Vultr API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vultr api devolvió status HTTP %d", resp.StatusCode)
	}

	var data vultrRegionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("error al decodificar regiones de Vultr: %w", err)
	}

	return data.Regions, nil
}
