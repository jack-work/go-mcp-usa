package mcp

type Server struct {
	ID  string
	Env *[]string // environment variables passed to the container
}

func (server *Server) Setup() error {
	return nil
}

type Client struct {
	Servers map[string]Server
}
