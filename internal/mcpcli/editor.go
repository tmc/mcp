package mcpcli

import (
	"errors"
	"os"
	"os/exec"
)

// EditTempFile opens contents in $EDITOR and returns the edited bytes.
func EditTempFile(contents []byte, suffix string) ([]byte, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return nil, errors.New("$EDITOR is not set")
	}
	file, err := os.CreateTemp("", "mcp-*"+suffix)
	if err != nil {
		return nil, err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(contents); err != nil {
		file.Close()
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	cmd := exec.Command(editor, file.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return os.ReadFile(file.Name())
}
