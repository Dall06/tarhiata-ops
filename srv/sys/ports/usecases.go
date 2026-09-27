package ports

import "github.com/Dall06/tarhiata-ops/srv/sys/domain"

type ConnectServerUseCase interface {
	Execute(config domain.ServerConfig) (*domain.ConnectionResult, error)
}

type InspectHostUseCase interface {
	Execute(config domain.ServerConfig) (*domain.HostInspection, error)
	ExecuteMetricsOnly(config domain.ServerConfig) (*domain.HostMetrics, error)
	ExecuteServicesOnly(config domain.ServerConfig) ([]domain.HostSystemService, error)
}


// Aliases de structs de dominio para preservar compatibilidad en puertos
type ProvisionCloudRequest = domain.ProvisionCloudRequest
type BootstrapMasterInput = domain.BootstrapMasterInput
type BootstrapMasterResult = domain.BootstrapMasterResult
type CreatePreviewEnvInput = domain.CreatePreviewEnvInput

type ProvisionCloudServerUseCase interface {
	Execute(req ProvisionCloudRequest) (*domain.ConnectionResult, error)
}

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

type BootstrapMasterServiceUseCase interface {
	Execute(input BootstrapMasterInput, config domain.ServerConfig) (*BootstrapMasterResult, error)
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

type ListDevicesUseCase interface {
	Execute(config domain.ServerConfig) (*domain.HostDevices, error)
}

