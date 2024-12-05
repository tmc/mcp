package mcp

import (
	"context"
	"io"
)

// Transport handles MCP protocol communication.
type Transport interface {
	io.ReadWriteCloser
}

// Optional transport extensions
type (
	Pinger interface {
		Ping(ctx context.Context) error
	}

	Statuser interface {
		Status() error
	}
)

// Handler processes MCP messages.
type Handler interface {
	Handle(ctx context.Context, msg []byte) ([]byte, error)
}

// NotificationHandler processes one-way messages.
type NotificationHandler interface {
	HandleNotification(ctx context.Context, msg []byte) error
}

// Resource represents an MCP resource.
type Resource interface {
	URI() string
	Read(ctx context.Context) ([]byte, error)
}

// Optional resource extensions
type Subscribable interface {
	Subscribe(ctx context.Context) (<-chan []byte, error)
}

// Tool executes an MCP tool operation.
type Tool interface {
	Run(ctx context.Context, args []byte) ([]byte, error)
}

// Optional tool extensions
type Validator interface {
	Validate(args []byte) error
}
