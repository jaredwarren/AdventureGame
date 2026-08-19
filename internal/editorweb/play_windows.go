//go:build windows

package editorweb

import "os/exec"

func setPlayProcGroup(cmd *exec.Cmd) {}

func killPlayProcGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
