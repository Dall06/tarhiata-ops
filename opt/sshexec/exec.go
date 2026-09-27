package sshexec

import (
	"fmt"
	"os/exec"

	"github.com/Dall06/tarhiata-ops/pkg/secutil"
	"github.com/Dall06/tarhiata-ops/pkg/sshclient"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

// Run ejecuta un comando remotamente si el host es remoto o localmente si el host es local.
func Run(config domain.ServerConfig, command string) (domain.CommandResult, error) {
	if secutil.IsLocalHost(config.Host, config.CloudProvider) {
		return RunLocal(command)
	}
	return RunSSH(config, command)
}

// RunLocal ejecuta un comando directamente en el intérprete de comandos del sistema operativo local.
func RunLocal(command string) (domain.CommandResult, error) {
	cmd := exec.Command("sh", "-c", command)
	outBytes, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return domain.CommandResult{
		Output:   string(outBytes),
		ExitCode: exitCode,
		Error:    err,
	}, nil
}

// RunSSH ejecuta un comando en un host remoto utilizando el pool de conexiones SSH seguras.
func RunSSH(config domain.ServerConfig, command string) (domain.CommandResult, error) {
	port := config.Port
	if port == 0 {
		port = 22
	}
	user := config.User
	if user == "" {
		user = "root"
	}

	client, err := sshclient.GlobalPool.Get(config.Host, user, config.PrivateKey, port)
	if err != nil {
		return domain.CommandResult{ExitCode: 1, Error: err}, fmt.Errorf("sshexec: error de conexión SSH con %s: %w", config.Host, err)
	}

	out, exitCode, errRun := client.RunCommand(command)
	if errRun != nil {
		return domain.CommandResult{
			Output:   out,
			ExitCode: exitCode,
			Error:    errRun,
		}, errRun
	}

	return domain.CommandResult{
		Output:   out,
		ExitCode: exitCode,
	}, nil
}
