package usecases

import (
	"encoding/base64"
	"fmt"

	"github.com/Dall06/tarhiata-ops/pkg/validator"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type ManageDBMigrationsUseCase struct {
	repo ports.ConfigRepository
	ssh  ports.SSHExecutor
}

func NewManageDBMigrationsUseCase(repo ports.ConfigRepository, ssh ports.SSHExecutor) ports.ManageDBMigrationsUseCase {
	return &ManageDBMigrationsUseCase{
		repo: repo,
		ssh:  ssh,
	}
}

func (uc *ManageDBMigrationsUseCase) GetFiles(dbName string) ([]domain.MigrationFile, error) {
	return uc.repo.GetMigrationFiles(dbName)
}

func (uc *ManageDBMigrationsUseCase) SaveFile(dbName, filename, content, downContent string) error {
	file := domain.MigrationFile{
		DBName:      dbName,
		Filename:    filename,
		Content:     content,
		DownContent: downContent,
		Status:      "pending",
	}
	return uc.repo.SaveMigrationFile(file)
}

func (uc *ManageDBMigrationsUseCase) DeleteFile(dbName, filename string) error {
	return uc.repo.DeleteMigrationFile(dbName, filename)
}

func (uc *ManageDBMigrationsUseCase) Execute(req domain.DatabaseMigrationRequest, config domain.ServerConfig) ([]domain.MigrationFile, error) {
	if req.TargetDB == "" {
		return nil, fmt.Errorf("debes especificar la base de datos de destino")
	}

	databases, err := uc.repo.GetDatabases(config.Name)
	if err != nil {
		return nil, fmt.Errorf("error leyendo bases de datos: %w", err)
	}

	var targetDB *domain.SavedDatabase
	for i := range databases {
		if databases[i].Name == req.TargetDB {
			targetDB = &databases[i]
			break
		}
	}

	if targetDB == nil {
		return nil, fmt.Errorf("la base de datos '%s' no se encuentra en el catálogo", req.TargetDB)
	}

	if err := uc.ssh.Connect(config); err != nil {
		return nil, fmt.Errorf("error de conexión SSH al nodo: %w", err)
	}
	defer uc.ssh.Close()

	allFiles, err := uc.repo.GetMigrationFiles(req.TargetDB)
	if err != nil {
		return nil, fmt.Errorf("error leyendo archivos de migración: %w", err)
	}

	serviceName := fmt.Sprintf("tarhiata-db-%s", targetDB.Name)

	if len(req.Filenames) > 0 {
		return uc.executeFiles(req, targetDB, allFiles, serviceName)
	}

	if req.SqlContent != "" {
		return uc.executeDirectSQL(req.SqlContent, targetDB, serviceName)
	}

	return nil, nil
}

func (uc *ManageDBMigrationsUseCase) executeFiles(req domain.DatabaseMigrationRequest, targetDB *domain.SavedDatabase, allFiles []domain.MigrationFile, serviceName string) ([]domain.MigrationFile, error) {
	fileMap := make(map[string]domain.MigrationFile)
	for _, f := range allFiles {
		fileMap[f.Filename] = f
	}

	isDownAction := req.Action == "down"
	var executed []domain.MigrationFile

	for _, fname := range req.Filenames {
		mf, exists := fileMap[fname]
		if !exists {
			continue
		}

		sqlToRun := mf.Content
		if isDownAction {
			if mf.DownContent == "" {
				return nil, fmt.Errorf("el archivo de migración '%s' no tiene sentencias SQL de regresión (DownContent) definidas", fname)
			}
			sqlToRun = mf.DownContent
		}

		b64Content := base64.StdEncoding.EncodeToString([]byte(sqlToRun))
		execCmd := buildMigrationCommand(targetDB.Engine, targetDB.Password, serviceName, b64Content)

		res, errExec := uc.ssh.RunCommand(execCmd)
		status := "applied"
		if isDownAction {
			status = "reverted"
		}
		logs := ""
		if res != nil {
			logs = res.Output
		}
		if errExec != nil || (res != nil && res.ExitCode != 0) {
			status = "failed"
			if logs == "" && errExec != nil {
				logs = errExec.Error()
			}
		}

		if recErr := uc.repo.RecordMigrationExecution(targetDB.Name, fname, status, logs); recErr != nil {
			fmt.Printf("⚠️ Error registrando ejecución de migración: %v\n", recErr)
		}
		mf.Status = status
		mf.LogOutput = logs
		executed = append(executed, mf)

		if status == "failed" {
			break
		}
	}

	return executed, nil
}

func (uc *ManageDBMigrationsUseCase) executeDirectSQL(sqlContent string, targetDB *domain.SavedDatabase, serviceName string) ([]domain.MigrationFile, error) {
	b64Content := base64.StdEncoding.EncodeToString([]byte(sqlContent))
	execCmd := buildMigrationCommand(targetDB.Engine, targetDB.Password, serviceName, b64Content)

	res, errExec := uc.ssh.RunCommand(execCmd)
	status := "applied"
	logs := ""
	if res != nil {
		logs = res.Output
	}
	if errExec != nil || (res != nil && res.ExitCode != 0) {
		status = "failed"
		if logs == "" && errExec != nil {
			logs = errExec.Error()
		}
	}

	directFile := domain.MigrationFile{
		DBName:    targetDB.Name,
		Filename:  "direct_sql_execution.sql",
		Content:   sqlContent,
		Status:    status,
		LogOutput: logs,
	}
	return []domain.MigrationFile{directFile}, nil
}

func buildMigrationCommand(engine, password, serviceName, b64Content string) string {
	quotedService := validator.ShellQuote(serviceName)
	switch engine {
	case "postgres":
		return fmt.Sprintf("echo '%s' | base64 -d | docker exec -i $(docker ps -q -f name=%s | head -n 1) psql -U admin -d db", b64Content, quotedService)
	case "mongo", "mongodb":
		return fmt.Sprintf("echo '%s' | base64 -d | docker exec -i $(docker ps -q -f name=%s | head -n 1) mongosh -u admin -p %s db", b64Content, quotedService, validator.ShellQuote(password))
	default:
		return fmt.Sprintf("echo '%s' | base64 -d | docker exec -i $(docker ps -q -f name=%s | head -n 1) mysql -u admin -p%s db", b64Content, quotedService, validator.ShellQuote(password))
	}
}
