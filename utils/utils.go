package utils

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"
)

type NopCloser struct {
	io.Reader
}

func (NopCloser) Close() error { return nil }

func ClearConsole() {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "cls")
	default:
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	_ = cmd.Run()
}

func FormatDuration(d time.Duration) string {
	min := int(d.Minutes())
	sec := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", min, sec)
}
