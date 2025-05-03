package mcp

import (
	"go-mcp-usa/jsonrpc"
)

type Server interface {
	GetID() string
	GetEnv() *[]string
	Setup() error
}

type Client struct {
	TargetServer Server
	jsonrpc.Client
}
