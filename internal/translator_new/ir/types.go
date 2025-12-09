package ir

// MetadataKey constants for built-in tools and provider-specific passthrough.
// Using constants prevents typos and enables IDE autocomplete.
const (
	// Built-in tools (Gemini)
	MetaGoogleSearch          = "google_search"
	MetaGoogleSearchRetrieval = "google_search_retrieval"
	MetaCodeExecution         = "code_execution"
	MetaURLContext            = "url_context"
	MetaGroundingMetadata     = "grounding_metadata" // Response: search results from googleSearch

	// Format-specific passthrough keys (namespaced to prevent collisions)
	// OpenAI-specific fields
	MetaOpenAILogprobs         = "openai:logprobs"          // boolean - request log probabilities
	MetaOpenAITopLogprobs      = "openai:top_logprobs"      // int - number of top tokens
	MetaOpenAILogitBias        = "openai:logit_bias"        // map - token probability bias
	MetaOpenAISeed             = "openai:seed"              // int - deterministic sampling seed
	MetaOpenAIUser             = "openai:user"              // string - end-user identifier
	MetaOpenAIFrequencyPenalty = "openai:frequency_penalty" // float - repetition penalty
	MetaOpenAIPresencePenalty  = "openai:presence_penalty"  // float - presence penalty

	// Gemini-specific fields
	MetaGeminiCachedContent = "gemini:cachedContent" // string - cached content name
	MetaGeminiLabels        = "gemini:labels"        // map - request labels

	// Claude-specific fields
	MetaClaudeMetadata = "claude:metadata" // map - request metadata (user_id, etc.)
)

// EventType defines the type of event in the unified stream.
type EventType string

const (
	EventTypeToken            EventType = "token"
	EventTypeReasoning        EventType = "reasoning"         // For model reasoning/thinking content
	EventTypeReasoningSummary EventType = "reasoning_summary" // For reasoning summary (Responses API)
	EventTypeToolCall         EventType = "tool_call"         // Complete tool call
	EventTypeToolCallDelta    EventType = "tool_call_delta"   // Incremental tool call arguments (Responses API)
	EventTypeImage            EventType = "image"             // For inline image content
	EventTypeCodeExecution    EventType = "code_execution"    // For Gemini code execution results
	EventTypeError            EventType = "error"
	EventTypeFinish           EventType = "finish"
)

// FinishReason defines why the model stopped generating.
type FinishReason string

const (
	FinishReasonStop          FinishReason = "stop"           // Natural completion
	FinishReasonLength        FinishReason = "length"         // Max tokens reached
	FinishReasonToolCalls     FinishReason = "tool_calls"     // Model wants to call tools
	FinishReasonContentFilter FinishReason = "content_filter" // Content filtered by safety
	FinishReasonError         FinishReason = "error"          // Error occurred
	FinishReasonUnknown       FinishReason = "unknown"        // Unknown reason
)

// UnifiedEvent represents a single event in the chat stream.
// It is the "Esperanto" response format.
type UnifiedEvent struct {
	Type              EventType
	Content           string             // For EventTypeToken
	Reasoning         string             // For EventTypeReasoning (model thinking/reasoning content)
	ReasoningSummary  string             // For EventTypeReasoningSummary (Responses API)
	ThoughtSignature  string             // For Gemini thought signatures, Claude signatures, OpenAI reasoning_opaque, etc.
	ToolCall          *ToolCall          // For EventTypeToolCall
	ToolCallIndex     int                // Index for tool call in parallel calls (Responses API)
	Image             *ImagePart         // For EventTypeImage (inline image content)
	CodeExecution     *CodeExecutionPart // For EventTypeCodeExecution (Gemini code execution)
	GroundingMetadata *GroundingMetadata // For search grounding results (Gemini googleSearch)
	Error             error              // For EventTypeError
	Usage             *Usage             // Optional usage stats on Finish
	FinishReason      FinishReason       // Why generation stopped (for EventTypeFinish)
	Refusal           string             // Refusal message (if model refuses to answer)
	Logprobs          interface{}        // Log probabilities (if requested)
	ContentFilter     interface{}        // Content filter results
	SystemFingerprint string             // System fingerprint
}

// Usage represents token usage statistics.
type Usage struct {
	PromptTokens       int
	CompletionTokens   int
	TotalTokens        int
	ThoughtsTokenCount int // Reasoning/thinking token count (for completion_tokens_details)
	CachedTokens       int // Cached input tokens (Responses API prompt caching)
	AudioTokens        int // Audio input tokens
	AcceptedPredictionTokens int // Accepted prediction tokens
	RejectedPredictionTokens int // Rejected prediction tokens
}

// ResponseMeta contains metadata from upstream response for passthrough.
// Used to preserve original response fields like responseId, createTime, finishReason.
type ResponseMeta struct {
	ResponseID         string      // Original response ID from upstream (e.g., Gemini responseId)
	CreateTime         int64       // Unix timestamp from upstream (parsed from createTime)
	NativeFinishReason string      // Original finish reason string from upstream (e.g., "STOP", "MAX_TOKENS")
	Logprobs           interface{} // Log probabilities from response (OpenAI format)
}

// CandidateResult holds the result of a single candidate/choice from the model.
// Used when candidateCount/n > 1 to return multiple alternatives.
type CandidateResult struct {
	Index        int          // Candidate index (0-based)
	Messages     []Message    // Messages from this candidate
	FinishReason FinishReason // Why this candidate stopped
	Logprobs     interface{}  // Log probabilities for this candidate (OpenAI format)
}

// ToolCall represents a request from the model to execute a tool.
type ToolCall struct {
	ID               string
	Name             string
	Args             string // JSON string of arguments
	PartialArgs      string // Raw partial arguments (e.g. Gemini partialArgs)
	ThoughtSignature string // Gemini thought signature for this tool call
}

// Role defines the role of the message sender.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// ContentType defines the type of content part.
type ContentType string

const (
	ContentTypeText          ContentType = "text"
	ContentTypeReasoning     ContentType = "reasoning"      // For model reasoning/thinking content
	ContentTypeImage         ContentType = "image"
	ContentTypeFile          ContentType = "file"           // For file inputs (PDF, etc.) - Responses API
	ContentTypeToolResult    ContentType = "tool_result"
	ContentTypeExecutableCode ContentType = "executable_code" // Gemini code execution: code to run
	ContentTypeCodeResult    ContentType = "code_result"      // Gemini code execution: result
)

// ContentPart represents a discrete part of a message (e.g., a block of text, an image).
type ContentPart struct {
	Type             ContentType
	Text             string          // Populated if Type == ContentTypeText
	Reasoning        string          // Populated if Type == ContentTypeReasoning
	ThoughtSignature string          // Gemini thought signature
	Image            *ImagePart      // Populated if Type == ContentTypeImage
	File             *FilePart       // Populated if Type == ContentTypeFile (Responses API)
	ToolResult       *ToolResultPart // Populated if Type == ContentTypeToolResult
	CodeExecution    *CodeExecutionPart // Populated if Type == ContentTypeExecutableCode or ContentTypeCodeResult
}

type ImagePart struct {
	MimeType string
	Data     string // Base64 encoded data
	URL      string // URL for remote images (Responses API)
}

// FilePart represents a file input (PDF, etc.) for Responses API.
type FilePart struct {
	FileID   string // File ID from uploaded file
	FileURL  string // URL for remote file
	Filename string // Original filename
	FileData string // Base64 encoded data (data:application/pdf;base64,...)
}

type ToolResultPart struct {
	ToolCallID string
	Result     string       // JSON string result
	Images     []*ImagePart // Multimodal tool result (images)
	Files      []*FilePart  // Multimodal tool result (files)
}

// CodeExecutionPart represents Gemini code execution content.
type CodeExecutionPart struct {
	Language string // Programming language (e.g., "PYTHON")
	Code     string // Source code (for executableCode)
	Outcome  string // Execution outcome: "OUTCOME_OK", "OUTCOME_FAILED", etc. (for codeExecutionResult)
	Output   string // Execution output/result (for codeExecutionResult)
}

// GroundingMetadata contains search grounding information from Gemini.
type GroundingMetadata struct {
	SearchEntryPoint *SearchEntryPoint `json:"searchEntryPoint,omitempty"`
	GroundingChunks  []GroundingChunk  `json:"groundingChunks,omitempty"`
	WebSearchQueries []string          `json:"webSearchQueries,omitempty"`
}

// SearchEntryPoint contains the rendered search entry point HTML.
type SearchEntryPoint struct {
	RenderedContent string `json:"renderedContent,omitempty"`
}

// GroundingChunk represents a single grounding source.
type GroundingChunk struct {
	Web *WebGrounding `json:"web,omitempty"`
}

// WebGrounding contains web source information.
type WebGrounding struct {
	URI   string `json:"uri,omitempty"`
	Title string `json:"title,omitempty"`
}

// Message represents a single message in the conversation history.
type Message struct {
	Role      Role
	Content   []ContentPart
	ToolCalls []ToolCall // Populated if Role == RoleAssistant and there are tool calls
}

// ToolDefinition represents a tool capability exposed to the model.
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]interface{} // JSON Schema object (cleaned)
}

// UnifiedChatRequest represents the unified chat request structure.
// It is the "Esperanto" request format.
type UnifiedChatRequest struct {
	Model            string
	Messages         []Message
	Tools            []ToolDefinition
	Temperature      *float64 // Pointer to allow nil (default)
	TopP             *float64
	TopK             *int
	MaxTokens        *int
	StopSequences    []string
	FrequencyPenalty *float64 // Repetition penalty (OpenAI/Gemini)
	PresencePenalty  *float64 // Presence penalty (OpenAI/Gemini)
	Logprobs         *bool    // Request log probabilities (OpenAI/Gemini)
	TopLogprobs      *int     // Number of top log probabilities (OpenAI)
	CandidateCount   *int     // Number of candidates/completions (OpenAI n / Gemini candidateCount)
	Thinking         *ThinkingConfig // Specific to models that support "thinking" (e.g. Gemini 2.0 Flash Thinking)
	SafetySettings   []SafetySetting // Safety/content filtering settings
	ImageConfig      *ImageConfig    // Image generation configuration
	ResponseModality []string        // Response modalities (e.g., ["TEXT", "IMAGE"])
	Metadata         map[string]any  // Additional provider-specific metadata

	// Responses API specific fields
	Instructions       string         // System instructions (Responses API)
	PreviousResponseID string         // For conversation continuity (Responses API)
	PromptID           string         // Prompt template ID (Responses API)
	PromptVersion      string         // Prompt template version (Responses API)
	PromptVariables    map[string]any // Variables for prompt template (Responses API)
	PromptCacheKey     string         // Cache key for prompt caching (Responses API)
	Store              *bool          // Whether to store the response (Responses API)
	ParallelToolCalls  *bool          // Whether to allow parallel tool calls (Responses API)
	ToolChoice         string         // Tool choice mode (Responses API)
	ResponseSchema     map[string]any        // JSON Schema for structured output (Gemini/Ollama)
	FunctionCalling    *FunctionCallingConfig // Function calling configuration
}

// FunctionCallingConfig controls function calling behavior.
type FunctionCallingConfig struct {
	Mode                       string   // "AUTO", "ANY", "NONE"
	AllowedFunctionNames       []string // Whitelist of functions
	StreamFunctionCallArguments bool     // Enable streaming of arguments (Gemini 3+)
}

// ThinkingConfig controls the reasoning capabilities of the model.
type ThinkingConfig struct {
	IncludeThoughts bool
	Budget          int    // Token budget for thinking (-1 for auto, 0 for disabled)
	Summary         string // Reasoning summary mode: "auto", "concise", "detailed" (Responses API)
	Effort          string // Reasoning effort: "none", "low", "medium", "high" (Responses API)
}

// SafetySetting represents content safety filtering configuration.
type SafetySetting struct {
	Category  string // e.g., "HARM_CATEGORY_HARASSMENT"
	Threshold string // e.g., "OFF", "BLOCK_NONE", "BLOCK_LOW_AND_ABOVE"
}

// ImageConfig controls image generation parameters.
type ImageConfig struct {
	AspectRatio string // e.g., "1:1", "16:9", "9:16"
	ImageSize   string // e.g., "256x256", "512x512"
}
