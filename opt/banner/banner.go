package banner

import "fmt"

// PrintServerBanner muestra la salida de consola oficial al iniciar Tarhiata.
func PrintServerBanner(port int) {
	fmt.Printf("\n\033[1;36m========================================================\033[0m\n")
	fmt.Printf("\033[1;32m 🚀 TARHIATA-OPS WEB CONTROL PLANE OPERACIONAL! \033[0m\n")
	fmt.Printf(" 🌐 Web Dashboard: \033[1;34mhttp://localhost:%d\033[0m\n", port)
	fmt.Printf(" ⌨️  Aesthetic:     \033[1;33mTerminal Dark Vercel Style (⌘K Enabled)\033[0m\n")
	fmt.Printf("\033[1;36m========================================================\033[0m\n\n")
}
