package main

import "go-mcp-usa/mcp"

var braveImageName = "mcp/brave-search"
var Brave = DockerServer{
	ImageName: &braveImageName,
	Server: mcp.Server{
		Env: &[]string{"BRAVE_API_KEY"},
	},
}

// var imageName = "mcp/brave-search"
// var Brave = DockerServer{
// 	ImageName: &imageName,
// 	Server: mcp.Server{
// 		Env: &[]string{"BRAVE_API_KEY"},
// 	},
// }
