package mcp

type Server struct {
	ImageName   *string
	ContainerId *string
	Env         *[]string
}

type Client struct {
	Servers map[string]Server
}
