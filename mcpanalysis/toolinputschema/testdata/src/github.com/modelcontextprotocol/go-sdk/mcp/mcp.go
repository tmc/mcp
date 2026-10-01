package mcp

type Tool struct {
	Name        string
	InputSchema any
}
type Server struct{}

func (*Server) AddTool(*Tool, any)                      {}
func AddTool[In, Out any](*Server, *Tool, func(In) Out) {}
