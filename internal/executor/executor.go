package executor

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// Exec replaces the current process with the given command,
// injecting the provided env vars into the current environment.
// This uses syscall.Exec so the child process becomes PID 1 in Docker.
func Exec(args []string, envVars map[string]string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	binary, err := exec.LookPath(args[0])
	if err != nil {
		return fmt.Errorf("command not found: %s", args[0])
	}

	// Build environment: current env + injected vars
	environ := os.Environ()
	for k, v := range envVars {
		environ = append(environ, fmt.Sprintf("%s=%s", k, v))
	}

	return syscall.Exec(binary, args, environ) //#nosec G204 -- args are user-provided CLI arguments for process replacement
}
