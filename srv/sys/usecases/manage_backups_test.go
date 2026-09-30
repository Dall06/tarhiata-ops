package usecases

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/pkg/validator"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestCreateSnapshotContainerResolution(t *testing.T) {
	repo := mocks.NewMockConfigRepository()
	repo.Databases = []domain.SavedDatabase{
		{Name: "postgres-main", Engine: "postgres", DeployType: "single-node"},
	}
	mockSSH := mocks.NewMockSSHExecutor()
	mockSSH.MockResponses["docker exec"] = &domain.CommandResult{Output: "success", ExitCode: 0}

	uc := NewManageBackupsUseCase(repo, mockSSH)
	req := domain.BackupRequest{
		TargetName: "postgres-main",
		TargetType: "database",
	}

	cfg := domain.ServerConfig{Host: "127.0.0.1", User: "root"}
	backup, err := uc.CreateSnapshot(req, cfg)
	if err != nil {
		t.Fatalf("expected snapshot success, got error: %v", err)
	}

	if backup == nil || backup.TargetName != "postgres-main" {
		t.Errorf("snapshot returned unexpected backup: %+v", backup)
	}
}

// TestManageBackupsUseCase_ShellInjectionPrevention valida el flujo completo de cada
// punto de inyección de comandos cerrado en manage_backups.go: los valores maliciosos
// deben llegar citados a los comandos reales enviados por SSH.
func TestManageBackupsUseCase_ShellInjectionPrevention(t *testing.T) {
	t.Run("CreateSnapshot de volumen cita TargetName y remotePath en el tar", func(t *testing.T) {
		repo := mocks.NewMockConfigRepository()
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewManageBackupsUseCase(repo, mockSSH)

		const evilName = "vol'; rm -rf / #"
		req := domain.BackupRequest{TargetName: evilName, TargetType: "volume"}
		if _, err := uc.CreateSnapshot(req, domain.ServerConfig{Host: "1.2.3.4"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var tarCmd string
		for _, c := range mockSSH.CommandsExecuted {
			if strings.HasPrefix(c, "tar -czf") {
				tarCmd = c
			}
		}
		if !strings.Contains(tarCmd, "-C /opt/data "+validator.ShellQuote(evilName)) {
			t.Errorf("comando tar inseguro: %s", tarCmd)
		}
	})

	t.Run("CreateSnapshot mysql cita el password en el dump", func(t *testing.T) {
		repo := mocks.NewMockConfigRepository()
		repo.Databases = []domain.SavedDatabase{{Name: "shop-db", Engine: "mysql", Password: `p' ; rm -rf / #`}}
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewManageBackupsUseCase(repo, mockSSH)

		req := domain.BackupRequest{TargetName: "shop-db", TargetType: "database"}
		if _, err := uc.CreateSnapshot(req, domain.ServerConfig{Host: "1.2.3.4"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var dumpCmd string
		for _, c := range mockSSH.CommandsExecuted {
			if strings.Contains(c, "mysqldump") {
				dumpCmd = c
			}
		}
		want := "-p" + validator.ShellQuote(`p' ; rm -rf / #`)
		if !strings.Contains(dumpCmd, want) {
			t.Errorf("comando mysqldump inseguro:\n got:  %s\n want fragment: %s", dumpCmd, want)
		}
	})

	t.Run("RestoreSnapshot cita un FilePath malicioso persistido en el catálogo", func(t *testing.T) {
		repo := mocks.NewMockConfigRepository()
		const evilPath = `/opt/tarhiata/backups/x'; rm -rf / #.sql.gz`
		repo.Backups = []domain.SavedBackup{{ID: 1, TargetType: "volume", FilePath: evilPath}}
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewManageBackupsUseCase(repo, mockSSH)

		if err := uc.RestoreSnapshot(1, domain.ServerConfig{Host: "1.2.3.4"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var got string
		for _, c := range mockSSH.CommandsExecuted {
			if strings.HasPrefix(c, "tar -xzf") {
				got = c
			}
		}
		want := "tar -xzf " + validator.ShellQuote(evilPath) + " -C /opt/data/"
		if got != want {
			t.Errorf("comando de restauración inseguro:\n got:  %s\n want: %s", got, want)
		}
	})

	t.Run("DownloadSnapshot cita un FilePath malicioso persistido en el catálogo", func(t *testing.T) {
		repo := mocks.NewMockConfigRepository()
		const evilPath = `/opt/tarhiata/backups/x'; rm -rf / #.sql.gz`
		repo.Backups = []domain.SavedBackup{{ID: 1, FilePath: evilPath, Filename: "x.sql.gz"}}
		mockSSH := mocks.NewMockSSHExecutor()
		mockSSH.MockResponses["base64"] = &domain.CommandResult{Output: "aGVsbG8=", ExitCode: 0}
		uc := NewManageBackupsUseCase(repo, mockSSH)

		if _, err := uc.DownloadSnapshot(1, domain.ServerConfig{Host: "1.2.3.4"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var got string
		for _, c := range mockSSH.CommandsExecuted {
			if strings.HasPrefix(c, "base64") {
				got = c
			}
		}
		want := "base64 -w 0 " + validator.ShellQuote(evilPath)
		if got != want {
			t.Errorf("comando de descarga inseguro:\n got:  %s\n want: %s", got, want)
		}
	})

	t.Run("uploadToS3 externo cita URL/credenciales/bucket anidado para ambas capas de shell", func(t *testing.T) {
		repo := mocks.NewMockConfigRepository()
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewManageBackupsUseCase(repo, mockSSH)

		req := domain.BackupRequest{
			TargetType:  "volume",
			TargetName:  "shop",
			CustomS3URL: `https://s3.example.com"; rm -rf / #`,
			AccessKey:   `AK' ; rm -rf / #`,
			SecretKey:   `SK' ; rm -rf / #`,
			BucketName:  "backups",
		}
		backup, err := uc.CreateSnapshot(req, domain.ServerConfig{Host: "1.2.3.4"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var got string
		for _, c := range mockSSH.CommandsExecuted {
			if strings.HasPrefix(c, "docker run --rm") {
				got = c
			}
		}

		innerCmd := fmt.Sprintf("mc alias set target %s %s %s 2>/dev/null && mc mb target/%s 2>/dev/null; mc cp /backups/%s target/%s/",
			validator.ShellQuote(req.CustomS3URL), validator.ShellQuote(req.AccessKey), validator.ShellQuote(req.SecretKey),
			validator.ShellQuote(req.BucketName), validator.ShellQuote(backup.Filename), validator.ShellQuote(req.BucketName))
		want := "docker run --rm -v /opt/tarhiata/backups:/backups minio/mc:latest sh -c " + validator.ShellQuote(innerCmd)

		if got != want {
			t.Errorf("comando de subida S3 inseguro:\n got:  %s\n want: %s", got, want)
		}
	})
}
