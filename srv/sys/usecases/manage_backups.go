package usecases

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/Dall06/tarhiata-ops/pkg/validator"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type SnapshotDownloadResult = domain.SnapshotDownloadResult

type ManageBackupsUseCase struct {
	repo ports.ConfigRepository
	ssh  ports.SSHExecutor
}

func NewManageBackupsUseCase(repo ports.ConfigRepository, ssh ports.SSHExecutor) *ManageBackupsUseCase {
	return &ManageBackupsUseCase{repo: repo, ssh: ssh}
}

func (uc *ManageBackupsUseCase) CreateSnapshot(req domain.BackupRequest, config domain.ServerConfig) (*domain.SavedBackup, error) {
	if err := uc.ssh.Connect(config); err != nil {
		return nil, fmt.Errorf("falló conexión SSH: %w", err)
	}
	defer func() {
		if clErr := uc.ssh.Close(); clErr != nil {
			slog.Warn("manage_backups: error cerrando sesión SSH en CreateSnapshot", "error", clErr)
		}
	}()

	if _, err := uc.ssh.RunCommand("mkdir -p /opt/tarhiata/backups"); err != nil {
		return nil, fmt.Errorf("falló al crear directorio de backups: %w", err)
	}

	ts := time.Now().Format("20060102_150405")
	dumpCmd, filename, remotePath, engine := uc.buildDumpCommand(req, ts, config.Name)

	res, err := uc.ssh.RunCommand(dumpCmd)
	if err != nil || res == nil || res.ExitCode != 0 {
		out := ""
		if res != nil {
			out = res.Output
		}
		if out == "" && err != nil {
			out = err.Error()
		}
		return nil, fmt.Errorf("error al ejecutar backup: %s", out)
	}

	sizeBytes := uc.getRemoteFileSize(remotePath)
	s3Location := uc.uploadToS3(req, remotePath, filename)

	backup := domain.SavedBackup{
		TargetName: req.TargetName,
		TargetType: req.TargetType,
		Engine:     engine,
		Filename:   filename,
		FilePath:   remotePath,
		SizeBytes:  sizeBytes,
		Status:     "completed",
		S3Location: s3Location,
		CreatedAt:  time.Now().Format("2006-01-02 15:04:05"),
	}

	if err := uc.repo.SaveBackup(backup); err != nil {
		return nil, fmt.Errorf("error al guardar registro de backup en sqlite: %w", err)
	}

	backups, err := uc.repo.GetBackups()
	if err == nil && len(backups) > 0 {
		return &backups[0], nil
	}

	return &backup, nil
}

func (uc *ManageBackupsUseCase) buildDumpCommand(req domain.BackupRequest, ts string, serverName string) (string, string, string, string) {
	if req.TargetType != "database" {
		filename := fmt.Sprintf("backup_vol_%s_%s.tar.gz", req.TargetName, ts)
		remotePath := fmt.Sprintf("/opt/tarhiata/backups/%s", filename)
		dumpCmd := fmt.Sprintf("tar -czf %s -C /opt/data %s 2>/dev/null || true", validator.ShellQuote(remotePath), validator.ShellQuote(req.TargetName))
		return dumpCmd, filename, remotePath, "volume"
	}

	db, err := uc.repo.GetDatabase(req.TargetName, serverName)
	if err != nil || db == nil {
		engineName := strings.TrimSpace(req.Engine)
		if engineName == "" {
			engineName = "postgres"
		}
		db = &domain.SavedDatabase{
			Name:   req.TargetName,
			Engine: engineName,
		}
	}

	quotedDBName := validator.ShellQuote(db.Name)
	containerTarget := fmt.Sprintf("$(CID=$(docker ps -q -f name=tarhiata-db-%s | head -n 1); if [ -z \"$CID\" ]; then CID=$(docker ps -q -f name=%s | head -n 1); fi; echo $CID)", quotedDBName, quotedDBName)
	filename := fmt.Sprintf("backup_%s_%s.sql.gz", db.Name, ts)
	remotePath := fmt.Sprintf("/opt/tarhiata/backups/%s", filename)

	var dumpCmd string
	switch strings.ToLower(db.Engine) {
	case "postgres":
		dumpCmd = fmt.Sprintf("docker exec %s pg_dumpall -U postgres | gzip > %s", containerTarget, validator.ShellQuote(remotePath))
	case "mongo", "mongodb":
		filename = fmt.Sprintf("backup_%s_%s.archive.gz", db.Name, ts)
		remotePath = fmt.Sprintf("/opt/tarhiata/backups/%s", filename)
		dumpCmd = fmt.Sprintf("docker exec %s mongodump --archive --gzip > %s", containerTarget, validator.ShellQuote(remotePath))
	case "mysql", "mariadb":
		pass := db.Password
		if pass == "" {
			pass = "root"
		}
		dumpCmd = fmt.Sprintf("docker exec %s mysqldump --all-databases -u root -p%s | gzip > %s", containerTarget, validator.ShellQuote(pass), validator.ShellQuote(remotePath))
	case "redis":
		filename = fmt.Sprintf("backup_%s_%s.rdb.gz", db.Name, ts)
		remotePath = fmt.Sprintf("/opt/tarhiata/backups/%s", filename)
		dumpCmd = fmt.Sprintf("docker exec %s redis-cli SAVE && cat /data/dump.rdb | gzip > %s", containerTarget, validator.ShellQuote(remotePath))
	default:
		dumpCmd = fmt.Sprintf("docker exec %s pg_dumpall -U postgres | gzip > %s", containerTarget, validator.ShellQuote(remotePath))
	}

	return dumpCmd, filename, remotePath, db.Engine
}

func (uc *ManageBackupsUseCase) getRemoteFileSize(remotePath string) int64 {
	quotedPath := validator.ShellQuote(remotePath)
	sizeRes, errSize := uc.ssh.RunCommand(fmt.Sprintf("stat -c%%s %s 2>/dev/null || wc -c < %s", quotedPath, quotedPath))
	if errSize == nil && sizeRes != nil {
		if parsed, errParse := strconv.ParseInt(strings.TrimSpace(sizeRes.Output), 10, 64); errParse == nil {
			return parsed
		}
	}
	return 0
}

func (uc *ManageBackupsUseCase) uploadToS3(req domain.BackupRequest, remotePath, filename string) string {
	if req.CustomS3URL != "" || req.S3Target == "custom" {
		bucket := req.BucketName
		if bucket == "" {
			bucket = "backups"
		}
		s3Location := fmt.Sprintf("%s/%s/%s", req.CustomS3URL, bucket, filename)
		// Cita cada valor para el shell interno del contenedor y luego envuelve todo el
		// comando interno como un único literal para el shell remoto que lanza "docker run",
		// para que $()/backticks/comillas en la URL o credenciales no se ejecuten en ninguna
		// de las dos capas.
		innerCmd := fmt.Sprintf("mc alias set target %s %s %s 2>/dev/null && mc mb target/%s 2>/dev/null; mc cp /backups/%s target/%s/",
			validator.ShellQuote(req.CustomS3URL), validator.ShellQuote(req.AccessKey), validator.ShellQuote(req.SecretKey),
			validator.ShellQuote(bucket), validator.ShellQuote(filename), validator.ShellQuote(bucket))
		uploadCmd := "docker run --rm -v /opt/tarhiata/backups:/backups minio/mc:latest sh -c " + validator.ShellQuote(innerCmd)
		resUpload, errUpload := uc.ssh.RunCommand(uploadCmd)
		if errUpload != nil || (resUpload != nil && resUpload.ExitCode != 0) {
			slog.Warn("Fallo en subida S3 externo", "error", errUpload)
		}
		if errUpload == nil && (resUpload == nil || resUpload.ExitCode == 0) {
			slog.Info("Snapshot subido exitosamente a S3 Externo", "location", s3Location)
		}
		return s3Location
	}

	if req.S3Target != "" {
		bucket := req.BucketName
		if bucket == "" {
			bucket = "backups"
		}
		cleanMinIO := req.S3Target
		if idx := strings.Index(cleanMinIO, "_"); idx != -1 {
			cleanMinIO = cleanMinIO[:idx]
		}
		cleanMinIO = strings.TrimPrefix(cleanMinIO, "tarhiata-db-")

		quotedMinIOName := validator.ShellQuote(cleanMinIO)
		minioContainer := fmt.Sprintf("$(docker ps -q -f name=tarhiata-db-%s | head -n 1)", quotedMinIOName)
		s3Location := fmt.Sprintf("s3://%s/%s/%s", cleanMinIO, bucket, filename)

		quotedBucket := validator.ShellQuote(bucket)
		quotedRemotePath := validator.ShellQuote(remotePath)
		quotedFilename := validator.ShellQuote(filename)
		uploadCmd := fmt.Sprintf("docker exec %s mc alias set local http://localhost:9000 admin admin_pass 2>/dev/null; docker exec %s mc mb local/%s 2>/dev/null; docker cp %s $(docker ps -q -f name=tarhiata-db-%s | head -n 1):/tmp/%s 2>/dev/null && docker exec %s mc cp /tmp/%s local/%s/ 2>/dev/null",
			minioContainer, minioContainer, quotedBucket, quotedRemotePath, quotedMinIOName, quotedFilename, minioContainer, quotedFilename, quotedBucket)
		resUpload, errUpload := uc.ssh.RunCommand(uploadCmd)
		if errUpload != nil || (resUpload != nil && resUpload.ExitCode != 0) {
			slog.Warn("Fallo en subida MinIO S3", "error", errUpload)
		}
		if errUpload == nil && (resUpload == nil || resUpload.ExitCode == 0) {
			slog.Info("Snapshot respaldado exitosamente en MinIO", "location", s3Location)
		}
		return s3Location
	}

	return ""
}

func (uc *ManageBackupsUseCase) RestoreSnapshot(backupID int, config domain.ServerConfig) error {
	backup, err := uc.repo.GetBackupByID(backupID)
	if err != nil || backup == nil {
		return fmt.Errorf("backup ID %d no encontrado", backupID)
	}

	if err := uc.ssh.Connect(config); err != nil {
		return fmt.Errorf("falló conexión SSH: %w", err)
	}
	defer func() {
		if clErr := uc.ssh.Close(); clErr != nil {
			slog.Warn("manage_backups: error cerrando sesión SSH en RestoreSnapshot", "error", clErr)
		}
	}()

	if backup.TargetType != "database" {
		restoreCmd := fmt.Sprintf("tar -xzf %s -C /opt/data/", validator.ShellQuote(backup.FilePath))
		res, err := uc.ssh.RunCommand(restoreCmd)
		if err != nil || res == nil || res.ExitCode != 0 {
			out := ""
			if res != nil {
				out = res.Output
			}
			if out == "" && err != nil {
				out = err.Error()
			}
			return fmt.Errorf("falló la restauración del volumen: %s", out)
		}
		return nil
	}

	db, err := uc.repo.GetDatabase(backup.TargetName, config.Name)
	if err != nil || db == nil {
		engineName := strings.TrimSpace(backup.Engine)
		if engineName == "" {
			engineName = "postgres"
		}
		db = &domain.SavedDatabase{
			Name:   backup.TargetName,
			Engine: engineName,
		}
	}
	quotedDBName := validator.ShellQuote(db.Name)
	containerTarget := fmt.Sprintf("$(CID=$(docker ps -q -f name=tarhiata-db-%s | head -n 1); if [ -z \"$CID\" ]; then CID=$(docker ps -q -f name=%s | head -n 1); fi; echo $CID)", quotedDBName, quotedDBName)
	quotedFilePath := validator.ShellQuote(backup.FilePath)
	var restoreCmd string

	switch strings.ToLower(db.Engine) {
	case "postgres":
		restoreCmd = fmt.Sprintf("gunzip -c %s | docker exec -i %s psql -U postgres", quotedFilePath, containerTarget)
	case "mongo", "mongodb":
		restoreCmd = fmt.Sprintf("cat %s | docker exec -i %s mongorestore --archive --gzip", quotedFilePath, containerTarget)
	case "mysql", "mariadb":
		pass := db.Password
		if pass == "" {
			pass = "root"
		}
		restoreCmd = fmt.Sprintf("gunzip -c %s | docker exec -i %s mysql -u root -p%s", quotedFilePath, containerTarget, validator.ShellQuote(pass))
	default:
		restoreCmd = fmt.Sprintf("gunzip -c %s | docker exec -i %s psql -U postgres", quotedFilePath, containerTarget)
	}

	res, err := uc.ssh.RunCommand(restoreCmd)
	if err != nil || res == nil || res.ExitCode != 0 {
		out := ""
		if res != nil {
			out = res.Output
		}
		if out == "" && err != nil {
			out = err.Error()
		}
		return fmt.Errorf("falló la restauración de BD: %s", out)
	}
	return nil
}

func (uc *ManageBackupsUseCase) DownloadSnapshot(backupID int, config domain.ServerConfig) (*SnapshotDownloadResult, error) {
	backup, err := uc.repo.GetBackupByID(backupID)
	if err != nil || backup == nil {
		return nil, fmt.Errorf("backup ID %d no encontrado", backupID)
	}

	if err := uc.ssh.Connect(config); err != nil {
		return nil, fmt.Errorf("falló conexión SSH: %w", err)
	}
	defer func() {
		if clErr := uc.ssh.Close(); clErr != nil {
			slog.Warn("manage_backups: error cerrando sesión SSH en DownloadSnapshot", "error", clErr)
		}
	}()

	res, err := uc.ssh.RunCommand(fmt.Sprintf("base64 -w 0 %s", validator.ShellQuote(backup.FilePath)))
	if err != nil || res == nil || res.ExitCode != 0 {
		out := ""
		if res != nil {
			out = res.Output
		}
		if out == "" && err != nil {
			out = err.Error()
		}
		return nil, fmt.Errorf("falló la lectura remota del backup: %s", out)
	}

	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(res.Output))
	if err != nil {
		return nil, fmt.Errorf("error al decodificar datos del backup: %w", err)
	}

	return &SnapshotDownloadResult{
		Data:     data,
		Filename: backup.Filename,
	}, nil
}
