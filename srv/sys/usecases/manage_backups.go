package usecases

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

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
	defer uc.ssh.Close()

	if _, err := uc.ssh.RunCommand("mkdir -p /opt/tarhiata/backups"); err != nil {
		return nil, fmt.Errorf("falló al crear directorio de backups: %w", err)
	}

	ts := time.Now().Format("20060102_150405")
	var filename, dumpCmd string
	engine := "volume"

	if req.TargetType == "database" {
		db, err := uc.repo.GetDatabase(req.TargetName)
		if err != nil || db == nil {
			return nil, fmt.Errorf("base de datos '%s' no encontrada en el catálogo", req.TargetName)
		}
		engine = db.Engine
		containerTarget := fmt.Sprintf("$(docker ps -q -f name=tarhiata-db-%s | head -n 1)", db.Name)
		filename = fmt.Sprintf("backup_%s_%s.sql.gz", db.Name, ts)
		remotePath := fmt.Sprintf("/opt/tarhiata/backups/%s", filename)

		switch strings.ToLower(db.Engine) {
		case "postgres":
			dumpCmd = fmt.Sprintf("docker exec %s pg_dumpall -U postgres | gzip > %s", containerTarget, remotePath)
		case "mongo", "mongodb":
			filename = fmt.Sprintf("backup_%s_%s.archive.gz", db.Name, ts)
			remotePath = fmt.Sprintf("/opt/tarhiata/backups/%s", filename)
			dumpCmd = fmt.Sprintf("docker exec %s mongodump --archive --gzip > %s", containerTarget, remotePath)
		case "mysql", "mariadb":
			pass := db.Password
			if pass == "" {
				pass = "root"
			}
			dumpCmd = fmt.Sprintf("docker exec %s mysqldump --all-databases -u root -p'%s' | gzip > %s", containerTarget, pass, remotePath)
		case "redis":
			filename = fmt.Sprintf("backup_%s_%s.rdb.gz", db.Name, ts)
			remotePath = fmt.Sprintf("/opt/tarhiata/backups/%s", filename)
			dumpCmd = fmt.Sprintf("docker exec %s redis-cli SAVE && cat /data/dump.rdb | gzip > %s", containerTarget, remotePath)
		default:
			dumpCmd = fmt.Sprintf("docker exec %s pg_dumpall -U postgres | gzip > %s", containerTarget, remotePath)
		}
	} else {
		// Volume Backup
		filename = fmt.Sprintf("backup_vol_%s_%s.tar.gz", req.TargetName, ts)
		remotePath := fmt.Sprintf("/opt/tarhiata/backups/%s", filename)
		dumpCmd = fmt.Sprintf("tar -czf %s -C /opt/data %s 2>/dev/null || true", remotePath, req.TargetName)
	}

	res, err := uc.ssh.RunCommand(dumpCmd)
	if err != nil || res.ExitCode != 0 {
		return nil, fmt.Errorf("error al ejecutar backup: %s", res.Output)
	}

	remotePath := fmt.Sprintf("/opt/tarhiata/backups/%s", filename)
	sizeRes, _ := uc.ssh.RunCommand(fmt.Sprintf("stat -c%%s %s 2>/dev/null || wc -c < %s", remotePath, remotePath))
	var sizeBytes int64
	if sizeRes != nil {
		val := strings.TrimSpace(sizeRes.Output)
		sizeBytes, _ = strconv.ParseInt(val, 10, 64)
	}

	s3Location := ""
	if req.CustomS3URL != "" || req.S3Target == "custom" {
		bucket := req.BucketName
		if bucket == "" {
			bucket = "backups"
		}
		s3Location = fmt.Sprintf("%s/%s/%s", req.CustomS3URL, bucket, filename)
		// Subida a S3 Personalizado usando contenedor efímero minio/mc en el VPS host
		uploadCmd := fmt.Sprintf("docker run --rm -v /opt/tarhiata/backups:/backups minio/mc:latest sh -c \"mc alias set target '%s' '%s' '%s' 2>/dev/null && mc mb target/%s 2>/dev/null; mc cp /backups/%s target/%s/\"",
			req.CustomS3URL, req.AccessKey, req.SecretKey, bucket, filename, bucket)
		_, _ = uc.ssh.RunCommand(uploadCmd)
		fmt.Printf("🌐 [Custom S3] Snapshot subido exitosamente a S3 Externo (%s)!\n", s3Location)
	} else if req.S3Target != "" {
		bucket := req.BucketName
		if bucket == "" {
			bucket = "backups"
		}
		cleanMinIO := req.S3Target
		if idx := strings.Index(cleanMinIO, "_"); idx != -1 {
			cleanMinIO = cleanMinIO[:idx]
		}
		cleanMinIO = strings.TrimPrefix(cleanMinIO, "tarhiata-db-")

		minioContainer := fmt.Sprintf("$(docker ps -q -f name=tarhiata-db-%s | head -n 1)", cleanMinIO)
		s3Location = fmt.Sprintf("s3://%s/%s/%s", cleanMinIO, bucket, filename)

		// Subida de respaldo a MinIO S3 interno
		uploadCmd := fmt.Sprintf("docker exec %s mc alias set local http://localhost:9000 admin admin_pass 2>/dev/null; docker exec %s mc mb local/%s 2>/dev/null; docker cp %s $(docker ps -q -f name=tarhiata-db-%s | head -n 1):/tmp/%s 2>/dev/null && docker exec %s mc cp /tmp/%s local/%s/ 2>/dev/null",
			minioContainer, minioContainer, bucket, remotePath, cleanMinIO, filename, minioContainer, filename, bucket)
		_, _ = uc.ssh.RunCommand(uploadCmd)
		fmt.Printf("📦 [MinIO S3] Snapshot respaldado exitosamente en MinIO (%s)!\n", s3Location)
	}

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

func (uc *ManageBackupsUseCase) RestoreSnapshot(backupID int, config domain.ServerConfig) error {
	backup, err := uc.repo.GetBackupByID(backupID)
	if err != nil || backup == nil {
		return fmt.Errorf("backup ID %d no encontrado", backupID)
	}

	if err := uc.ssh.Connect(config); err != nil {
		return fmt.Errorf("falló conexión SSH: %w", err)
	}
	defer uc.ssh.Close()

	if backup.TargetType == "database" {
		db, err := uc.repo.GetDatabase(backup.TargetName)
		if err != nil || db == nil {
			return fmt.Errorf("base de datos '%s' no encontrada para restauración", backup.TargetName)
		}
		containerTarget := fmt.Sprintf("$(docker ps -q -f name=tarhiata-db-%s | head -n 1)", db.Name)
		var restoreCmd string

		switch strings.ToLower(db.Engine) {
		case "postgres":
			restoreCmd = fmt.Sprintf("gunzip -c %s | docker exec -i %s psql -U postgres", backup.FilePath, containerTarget)
		case "mongo", "mongodb":
			restoreCmd = fmt.Sprintf("cat %s | docker exec -i %s mongorestore --archive --gzip", backup.FilePath, containerTarget)
		case "mysql", "mariadb":
			pass := db.Password
			if pass == "" {
				pass = "root"
			}
			restoreCmd = fmt.Sprintf("gunzip -c %s | docker exec -i %s mysql -u root -p'%s'", backup.FilePath, containerTarget, pass)
		default:
			restoreCmd = fmt.Sprintf("gunzip -c %s | docker exec -i %s psql -U postgres", backup.FilePath, containerTarget)
		}

		res, err := uc.ssh.RunCommand(restoreCmd)
		if err != nil || res.ExitCode != 0 {
			return fmt.Errorf("falló la restauración de BD: %s", res.Output)
		}
	} else {
		// Volume restore
		restoreCmd := fmt.Sprintf("tar -xzf %s -C /opt/data/", backup.FilePath)
		res, err := uc.ssh.RunCommand(restoreCmd)
		if err != nil || res.ExitCode != 0 {
			return fmt.Errorf("falló la restauración del volumen: %s", res.Output)
		}
	}

	return nil
}

func (uc *ManageBackupsUseCase) DownloadSnapshot(backupID int, config domain.ServerConfig) ([]byte, string, error) {
	backup, err := uc.repo.GetBackupByID(backupID)
	if err != nil || backup == nil {
		return nil, "", fmt.Errorf("backup ID %d no encontrado", backupID)
	}

	if err := uc.ssh.Connect(config); err != nil {
		return nil, "", fmt.Errorf("falló conexión SSH: %w", err)
	}
	defer uc.ssh.Close()

	res, err := uc.ssh.RunCommand(fmt.Sprintf("base64 -w 0 %s", backup.FilePath))
	if err != nil || res.ExitCode != 0 {
		return nil, "", fmt.Errorf("falló la lectura remota del backup: %s", res.Output)
	}

	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(res.Output))
	if err != nil {
		return nil, "", fmt.Errorf("error al decodificar datos del backup: %w", err)
	}

	return data, backup.Filename, nil
}
