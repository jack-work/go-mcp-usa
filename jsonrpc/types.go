package jsonrpc

type Message[TParams any] struct {
	JSONRPC string  `json:"jsonrpc"`
	ID      string  `json:"id,omitempty"`
	Method  string  `json:"method,omitempty"`
	Params  TParams `json:"params,omitempty"`
	Result  any     `json:"result,omitempty"`
	Error   *Error  `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
