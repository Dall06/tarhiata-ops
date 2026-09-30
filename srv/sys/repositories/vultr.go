package repositories

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

// VultrHTTPClient implementa ports.VultrClient contra la API pública real de Vultr v2.
type VultrHTTPClient struct {
	client *http.Client
}

func NewVultrHTTPClient() *VultrHTTPClient {
	return &VultrHTTPClient{client: &http.Client{Timeout: 10 * time.Second}}
}

type vultrPlansResponse struct {
	Plans []domain.VultrPlan `json:"plans"`
}

type vultrRegionsResponse struct {
	Regions []domain.VultrRegion `json:"regions"`
}

type vultrSSHKeysResponse struct {
	SSHKeys []domain.VultrSSHKey `json:"ssh_keys"`
}

func (c *VultrHTTPClient) get(path, apiKey string, out any) error {
	req, err := http.NewRequest(http.MethodGet, "https://api.vultr.com/v2/"+path, nil)
	if err != nil {
		return err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("error al conectar con Vultr API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("vultr api devolvió status HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *VultrHTTPClient) GetPlans(apiKey string) ([]domain.VultrPlan, error) {
	var data vultrPlansResponse
	if err := c.get("plans", apiKey, &data); err != nil {
		return nil, fmt.Errorf("error al decodificar planes de Vultr: %w", err)
	}
	return data.Plans, nil
}

func (c *VultrHTTPClient) GetRegions(apiKey string) ([]domain.VultrRegion, error) {
	var data vultrRegionsResponse
	if err := c.get("regions", apiKey, &data); err != nil {
		return nil, fmt.Errorf("error al decodificar regiones de Vultr: %w", err)
	}
	return data.Regions, nil
}

func (c *VultrHTTPClient) GetSSHKeys(apiKey string) ([]domain.VultrSSHKey, error) {
	var data vultrSSHKeysResponse
	if err := c.get("ssh-keys", apiKey, &data); err != nil {
		return nil, fmt.Errorf("error al decodificar llaves SSH de Vultr: %w", err)
	}
	return data.SSHKeys, nil
}
