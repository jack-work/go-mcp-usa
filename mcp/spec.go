package mcp

// Annotations provides optional annotations for the client
type Annotations struct {
	Audience []Role   `json:"audience,omitempty"` // Describes who the intended customer of this object or data is
	Priority *float64 `json:"priority,omitempty"` // Describes how important this data is for operating the server (0-1)
}

// AudioContent represents audio provided to or from an LLM
type AudioContent struct {
	Annotations *Annotations `json:"annotations,omitempty"` // Optional annotations for the client
	Data        string       `json:"data"`                  // The base64-encoded audio data
	MimeType    string       `json:"mimeType"`              // The MIME type of the audio
	Type        string       `json:"type"`                  // Must be "audio"
}

// BlobResourceContents represents binary resource content
type BlobResourceContents struct {
	Blob     string  `json:"blob"`               // A base64-encoded string representing the binary data
	MimeType *string `json:"mimeType,omitempty"` // The MIME type of this resource, if known
	URI      string  `json:"uri"`                // The URI of this resource
}

// CallToolRequest is used by the client to invoke a tool provided by the server
type CallToolRequest struct {
	Method string                `json:"method"` // Must be "tools/call"
	Params CallToolRequestParams `json:"params"`
}

// CallToolRequestParams contains the parameters for a tool call request
type CallToolRequestParams struct {
	Arguments map[string]any `json:"arguments,omitempty"` // The arguments to pass to the tool
	Name      string         `json:"name"`                // The name of the tool to call
}

// CallToolResult is the server's response to a tool call
type CallToolResult struct {
	Meta    map[string]any `json:"_meta,omitempty"`   // Additional metadata
	Content []any          `json:"content"`           // Content can be TextContent, ImageContent, AudioContent, or EmbeddedResource
	IsError bool           `json:"isError,omitempty"` // Whether the tool call ended in an error
}

// CancelledNotification can be sent by either side to indicate cancelling a request
type CancelledNotification struct {
	Method string                      `json:"method"` // Must be "notifications/cancelled"
	Params CancelledNotificationParams `json:"params"`
}

// CancelledNotificationParams contains the parameters for a cancelled notification
type CancelledNotificationParams struct {
	Reason    *string   `json:"reason,omitempty"` // An optional reason for cancellation
	RequestID RequestID `json:"requestId"`        // The ID of the request to cancel
}

// ClientCapabilities describes capabilities a client may support
type ClientCapabilities struct {
	Experimental map[string]map[string]any `json:"experimental,omitempty"` // Experimental capabilities
	Roots        *RootsCapability          `json:"roots,omitempty"`        // Present if client supports listing roots
	Sampling     map[string]any            `json:"sampling,omitempty"`     // Present if client supports sampling from an LLM
}

// RootsCapability represents the roots capability settings
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"` // Whether the client supports notifications for changes to the roots list
}

// CompleteRequest is a request from the client to the server for completion options
type CompleteRequest struct {
	Method string                `json:"method"` // Must be "completion/complete"
	Params CompleteRequestParams `json:"params"`
}

// CompleteRequestParams contains the parameters for a complete request
type CompleteRequestParams struct {
	Argument CompleteArgument `json:"argument"` // The argument's information
	Ref      any              `json:"ref"`      // Either PromptReference or ResourceReference
}

// CompleteArgument represents completion argument information
type CompleteArgument struct {
	Name  string `json:"name"`  // The name of the argument
	Value string `json:"value"` // The value to use for completion matching
}

// CompleteResult is the server's response to a completion/complete request
type CompleteResult struct {
	Meta       map[string]any `json:"_meta,omitempty"` // Additional metadata
	Completion CompletionInfo `json:"completion"`      // Completion information
}

// CompletionInfo contains completion details
type CompletionInfo struct {
	HasMore bool     `json:"hasMore,omitempty"` // Indicates if there are additional options
	Total   *int     `json:"total,omitempty"`   // The total number of options available
	Values  []string `json:"values"`            // An array of completion values
}

// CreateMessageRequest is a request from the server to sample an LLM via the client
type CreateMessageRequest struct {
	Method string                     `json:"method"` // Must be "sampling/createMessage"
	Params CreateMessageRequestParams `json:"params"`
}

// CreateMessageRequestParams contains parameters for create message request
type CreateMessageRequestParams struct {
	IncludeContext   *string           `json:"includeContext,omitempty"`   // Whether to include context
	MaxTokens        int               `json:"maxTokens"`                  // Maximum tokens to sample
	Messages         []SamplingMessage `json:"messages"`                   // The messages to sample from
	Metadata         map[string]any    `json:"metadata,omitempty"`         // Optional provider-specific metadata
	ModelPreferences *ModelPreferences `json:"modelPreferences,omitempty"` // Model selection preferences
	StopSequences    []string          `json:"stopSequences,omitempty"`    // Sequences that should stop sampling
	SystemPrompt     *string           `json:"systemPrompt,omitempty"`     // Optional system prompt
	Temperature      *float64          `json:"temperature,omitempty"`      // Temperature for sampling
}

// CreateMessageResult is the client's response to a sampling/create_message request
type CreateMessageResult struct {
	Meta       map[string]any `json:"_meta,omitempty"`      // Additional metadata
	Content    any            `json:"content"`              // Can be TextContent, ImageContent, or AudioContent
	Model      string         `json:"model"`                // The name of the model that generated the message
	Role       Role           `json:"role"`                 // The role of the message (assistant)
	StopReason *string        `json:"stopReason,omitempty"` // The reason why sampling stopped
}

// Cursor is an opaque token used for pagination
type Cursor string

// EmbeddedResource represents the contents of a resource embedded into a prompt or result
type EmbeddedResource struct {
	Annotations *Annotations `json:"annotations,omitempty"` // Optional annotations for the client
	Resource    any          `json:"resource"`              // Either TextResourceContents or BlobResourceContents
	Type        string       `json:"type"`                  // Must be "resource"
}

// GetPromptRequest is used by the client to get a prompt from the server
type GetPromptRequest struct {
	Method string                 `json:"method"` // Must be "prompts/get"
	Params GetPromptRequestParams `json:"params"`
}

// GetPromptRequestParams contains parameters for the get prompt request
type GetPromptRequestParams struct {
	Arguments map[string]string `json:"arguments,omitempty"` // Arguments for templating
	Name      string            `json:"name"`                // The prompt name
}

// GetPromptResult is the server's response to a prompts/get request
type GetPromptResult struct {
	Meta        map[string]any  `json:"_meta,omitempty"`       // Additional metadata
	Description *string         `json:"description,omitempty"` // Optional description
	Messages    []PromptMessage `json:"messages"`              // The prompt messages
}

// ImageContent represents an image provided to or from an LLM
type ImageContent struct {
	Annotations *Annotations `json:"annotations,omitempty"` // Optional annotations for the client
	Data        string       `json:"data"`                  // The base64-encoded image data
	MimeType    string       `json:"mimeType"`              // The MIME type of the image
	Type        string       `json:"type"`                  // Must be "image"
}

// Implementation describes the name and version of an MCP implementation
type Implementation struct {
	Name    string `json:"name"`    // Name of the implementation
	Version string `json:"version"` // Version of the implementation
}

// InitializeRequest is sent from client to server when it first connects
type InitializeRequest struct {
	Method string                  `json:"method"` // Must be "initialize"
	Params InitializeRequestParams `json:"params"`
}

// InitializeRequestParams contains the parameters for an initialize request
type InitializeRequestParams struct {
	Capabilities    ClientCapabilities `json:"capabilities"`    // Client capabilities
	ClientInfo      Implementation     `json:"clientInfo"`      // Client implementation info
	ProtocolVersion string             `json:"protocolVersion"` // Supported protocol version
}

// InitializeResult is sent from server to client after receiving initialize request
type InitializeResult struct {
	Meta            map[string]any     `json:"_meta,omitempty"`        // Additional metadata
	Capabilities    ServerCapabilities `json:"capabilities"`           // Server capabilities
	Instructions    *string            `json:"instructions,omitempty"` // Optional usage instructions
	ProtocolVersion string             `json:"protocolVersion"`        // Protocol version to use
	ServerInfo      Implementation     `json:"serverInfo"`             // Server implementation info
}

// InitializedNotification is sent from client to server after initialization
type InitializedNotification struct {
	Method string         `json:"method"` // Must be "notifications/initialized"
	Params map[string]any `json:"params,omitempty"`
}

// ListPromptsRequest is sent from client to get a list of available prompts
type ListPromptsRequest struct {
	Method string                    `json:"method"` // Must be "prompts/list"
	Params *ListPromptsRequestParams `json:"params,omitempty"`
}

// ListPromptsRequestParams contains parameters for list prompts request
type ListPromptsRequestParams struct {
	Cursor *string `json:"cursor,omitempty"` // Pagination cursor
}

// ListPromptsResult is the server's response to a prompts/list request
type ListPromptsResult struct {
	Meta       map[string]any `json:"_meta,omitempty"`      // Additional metadata
	NextCursor *string        `json:"nextCursor,omitempty"` // Pagination token for next page
	Prompts    []Prompt       `json:"prompts"`              // Available prompts
}

// ListResourceTemplatesRequest requests available resource templates
type ListResourceTemplatesRequest struct {
	Method string                              `json:"method"` // Must be "resources/templates/list"
	Params *ListResourceTemplatesRequestParams `json:"params,omitempty"`
}

// ListResourceTemplatesRequestParams contains parameters for the request
type ListResourceTemplatesRequestParams struct {
	Cursor *string `json:"cursor,omitempty"` // Pagination cursor
}

// ListResourceTemplatesResult is the server's response
type ListResourceTemplatesResult struct {
	Meta              map[string]any     `json:"_meta,omitempty"`      // Additional metadata
	NextCursor        *string            `json:"nextCursor,omitempty"` // Pagination token for next page
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`    // Available resource templates
}

// ListResourcesRequest requests available resources
type ListResourcesRequest struct {
	Method string                      `json:"method"` // Must be "resources/list"
	Params *ListResourcesRequestParams `json:"params,omitempty"`
}

// ListResourcesRequestParams contains parameters for list resources request
type ListResourcesRequestParams struct {
	Cursor *string `json:"cursor,omitempty"` // Pagination cursor
}

// ListResourcesResult is the server's response
type ListResourcesResult struct {
	Meta       map[string]any `json:"_meta,omitempty"`      // Additional metadata
	NextCursor *string        `json:"nextCursor,omitempty"` // Pagination token for next page
	Resources  []Resource     `json:"resources"`            // Available resources
}

// ListRootsRequest requests a list of root URIs from the client
type ListRootsRequest struct {
	Method string         `json:"method"` // Must be "roots/list"
	Params map[string]any `json:"params,omitempty"`
}

// ListRootsResult is the client's response
type ListRootsResult struct {
	Meta  map[string]any `json:"_meta,omitempty"` // Additional metadata
	Roots []Root         `json:"roots"`           // Available roots
}

// ListToolsRequest requests available tools
type ListToolsRequest struct {
	Method string                  `json:"method"` // Must be "tools/list"
	Params *ListToolsRequestParams `json:"params,omitempty"`
}

// ListToolsRequestParams contains parameters for list tools request
type ListToolsRequestParams struct {
	Cursor *string `json:"cursor,omitempty"` // Pagination cursor
}

// ListToolsResult is the server's response
type ListToolsResult struct {
	Meta       map[string]any `json:"_meta,omitempty"`      // Additional metadata
	NextCursor *string        `json:"nextCursor,omitempty"` // Pagination token for next page
	Tools      []Tool         `json:"tools"`                // Available tools
}

// LoggingLevel represents the severity of a log message
type LoggingLevel string

const (
	LoggingLevelAlert     LoggingLevel = "alert"
	LoggingLevelCritical  LoggingLevel = "critical"
	LoggingLevelDebug     LoggingLevel = "debug"
	LoggingLevelEmergency LoggingLevel = "emergency"
	LoggingLevelError     LoggingLevel = "error"
	LoggingLevelInfo      LoggingLevel = "info"
	LoggingLevelNotice    LoggingLevel = "notice"
	LoggingLevelWarning   LoggingLevel = "warning"
)

// LoggingMessageNotification is a log message from server to client
type LoggingMessageNotification struct {
	Method string                           `json:"method"` // Must be "notifications/message"
	Params LoggingMessageNotificationParams `json:"params"`
}

// LoggingMessageNotificationParams contains parameters for logging notification
type LoggingMessageNotificationParams struct {
	Data   any          `json:"data"`             // The data to log
	Level  LoggingLevel `json:"level"`            // The severity level
	Logger *string      `json:"logger,omitempty"` // Optional logger name
}

// ModelHint provides hints for model selection
type ModelHint struct {
	Name string `json:"name,omitempty"` // A hint for a model name
}

// ModelPreferences describes server preferences for model selection during sampling
type ModelPreferences struct {
	CostPriority         *float64    `json:"costPriority,omitempty"`         // Priority for cost (0-1)
	Hints                []ModelHint `json:"hints,omitempty"`                // Optional model selection hints
	IntelligencePriority *float64    `json:"intelligencePriority,omitempty"` // Priority for intelligence (0-1)
	SpeedPriority        *float64    `json:"speedPriority,omitempty"`        // Priority for speed (0-1)
}

// PingRequest checks that the other party is still alive
type PingRequest struct {
	Method string         `json:"method"` // Must be "ping"
	Params map[string]any `json:"params,omitempty"`
}

// ProgressNotification informs of progress for a long-running request
type ProgressNotification struct {
	Method string                     `json:"method"` // Must be "notifications/progress"
	Params ProgressNotificationParams `json:"params"`
}

// ProgressNotificationParams contains parameters for progress notification
type ProgressNotificationParams struct {
	Message       *string       `json:"message,omitempty"` // Optional message
	Progress      float64       `json:"progress"`          // Current progress
	ProgressToken ProgressToken `json:"progressToken"`     // Token from original request
	Total         *float64      `json:"total,omitempty"`   // Total progress required, if known
}

// ProgressToken is used to associate progress notifications with original request
type ProgressToken any // string or integer

// Prompt represents a prompt or prompt template that the server offers
type Prompt struct {
	Arguments   []PromptArgument `json:"arguments,omitempty"`   // Arguments for templating
	Description *string          `json:"description,omitempty"` // Optional description
	Name        string           `json:"name"`                  // The prompt name
}

// PromptArgument describes an argument a prompt can accept
type PromptArgument struct {
	Description *string `json:"description,omitempty"` // Human-readable description
	Name        string  `json:"name"`                  // Argument name
	Required    *bool   `json:"required,omitempty"`    // Whether this argument is required
}

// PromptListChangedNotification informs that available prompts changed
type PromptListChangedNotification struct {
	Method string         `json:"method"` // Must be "notifications/prompts/list_changed"
	Params map[string]any `json:"params,omitempty"`
}

// PromptMessage describes a message returned as part of a prompt
type PromptMessage struct {
	Content any  `json:"content"` // TextContent, ImageContent, AudioContent, or EmbeddedResource
	Role    Role `json:"role"`    // Message role
}

// PromptReference identifies a prompt
type PromptReference struct {
	Name string `json:"name"` // The prompt name
	Type string `json:"type"` // Must be "ref/prompt"
}

// ReadResourceRequest reads a specific resource URI
type ReadResourceRequest struct {
	Method string                    `json:"method"` // Must be "resources/read"
	Params ReadResourceRequestParams `json:"params"`
}

// ReadResourceRequestParams contains parameters for read resource request
type ReadResourceRequestParams struct {
	URI string `json:"uri"` // The URI to read
}

// ReadResourceResult is the server's response
type ReadResourceResult struct {
	Meta     map[string]any `json:"_meta,omitempty"` // Additional metadata
	Contents []any          `json:"contents"`        // Either TextResourceContents or BlobResourceContents
}

// RequestID uniquely identifies a request in JSON-RPC
type RequestID any // string or integer

// Resource represents a known resource the server can read
type Resource struct {
	Annotations *Annotations `json:"annotations,omitempty"` // Optional annotations
	Description *string      `json:"description,omitempty"` // Optional description
	MimeType    *string      `json:"mimeType,omitempty"`    // Optional MIME type
	Name        string       `json:"name"`                  // Human-readable name
	Size        *int         `json:"size,omitempty"`        // Optional size in bytes
	URI         string       `json:"uri"`                   // Resource URI
}

// ResourceListChangedNotification informs that available resources changed
type ResourceListChangedNotification struct {
	Method string         `json:"method"` // Must be "notifications/resources/list_changed"
	Params map[string]any `json:"params,omitempty"`
}

// ResourceReference references a resource or template
type ResourceReference struct {
	Type string `json:"type"` // Must be "ref/resource"
	URI  string `json:"uri"`  // The URI or template
}

// ResourceTemplate describes a template for resources
type ResourceTemplate struct {
	Annotations *Annotations `json:"annotations,omitempty"` // Optional annotations
	Description *string      `json:"description,omitempty"` // Optional description
	MimeType    *string      `json:"mimeType,omitempty"`    // Optional MIME type
	Name        string       `json:"name"`                  // Human-readable name
	URITemplate string       `json:"uriTemplate"`           // URI template
}

// ResourceUpdatedNotification informs that a resource has changed
type ResourceUpdatedNotification struct {
	Method string                            `json:"method"` // Must be "notifications/resources/updated"
	Params ResourceUpdatedNotificationParams `json:"params"`
}

// ResourceUpdatedNotificationParams contains parameters for resource update notification
type ResourceUpdatedNotificationParams struct {
	URI string `json:"uri"` // The updated resource URI
}

// Role represents the sender or recipient in a conversation
type Role string

const (
	RoleAssistant Role = "assistant"
	RoleUser      Role = "user"
)

// Root represents a root directory or file the server can operate on
type Root struct {
	Name *string `json:"name,omitempty"` // Optional human-readable name
	URI  string  `json:"uri"`            // The root URI (must start with file://)
}

// RootsListChangedNotification informs that the list of roots has changed
type RootsListChangedNotification struct {
	Method string         `json:"method"` // Must be "notifications/roots/list_changed"
	Params map[string]any `json:"params,omitempty"`
}

// SamplingMessage describes a message issued to or from an LLM API
type SamplingMessage struct {
	Content any  `json:"content"` // TextContent, ImageContent, or AudioContent
	Role    Role `json:"role"`    // Message role
}

// ServerCapabilities describes capabilities a server may support
type ServerCapabilities struct {
	Completions  map[string]any            `json:"completions,omitempty"`  // Present if server supports completions
	Experimental map[string]map[string]any `json:"experimental,omitempty"` // Experimental capabilities
	Logging      map[string]any            `json:"logging,omitempty"`      // Present if server supports logging
	Prompts      *PromptsCapability        `json:"prompts,omitempty"`      // Present if server offers prompts
	Resources    *ResourcesCapability      `json:"resources,omitempty"`    // Present if server offers resources
	Tools        *ToolsCapability          `json:"tools,omitempty"`        // Present if server offers tools
}

// PromptsCapability represents prompt capability settings
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"` // Whether server supports notifications for prompt list changes
}

// ResourcesCapability represents resource capability settings
type ResourcesCapability struct {
	ListChanged bool `json:"listChanged,omitempty"` // Whether server supports notifications for resource list changes
	Subscribe   bool `json:"subscribe,omitempty"`   // Whether server supports subscribing to resource updates
}

// ToolsCapability represents tool capability settings
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"` // Whether server supports notifications for tool list changes
}

// SetLevelRequest enables or adjusts logging
type SetLevelRequest struct {
	Method string                `json:"method"` // Must be "logging/setLevel"
	Params SetLevelRequestParams `json:"params"`
}

// SetLevelRequestParams contains parameters for set level request
type SetLevelRequestParams struct {
	Level LoggingLevel `json:"level"` // Desired logging level
}

// SubscribeRequest requests resource update notifications
type SubscribeRequest struct {
	Method string                 `json:"method"` // Must be "resources/subscribe"
	Params SubscribeRequestParams `json:"params"`
}

// SubscribeRequestParams contains parameters for subscribe request
type SubscribeRequestParams struct {
	URI string `json:"uri"` // Resource URI to subscribe to
}

// TextContent represents text provided to or from an LLM
type TextContent struct {
	Annotations *Annotations `json:"annotations,omitempty"` // Optional annotations
	Text        string       `json:"text"`                  // The text content
	Type        string       `json:"type"`                  // Must be "text"
}

// TextResourceContents represents text resource content
type TextResourceContents struct {
	MimeType *string `json:"mimeType,omitempty"` // Optional MIME type
	Text     string  `json:"text"`               // The text content
	URI      string  `json:"uri"`                // Resource URI
}

// Tool defines a tool the client can call
type Tool struct {
	Annotations *ToolAnnotations `json:"annotations,omitempty"` // Optional tool info
	Description *string          `json:"description,omitempty"` // Optional description
	InputSchema ToolInputSchema  `json:"inputSchema"`           // JSON Schema for parameters
	Name        string           `json:"name"`                  // Tool name
}

// ToolAnnotations provides additional tool information
type ToolAnnotations struct {
	DestructiveHint *bool   `json:"destructiveHint,omitempty"` // If true, may perform destructive updates
	IdempotentHint  *bool   `json:"idempotentHint,omitempty"`  // If true, repeated calls have no additional effect
	OpenWorldHint   *bool   `json:"openWorldHint,omitempty"`   // If true, may interact with external entities
	ReadOnlyHint    *bool   `json:"readOnlyHint,omitempty"`    // If true, doesn't modify environment
	Title           *string `json:"title,omitempty"`           // Human-readable title
}

// ToolInputSchema defines the expected parameters for a tool
type ToolInputSchema struct {
	Properties map[string]map[string]any `json:"properties,omitempty"` // Parameter properties
	Required   []string                  `json:"required,omitempty"`   // Required parameters
	Type       string                    `json:"type"`                 // Must be "object"
}

// ToolListChangedNotification informs that available tools changed
type ToolListChangedNotification struct {
	Method string         `json:"method"` // Must be "notifications/tools/list_changed"
	Params map[string]any `json:"params,omitempty"`
}

// UnsubscribeRequest cancels resource update notifications
type UnsubscribeRequest struct {
	Method string                   `json:"method"` // Must be "resources/unsubscribe"
	Params UnsubscribeRequestParams `json:"params"`
}

// UnsubscribeRequestParams contains parameters for unsubscribe request
type UnsubscribeRequestParams struct {
	URI string `json:"uri"` // Resource URI to unsubscribe from
}

// Generic JSON-RPC types

// JSONRPCRequest is a request that expects a response
type JSONRPCRequest struct {
	ID      RequestID      `json:"id"`               // Request ID
	JSONRPC string         `json:"jsonrpc"`          // Must be "2.0"
	Method  string         `json:"method"`           // Method name
	Params  map[string]any `json:"params,omitempty"` // Method parameters
}

// JSONRPCNotification is a notification which doesn't expect a response
type JSONRPCNotification struct {
	JSONRPC string         `json:"jsonrpc"`          // Must be "2.0"
	Method  string         `json:"method"`           // Method name
	Params  map[string]any `json:"params,omitempty"` // Method parameters
}

// JSONRPCResponse is a successful response to a request
type JSONRPCResponse struct {
	ID      RequestID      `json:"id"`      // Matching request ID
	JSONRPC string         `json:"jsonrpc"` // Must be "2.0"
	Result  map[string]any `json:"result"`  // Response result
}

// JSONRPCError is a response indicating an error occurred
type JSONRPCError struct {
	Error   JSONRPCErrorObject `json:"error"`   // Error information
	ID      RequestID          `json:"id"`      // Matching request ID
	JSONRPC string             `json:"jsonrpc"` // Must be "2.0"
}

// JSONRPCErrorObject contains error details
type JSONRPCErrorObject struct {
	Code    int    `json:"code"`           // Error code
	Data    any    `json:"data,omitempty"` // Additional error info
	Message string `json:"message"`        // Error description
}
