package runtime

import (
	"context"
	"os/exec"
)

func NewCommand(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	hideWindow(cmd)
	return cmd
}

func NewCommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, arg...)
	hideWindow(cmd)
	return cmd
}
