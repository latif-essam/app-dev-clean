package cli

import "fmt"

// Run is the CLI entrypoint. Returns a process exit code.
func Run(args []string, version string) int {
	if len(args) == 1 && (args[0] == "--version" || args[0] == "-v") {
		fmt.Println("app-dev-clean", version)
		return 0
	}
	fmt.Println("app-dev-clean", version, "(not yet implemented)")
	return 0
}
