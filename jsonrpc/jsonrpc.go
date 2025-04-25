package jsonrpc

type Message struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Method  string `json:"method,omitempty"`
	Params  any    `json:"params,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Client-specific types
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ClientCapabilities struct {
	Tools     bool `json:"tools"`
	Prompts   bool `json:"prompts"`
	Resources bool `json:"resources"`
}

type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
	Capabilities    ClientCapabilities `json:"capabilities"`
}
