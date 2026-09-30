package usecases

import (
	"errors"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

type fakeVultrClient struct {
	plans    []domain.VultrPlan
	regions  []domain.VultrRegion
	sshKeys  []domain.VultrSSHKey
	plansErr error
}

func (f *fakeVultrClient) GetPlans(apiKey string) ([]domain.VultrPlan, error) {
	if f.plansErr != nil {
		return nil, f.plansErr
	}
	return f.plans, nil
}

func (f *fakeVultrClient) GetRegions(apiKey string) ([]domain.VultrRegion, error) {
	return f.regions, nil
}

func (f *fakeVultrClient) GetSSHKeys(apiKey string) ([]domain.VultrSSHKey, error) {
	return f.sshKeys, nil
}

func TestListVultrPlansUseCase_Execute(t *testing.T) {
	t.Run("filtra planes con costo 0 y ordena ascendente", func(t *testing.T) {
		fake := &fakeVultrClient{plans: []domain.VultrPlan{
			{ID: "vc2-free", MonthlyCost: 0},
			{ID: "vc2-6", MonthlyCost: 6.0, RAM: 2048},
			{ID: "vc2-2", MonthlyCost: 2.5, RAM: 1024},
			{ID: "vc2-2b", MonthlyCost: 2.5, RAM: 512},
		}}
		uc := NewListVultrPlansUseCase().WithVultrClient(fake)

		plans, err := uc.ExecutePlans("token")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(plans) != 3 {
			t.Fatalf("se esperaban 3 planes (sin el gratuito), got %d", len(plans))
		}
		if plans[0].ID != "vc2-2b" || plans[1].ID != "vc2-2" || plans[2].ID != "vc2-6" {
			t.Errorf("orden inesperado: %+v", plans)
		}
	})

	t.Run("error del cliente Vultr se propaga", func(t *testing.T) {
		fake := &fakeVultrClient{plansErr: errors.New("vultr caído")}
		uc := NewListVultrPlansUseCase().WithVultrClient(fake)

		if _, err := uc.ExecutePlans("token"); err == nil {
			t.Fatal("se esperaba error propagado del cliente Vultr")
		}
	})

	t.Run("regiones se devuelven tal cual del cliente", func(t *testing.T) {
		fake := &fakeVultrClient{regions: []domain.VultrRegion{{ID: "mex", City: "Mexico City"}}}
		uc := NewListVultrPlansUseCase().WithVultrClient(fake)

		regions, err := uc.ExecuteRegions("token")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(regions) != 1 || regions[0].ID != "mex" {
			t.Errorf("unexpected regions: %+v", regions)
		}
	})
}
