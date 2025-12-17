package executor

import (
	"os"
	"strings"

	"github.com/nghyane/llm-mux/internal/translator/ir"
	log "github.com/sirupsen/logrus"
)

// debugThinking enables verbose logging of Claude thinking responses for debugging.
// Enable with: DEBUG_THINKING=1
var debugThinking = os.Getenv("DEBUG_THINKING") == "1"

// logThinkingRequest logs the request payload being sent to upstream for thinking debug.
func logThinkingRequest(payload []byte, model string) {
	if !isClaudeThinkingModel(model) {
		return
	}
	payloadStr := string(payload)
	// Log request to see if thinkingConfig is being sent
	if strings.Contains(payloadStr, "thinkingConfig") || strings.Contains(payloadStr, "thinking") {
		log.Debugf("[THINKING_TRACE] model=%s REQUEST (has thinking): %s", model, truncateForLog(payloadStr, 2000))
	} else {
		log.Debugf("[THINKING_TRACE] model=%s REQUEST (NO thinking config!): %s", model, truncateForLog(payloadStr, 2000))
	}
}

// logThinkingRawSSE logs the raw SSE line from upstream for thinking debug.
func logThinkingRawSSE(line []byte, model string) {
	if !isClaudeThinkingModel(model) {
		return
	}
	// Only log lines that might contain thinking content
	lineStr := string(line)
	if strings.Contains(lineStr, "thinking") || strings.Contains(lineStr, "thought") ||
		strings.Contains(lineStr, "reasoning") || strings.Contains(lineStr, "content_block") {
		log.Debugf("[THINKING_TRACE] model=%s RAW_SSE: %s", model, truncateForLog(lineStr, 500))
	}
}

// logThinkingPayload logs the parsed JSON payload for thinking debug.
func logThinkingPayload(payload []byte, model string) {
	if !isClaudeThinkingModel(model) {
		return
	}
	payloadStr := string(payload)
	// Log all payloads for thinking models to catch any thinking content
	if strings.Contains(payloadStr, "thinking") || strings.Contains(payloadStr, "thought") ||
		strings.Contains(payloadStr, "parts") || strings.Contains(payloadStr, "content_block") {
		log.Debugf("[THINKING_TRACE] model=%s PAYLOAD: %s", model, truncateForLog(payloadStr, 1000))
	}
}

// logThinkingEvents logs parsed IR events for thinking debug.
func logThinkingEvents(events []ir.UnifiedEvent, model string) {
	if !isClaudeThinkingModel(model) {
		return
	}
	for i, event := range events {
		switch event.Type {
		case ir.EventTypeReasoning:
			log.Debugf("[THINKING_TRACE] model=%s EVENT[%d] type=REASONING content=%s",
				model, i, truncateForLog(event.Reasoning, 200))
		case ir.EventTypeToken:
			// Log text tokens too to see full response structure
			log.Debugf("[THINKING_TRACE] model=%s EVENT[%d] type=TOKEN content=%s",
				model, i, truncateForLog(event.Content, 100))
		case ir.EventTypeToolCall:
			log.Debugf("[THINKING_TRACE] model=%s EVENT[%d] type=TOOL_CALL name=%s",
				model, i, event.ToolCall.Name)
		case ir.EventTypeFinish:
			log.Debugf("[THINKING_TRACE] model=%s EVENT[%d] type=FINISH reason=%s",
				model, i, event.FinishReason)
		}
	}
}

// isClaudeThinkingModel returns true if the model is a Claude thinking model.
func isClaudeThinkingModel(model string) bool {
	return strings.Contains(model, "claude") && strings.Contains(model, "thinking")
}

// truncateForLog truncates a string for logging purposes.
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...[truncated]"
}
