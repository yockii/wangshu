//go:build !windows

package runtime

import "os/exec"

func hideWindow(cmd *exec.Cmd) {
}
