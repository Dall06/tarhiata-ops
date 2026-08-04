package dto

import (
	"encoding/json"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

// DecodeServerConfig decodifica tanto la notación camelCase como snake_case enviada desde la interfaz web o CLI.
func DecodeServerConfig(data []byte) (domain.ServerConfig, error) {
	var c domain.ServerConfig
	type Alias domain.ServerConfig
	aux := struct {
		Alias
		KeyPath       string `json:"key_path"`
		PrivateKeyAlt string `json:"private_key"`
		DOToken       string `json:"do_token"`
		VultrToken    string `json:"vultr_token"`
	}{
		Alias: Alias(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return c, err
	}
	c = domain.ServerConfig(aux.Alias)
	if c.PrivateKey == "" {
		if aux.KeyPath != "" {
			c.PrivateKey = aux.KeyPath
		} else if aux.PrivateKeyAlt != "" {
			c.PrivateKey = aux.PrivateKeyAlt
		}
	}
	if c.DOAPIToken == "" && aux.DOToken != "" {
		c.DOAPIToken = aux.DOToken
	}
	if c.VultrAPIToken == "" && aux.VultrToken != "" {
		c.VultrAPIToken = aux.VultrToken
	}
	if c.VultrAPIToken == "" && c.DOAPIToken != "" {
		c.VultrAPIToken = c.DOAPIToken
	}
	return c, nil
}

// DecodeSavedService decodifica servicios recibidos en notación camelCase o snake_case.
func DecodeSavedService(data []byte) (domain.SavedService, error) {
	var s domain.SavedService
	type Alias domain.SavedService
	aux := struct {
		Alias
		ImageSourceSnake    string `json:"image_source"`
		EnableSSLSnake      *bool  `json:"enable_ssl"`
		PreDeployHookSnake string `json:"pre_deploy_hook"`
		HealthcheckCmdSnake string `json:"healthcheck_cmd"`
		EnvVarsSnake        string `json:"env_vars"`
		EnvFilePathSnake    string `json:"env_file_path"`
		TargetNodeSnake     string `json:"target_node"`
	}{
		Alias: Alias(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return s, err
	}
	s = domain.SavedService(aux.Alias)
	if s.ImageSource == "" && aux.ImageSourceSnake != "" {
		s.ImageSource = aux.ImageSourceSnake
	}
	if aux.EnableSSLSnake != nil && !s.EnableSSL {
		s.EnableSSL = *aux.EnableSSLSnake
	}
	if s.PreDeployHook == "" && aux.PreDeployHookSnake != "" {
		s.PreDeployHook = aux.PreDeployHookSnake
	}
	if s.HealthcheckCmd == "" && aux.HealthcheckCmdSnake != "" {
		s.HealthcheckCmd = aux.HealthcheckCmdSnake
	}
	if s.EnvVars == "" && aux.EnvVarsSnake != "" {
		s.EnvVars = aux.EnvVarsSnake
	}
	if s.EnvFilePath == "" && aux.EnvFilePathSnake != "" {
		s.EnvFilePath = aux.EnvFilePathSnake
	}
	if s.TargetNode == "" && aux.TargetNodeSnake != "" {
		s.TargetNode = aux.TargetNodeSnake
	}
	return s, nil
}

// DecodeServiceLink decodifica enlaces recibidos en notación camelCase o snake_case.
func DecodeServiceLink(data []byte) (domain.ServiceLink, error) {
	var l domain.ServiceLink
	type Alias domain.ServiceLink
	aux := struct {
		Alias
		SourceSvcSnake  string `json:"source_svc"`
		TargetSvcSnake  string `json:"target_svc"`
		EnvVarNameSnake string `json:"env_var_name"`
	}{
		Alias: Alias(l),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return l, err
	}
	l = domain.ServiceLink(aux.Alias)
	if l.SourceSvc == "" && aux.SourceSvcSnake != "" {
		l.SourceSvc = aux.SourceSvcSnake
	}
	if l.TargetSvc == "" && aux.TargetSvcSnake != "" {
		l.TargetSvc = aux.TargetSvcSnake
	}
	if l.EnvVarName == "" && aux.EnvVarNameSnake != "" {
		l.EnvVarName = aux.EnvVarNameSnake
	}
	return l, nil
}

// DecodeSavedDatabase decodifica bases de datos recibidas en camelCase o snake_case.
func DecodeSavedDatabase(data []byte) (domain.SavedDatabase, error) {
	var d domain.SavedDatabase
	type Alias domain.SavedDatabase
	aux := struct {
		Alias
		DeployTypeSnake     string `json:"deploy_type"`
		ExternalURLSnake    string `json:"external_url"`
		InternalPortSnake   int    `json:"internal_port"`
		VolumeHostPathSnake string `json:"volume_host_path"`
		TargetNodeSnake     string `json:"target_node"`
	}{
		Alias: Alias(d),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return d, err
	}
	d = domain.SavedDatabase(aux.Alias)
	if d.DeployType == "" && aux.DeployTypeSnake != "" {
		d.DeployType = aux.DeployTypeSnake
	}
	if d.ExternalURL == "" && aux.ExternalURLSnake != "" {
		d.ExternalURL = aux.ExternalURLSnake
	}
	if d.InternalPort == 0 && aux.InternalPortSnake != 0 {
		d.InternalPort = aux.InternalPortSnake
	}
	if d.VolumeHostPath == "" && aux.VolumeHostPathSnake != "" {
		d.VolumeHostPath = aux.VolumeHostPathSnake
	}
	if d.TargetNode == "" && aux.TargetNodeSnake != "" {
		d.TargetNode = aux.TargetNodeSnake
	}
	return d, nil
}
