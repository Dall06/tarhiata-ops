package ports

import "github.com/Dall06/tarhiata-ops/srv/sys/domain"

type InitServerUseCase interface {
	Execute(acmeEmail string) error
}



type DeployObservabilityUseCase interface {
	Execute(exposePublic bool) error
	ExecutePersistent(exposePublic bool, deployType string, grafanaPassword string) error
	ExecutePersistentWithVolume(exposePublic bool, deployType string, grafanaPassword string, volumePath string) error
}

type DeployDatabaseUseCase interface {
	Execute(db domain.SavedDatabase, config domain.ServerConfig) error
}

type ProvisionWorkerUseCase interface {
	Execute(config domain.ServerConfig, nodeName string, labelType string) (string, error)
	ExecuteWithRegion(config domain.ServerConfig, nodeName string, labelType string, region string) (string, error)
	ExecuteWithPlanAndRegion(config domain.ServerConfig, nodeName string, labelType string, plan string, region string) (string, error)
}

type DeployServiceUseCase interface {
	Execute(service domain.CustomService, config domain.DeployConfig) error
}

type UpdateServerUseCase interface {
	Execute() error
}

type LinkServicesUseCase interface {
	Execute(sourceSvc string, targetSvc string, envVarName string) (domain.ServiceLink, error)
}

type UnlinkServicesUseCase interface {
	Execute(sourceSvc string, targetSvc string) error
}

type BootstrapMasterInput struct {
	AppName      string `json:"app_name"`
	Image        string `json:"image"`
	Port         int    `json:"port"`
	Domain       string `json:"domain"`
	ExposePublic bool   `json:"expose_public"`
	DBEngine     string `json:"db_engine"`
	EnvVarName   string `json:"env_var_name"`
	TargetNode   string `json:"target_node,omitempty"`
}

type BootstrapMasterResult struct {
	App         domain.SavedService   `json:"app"`
	Database    *domain.SavedDatabase `json:"database,omitempty"`
	Link        *domain.ServiceLink   `json:"link,omitempty"`
	UnlinkedOld []string              `json:"unlinked_old,omitempty"`
}

type BootstrapMasterServiceUseCase interface {
	Execute(input BootstrapMasterInput, config domain.ServerConfig) (*BootstrapMasterResult, error)
}

// --- Manage Preview Environments (Entornos Temporales Efímeros) ---

type CreatePreviewEnvInput struct {
	Name        string `json:"name"`
	Image       string `json:"image"`
	ImageSource string `json:"imageSource,omitempty"`
	Port        int    `json:"port"`
	Domain      string `json:"domain"`
	LinkDBName  string `json:"link_db_name,omitempty"`
	LinkDbName  string `json:"linkDbName,omitempty"`
	TargetNode  string `json:"target_node,omitempty"`
	NodeTarget  string `json:"targetNode,omitempty"`
}

type ManagePreviewEnvUseCase interface {
	Create(input CreatePreviewEnvInput, config domain.ServerConfig) (*domain.SavedPreviewEnv, error)
	List() ([]domain.SavedPreviewEnv, error)
	Destroy(name string, config domain.ServerConfig) error
}

type ManageRegistryAuthUseCase interface {
	Save(cred domain.SavedRegistryCredential, config domain.ServerConfig) error
	List() ([]domain.SavedRegistryCredential, error)
	Delete(server string, config domain.ServerConfig) error
}

type ManageDBMigrationsUseCase interface {
	GetFiles(dbName string) ([]domain.MigrationFile, error)
	SaveFile(dbName, filename, content, downContent string) error
	DeleteFile(dbName, filename string) error
	Execute(req domain.DatabaseMigrationRequest, config domain.ServerConfig) ([]domain.MigrationFile, error)
}

type SyncClusterStateUseCase interface {
	ExportStateToRemote() error
	ImportStateFromRemote() (interface{}, error)
}

type ManageSSHKeysUseCase interface {
	ListKeys(cfg domain.ServerConfig) ([]domain.SSHKeyInfo, error)
	AddKey(cfg domain.ServerConfig, publicKey string) error
	DeleteKey(cfg domain.ServerConfig, targetIdentifier string) error
}
