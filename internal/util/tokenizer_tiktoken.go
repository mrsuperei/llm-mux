package util

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/nghyane/llm-mux/internal/translator/ir"
	"github.com/tiktoken-go/tokenizer"
)

// tiktokenCache caches tokenizer instances to avoid re-initialization overhead.
var (
	tiktokenCache   = make(map[tokenizer.Encoding]tokenizer.Codec)
	tiktokenCacheMu sync.RWMutex
)

// ImageTokenCostOpenAI is the fixed token cost (approximate) for images in OpenAI/Claude models.
// High-res mode is usually 85 + 170*tiles. We average to a safe estimate.
const ImageTokenCostOpenAI = 255

// CountTiktokenTokens counts tokens for OpenAI, Claude, Qwen, and other models using tiktoken.
// It automatically selects o200k_base or cl100k_base based on the model.
func CountTiktokenTokens(model string, req *ir.UnifiedChatRequest) int64 {
	if req == nil {
		return 0
	}

	encodingName := getTiktokenEncodingName(model)
	enc, err := getTiktokenCodec(encodingName)
	if err != nil {
		// Fallback to simpler counting or return 0 if critical
		return 0
	}

	var totalTokens int64 = 0

	// Overhead per message (improvising OpenAI's format: <|start|>{role/name}\n{content}<|end|>\n)
	// Usually 3 or 4 tokens per message depending on the model.
	tokensPerMessage := int64(3)
	if encodingName == tokenizer.O200kBase {
		tokensPerMessage = 3 // GPT-4o style
	}

	// 1. Instructions (Responses API system instructions)
	if req.Instructions != "" {
		ids, _, _ := enc.Encode(req.Instructions)
		totalTokens += int64(len(ids)) + tokensPerMessage
	}

	// 2. Messages
	for _, msg := range req.Messages {
		totalTokens += tokensPerMessage

		// Role tokens
		// Check cache for role token count to optimize? Role is short, just encode.
		roleIds, _, _ := enc.Encode(string(msg.Role))
		totalTokens += int64(len(roleIds))

		// Content tokens
		contentStr, imageCount := irMessageToString(&msg)
		if contentStr != "" {
			ids, _, _ := enc.Encode(contentStr)
			totalTokens += int64(len(ids))
		}
		totalTokens += int64(imageCount * ImageTokenCostOpenAI)
	}

	// 3. Tools
	if len(req.Tools) > 0 {
		toolsJSON, _ := json.Marshal(req.Tools)
		ids, _, _ := enc.Encode(string(toolsJSON))
		// Tools overhead
		totalTokens += int64(len(ids)) + 10 // +10 for structure overhead
	}

	// 4. Reply priming (every reply is primed with <|start|>assistant<|message|>)
	totalTokens += 3

	return totalTokens
}

// getTiktokenCodec returns a cached codec.
func getTiktokenCodec(encoding tokenizer.Encoding) (tokenizer.Codec, error) {
	tiktokenCacheMu.RLock()
	codec, ok := tiktokenCache[encoding]
	tiktokenCacheMu.RUnlock()
	if ok {
		return codec, nil
	}

	tiktokenCacheMu.Lock()
	defer tiktokenCacheMu.Unlock()

	// Double check
	if codec, ok := tiktokenCache[encoding]; ok {
		return codec, nil
	}

	codec, err := tokenizer.Get(encoding)
	if err != nil {
		return nil, err
	}

	tiktokenCache[encoding] = codec
	return codec, nil
}

// getTiktokenEncodingName maps model names to the best available tiktoken encoding.
func getTiktokenEncodingName(model string) tokenizer.Encoding {
	lower := strings.ToLower(model)

	switch {
	// GPT-5 / GPT-4o / Claude 3.x / Qwen -> o200k_base (200k context support)
	case strings.Contains(lower, "gpt-5"),
		strings.Contains(lower, "gpt-4o"),
		strings.Contains(lower, "claude"),
		strings.Contains(lower, "qwen"),
		strings.Contains(lower, "antigravity"): // Internal name often maps to Claude/Gemini
		return tokenizer.O200kBase

	// Legacy GPT-4 / GPT-3.5 -> cl100k_base
	case strings.Contains(lower, "gpt-4"),
		strings.Contains(lower, "gpt-3.5"),
		strings.Contains(lower, "turbo"):
		return tokenizer.Cl100kBase

	// Default for modern models
	default:
		return tokenizer.O200kBase
	}
}

// irMessageToString converts an IR message to a text representation for token counting.
// Returns the text content and image count. All nil checks are performed to prevent panics.
func irMessageToString(msg *ir.Message) (string, int) {
	var sb strings.Builder
	imageCount := 0

	for _, part := range msg.Content {
		switch part.Type {
		case ir.ContentTypeText:
			if part.Text != "" {
				sb.WriteString(part.Text)
			}
		case ir.ContentTypeReasoning:
			if part.Reasoning != "" {
				sb.WriteString(part.Reasoning)
			}
		case ir.ContentTypeCodeResult:
			if part.CodeExecution != nil && part.CodeExecution.Output != "" {
				sb.WriteString(part.CodeExecution.Output)
			}
		case ir.ContentTypeExecutableCode:
			if part.CodeExecution != nil && part.CodeExecution.Code != "" {
				sb.WriteString(part.CodeExecution.Code)
			}
		case ir.ContentTypeImage:
			if part.Image != nil {
				imageCount++
			}
		case ir.ContentTypeFile:
			if part.File != nil && part.File.FileData != "" {
				sb.WriteString(part.File.FileData)
			}
		case ir.ContentTypeToolResult:
			if part.ToolResult != nil {
				sb.WriteString(fmt.Sprintf("\nTool %s result: %s", part.ToolResult.ToolCallID, part.ToolResult.Result))
				imageCount += len(part.ToolResult.Images)
			}
		}
	}

	// Tool calls
	for _, tc := range msg.ToolCalls {
		sb.WriteString(fmt.Sprintf("\nCall tool %s(%s)", tc.Name, tc.Args))
	}

	return sb.String(), imageCount
}
