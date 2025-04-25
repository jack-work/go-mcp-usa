package mcp

type Server struct {
	ID  string
	Env *[]string // environment variables passed to the container
}

type DockerServer struct {
	Server
	ImageName     *string // used if container must be created
	ContainerName *string // if not specified and ImageName is specified, a new container will be created with a default name
}

type Client struct {
	Servers map[string]Server
}
