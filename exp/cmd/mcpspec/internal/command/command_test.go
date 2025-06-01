package command

import (
	"bytes"
	"context"
	"testing"
)

// MockCommand implements the Command interface for testing purposes.
type MockCommand struct {
	BaseCommand
	ExecuteFunc func(ctx context.Context, args []string) error
}

// Execute calls the mock's ExecuteFunc.
func (m *MockCommand) Execute(ctx context.Context, args []string) error {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, args)
	}
	return nil
}

func TestBaseCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   BaseCommand
		wantName  string
		wantUsage string
	}{
		{
			name: "basic command",
			command: BaseCommand{
				CommandName: "test",
				UsageText:   "test usage",
			},
			wantName:  "test",
			wantUsage: "test usage",
		},
		{
			name: "send command",
			command: BaseCommand{
				CommandName: "send",
				UsageText:   "send [options] message",
			},
			wantName:  "send",
			wantUsage: "send [options] message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.command.Name(); got != tt.wantName {
				t.Errorf("BaseCommand.Name() = %v, want %v", got, tt.wantName)
			}
			if got := tt.command.Usage(); got != tt.wantUsage {
				t.Errorf("BaseCommand.Usage() = %v, want %v", got, tt.wantUsage)
			}
		})
	}
}

func TestMockCommand_Execute(t *testing.T) {
	called := false
	cmd := &MockCommand{
		BaseCommand: BaseCommand{
			CommandName: "mock",
			UsageText:   "mock command",
			Input:       &bytes.Buffer{},
			Output:      &bytes.Buffer{},
			Error:       &bytes.Buffer{},
		},
		ExecuteFunc: func(ctx context.Context, args []string) error {
			called = true
			return nil
		},
	}

	err := cmd.Execute(context.Background(), []string{})
	if err != nil {
		t.Errorf("MockCommand.Execute() error = %v", err)
	}

	if !called {
		t.Error("MockCommand.Execute() did not call ExecuteFunc")
	}
}
