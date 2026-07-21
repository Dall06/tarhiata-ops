import os
import re

with open("srv/tarhiata/handlers/cli.go", "r") as f:
    lines = f.readlines()

def get_lines(start, end):
    return "".join(lines[start-1:end])

# Move Run back to main.go
run_func = get_lines(19, 108)
main_go = """package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/handlers"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/repositories"
	"github.com/charmbracelet/huh"
)

""" + run_func.replace("func Run() {", "func main() {")

main_go = main_go.replace("serverConfig = handleConfig(repo, serverConfig)", "serverConfig = handlers.NewConfigHandler(repo).Execute(serverConfig)")
main_go = main_go.replace("handleBootstrap(*serverConfig)", "handlers.NewBootstrapHandler(repo).Execute(*serverConfig)")
main_go = main_go.replace("handleServices(*serverConfig, repo)", "handlers.NewServiceHandler(repo).Execute(*serverConfig)")
main_go = main_go.replace("handleDatabases(*serverConfig, repo)", "handlers.NewDatabaseHandler(repo).Execute(*serverConfig)")
main_go = main_go.replace("handleTools(*serverConfig)", "handlers.NewToolHandler(repo).Execute(*serverConfig)")
main_go = main_go.replace("handleShell(*serverConfig)", "handlers.NewShellHandler(repo).Execute(*serverConfig)")

with open("cmd/tarhiata/main.go", "w") as f:
    f.write(main_go)

ports_handlers = """package ports

import "github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"

type ConfigHandler interface {
	Execute(current *domain.ServerConfig) *domain.ServerConfig
}

type BootstrapHandler interface {
	Execute(config domain.ServerConfig)
}

type ServiceHandler interface {
	Execute(config domain.ServerConfig)
}

type DatabaseHandler interface {
	Execute(config domain.ServerConfig)
}

type ToolHandler interface {
	Execute(config domain.ServerConfig)
}

type ShellHandler interface {
	Execute(config domain.ServerConfig)
}
"""
with open("srv/tarhiata/ports/handlers.go", "w") as f:
    f.write(ports_handlers)


def write_handler(filename, struct_name, interface_name, method_name, content):
    content = re.sub(r"func " + method_name + r"\((.*?)\)(.*?){", rf"func (h *{struct_name}) Execute(\1)\2{{", content)
    
    # Change repo. to h.repo.
    content = re.sub(r"(\b)repo\.", r"\1h.repo.", content)
    
    # Replace internal function calls inside the struct
    content = re.sub(r"\b(run[A-Za-z]+|show[A-Za-z]+)\(", r"h.\1(", content)
    content = re.sub(r"func h\.(run[A-Za-z]+|show[A-Za-z]+)\(", rf"func (h *{struct_name}) \1(", content)

    # Remove repo argument in signatures and internal calls
    content = re.sub(r"repo \*repositories\.SQLiteRepository, ", "", content)
    content = re.sub(r", repo \*repositories\.SQLiteRepository", "", content)
    content = re.sub(r"repo \*repositories\.SQLiteRepository", "", content)
    content = re.sub(r"\(repo, ", "(", content)
    content = re.sub(r", repo\)", ")", content)
    content = re.sub(r", repo,", ",", content)
    
    # Change specific Execute signatures
    if "Execute(current *domain.ServerConfig)" in content:
        pass # It's correct for config
    else:
        # Just ensure config is domain.ServerConfig
        content = content.replace("Execute(config domain.ServerConfig)", "Execute(config domain.ServerConfig)")

    file_content = f"""package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/repositories"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/usecases"
	"github.com/charmbracelet/huh"
)

type {struct_name} struct {{
	repo ports.ConfigRepository
}}

func New{interface_name}(repo ports.ConfigRepository) ports.{interface_name} {{
	return &{struct_name}{{repo: repo}}
}}

""" + content

    with open(f"srv/tarhiata/handlers/{filename}.go", "w") as f:
        f.write(file_content)

write_handler("config_handler", "configHandler", "ConfigHandler", "handleConfig", get_lines(110, 240))
write_handler("bootstrap_handler", "bootstrapHandler", "BootstrapHandler", "handleBootstrap", get_lines(242, 327))
write_handler("tool_handler", "toolHandler", "ToolHandler", "handleTools", get_lines(329, 371))
write_handler("shell_handler", "shellHandler", "ShellHandler", "handleShell", get_lines(373, 385))
write_handler("service_handler", "serviceHandler", "ServiceHandler", "handleServices", get_lines(387, 455) + "\n" + get_lines(457, 550) + "\n" + get_lines(552, 650) + "\n" + get_lines(652, 760) + "\n" + get_lines(762, 1053))
write_handler("database_handler", "databaseHandler", "DatabaseHandler", "handleDatabases", get_lines(1055, 1092) + "\n" + get_lines(1094, 1163))

os.remove("srv/tarhiata/handlers/cli.go")
