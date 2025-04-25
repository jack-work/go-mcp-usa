package servers

import (
	"go-mcp-usa/mcp"
)

var imageName = "mcp/brave-search"
var Brave = mcp.DockerServer{
	ImageName: &imageName,
	Server: mcp.Server{
		Env: &[]string{"BRAVE_API_KEY"},
	},
}
