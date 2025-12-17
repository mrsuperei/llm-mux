package util

import (
	"strings"

	"github.com/nghyane/llm-mux/internal/registry"
)

// DefaultThinkingBudget is the safe default budget for auto-enabling thinking.
// Provide a fixed value (e.g. 1024) instead of dynamic (-1) because some upstream
// providers (e.g. Antigravity/Google) rely on fixed budgets for mapped models like Claude.
const DefaultThinkingBudget = 1024

// ModelSupportsThinking reports whether the given model has Thinking capability
// according to the model registry metadata (provider-agnostic).
func ModelSupportsThinking(model string) bool {
	if model == "" {
		return false
	}
	if info := registry.GetGlobalRegistry().GetModelInfo(model); info != nil {
		return info.Thinking != nil
	}
	return false
}

// NormalizeThinkingBudget clamps the requested thinking budget to the
// supported range for the specified model using registry metadata only.
// If the model is unknown or has no Thinking metadata, returns the original budget.
// For dynamic (-1), returns -1 if DynamicAllowed; otherwise approximates mid-range
// or min (0 if zero is allowed and mid <= 0).
func NormalizeThinkingBudget(model string, budget int) int {
	if budget == -1 { // dynamic
		if found, min, max, zeroAllowed, dynamicAllowed := thinkingRangeFromRegistry(model); found {
			if dynamicAllowed {
				return -1
			}
			mid := (min + max) / 2
			if mid <= 0 && zeroAllowed {
				return 0
			}
			if mid <= 0 {
				return min
			}
			return mid
		}
		return -1
	}
	if found, min, max, zeroAllowed, _ := thinkingRangeFromRegistry(model); found {
		if budget == 0 {
			if zeroAllowed {
				return 0
			}
			return min
		}
		if budget < min {
			return min
		}
		if budget > max {
			return max
		}
		return budget
	}
	return budget
}

// GetAutoAppliedThinkingConfig returns the default thinking configuration for a model
// if it should be auto-applied (e.g. model supports thinking but no explicit config found).
// Returns (budget, include_thoughts, should_apply).
func GetAutoAppliedThinkingConfig(model string) (int, bool, bool) {
	if ModelSupportsThinking(model) {
		// Use fixed budget (1024) instead of dynamic (-1) as upstream might not support dynamic for Claude
		// or other mapped models.
		return DefaultThinkingBudget, true, true
	}
	return 0, false, false
}

// thinkingRangeFromRegistry attempts to read thinking ranges from the model registry.
func thinkingRangeFromRegistry(model string) (found bool, min int, max int, zeroAllowed bool, dynamicAllowed bool) {
	if model == "" {
		return false, 0, 0, false, false
	}
	lower := strings.ToLower(model)
	// Try exact match first, then registry lookup
	// (Note: GetGlobalRegistry().GetModelInfo handles normalization usually, but safety check)
	info := registry.GetGlobalRegistry().GetModelInfo(lower)
	if info == nil {
		// Try original model string if lower fail
		info = registry.GetGlobalRegistry().GetModelInfo(model)
	}

	if info == nil || info.Thinking == nil {
		return false, 0, 0, false, false
	}
	return true, info.Thinking.Min, info.Thinking.Max, info.Thinking.ZeroAllowed, info.Thinking.DynamicAllowed
}
