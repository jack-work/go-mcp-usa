package main

import "go-mcp-usa/mcp"

var imageName = "mcp/brave-search"
var Brave = DockerServer{
	ImageName: &imageName,
	Server: mcp.Server{
		Env: &[]string{"BRAVE_API_KEY"},
	},
}
