package registry

import (
	"math/rand"
	"sort"
	"sync"
)

type FamilyMember struct {
	Provider string
	ModelID  string
	Priority int
}

// ModelFamilies: canonical → provider-specific mappings (only where IDs differ)
var ModelFamilies = map[string][]FamilyMember{
	"claude-sonnet-4-5": {
		{Provider: "kiro", ModelID: "claude-sonnet-4-5", Priority: 1},
		{Provider: "antigravity", ModelID: "gemini-claude-sonnet-4-5", Priority: 1},
		{Provider: "claude", ModelID: "claude-sonnet-4-5-20250929", Priority: 2},
	},
	"claude-sonnet-4-5-thinking": {
		{Provider: "claude", ModelID: "claude-sonnet-4-5-thinking", Priority: 1},
		{Provider: "antigravity", ModelID: "gemini-claude-sonnet-4-5-thinking", Priority: 2},
	},
	"claude-opus-4-5": {
		{Provider: "claude", ModelID: "claude-opus-4-5-20251101", Priority: 1},
		{Provider: "kiro", ModelID: "claude-opus-4-5-20251101", Priority: 2},
	},
	"claude-opus-4-5-thinking": {
		{Provider: "antigravity", ModelID: "gemini-claude-opus-4-5-thinking", Priority: 1},
		{Provider: "claude", ModelID: "claude-opus-4-5-thinking", Priority: 2},
	},
	"claude-sonnet-4": {
		{Provider: "kiro", ModelID: "claude-sonnet-4-20250514", Priority: 1},
		{Provider: "claude", ModelID: "claude-sonnet-4-20250514", Priority: 2},
	},
	"claude-3-7-sonnet": {
		{Provider: "kiro", ModelID: "claude-3-7-sonnet-20250219", Priority: 1},
		{Provider: "claude", ModelID: "claude-3-7-sonnet-20250219", Priority: 2},
	},
	"gpt-5.1-codex-max": {
		{Provider: "github-copilot", ModelID: "gpt-5.1-codex-max", Priority: 1},
		{Provider: "openai", ModelID: "gpt-5.1-codex-max", Priority: 2},
	},
}

var (
	translationIndex map[string]map[string]string
	indexOnce        sync.Once
)

func buildIndexes() {
	translationIndex = make(map[string]map[string]string, len(ModelFamilies))
	for canonical, members := range ModelFamilies {
		pm := make(map[string]string, len(members))
		for _, m := range members {
			pm[m.Provider] = m.ModelID
		}
		translationIndex[canonical] = pm
	}
}

func IsCanonicalID(modelID string) bool {
	_, ok := ModelFamilies[modelID]
	return ok
}

func ResolveAllProviders(canonicalID string, availableProviders []string) ([]string, bool) {
	family, ok := ModelFamilies[canonicalID]
	if !ok {
		return nil, false
	}

	if len(availableProviders) == 1 {
		for _, m := range family {
			if m.Provider == availableProviders[0] {
				return []string{m.Provider}, true
			}
		}
		return nil, false
	}

	availableSet := make(map[string]struct{}, len(availableProviders))
	for _, p := range availableProviders {
		availableSet[p] = struct{}{}
	}

	type pg struct {
		pri int
		mem []string
	}
	groups := make([]pg, 0, 3)
	gIdx := make(map[int]int)

	for _, m := range family {
		if _, ok := availableSet[m.Provider]; !ok {
			continue
		}
		if idx, exists := gIdx[m.Priority]; exists {
			groups[idx].mem = append(groups[idx].mem, m.Provider)
		} else {
			gIdx[m.Priority] = len(groups)
			groups = append(groups, pg{pri: m.Priority, mem: []string{m.Provider}})
		}
	}

	if len(groups) == 0 {
		return nil, false
	}

	sort.Slice(groups, func(i, j int) bool { return groups[i].pri < groups[j].pri })

	result := make([]string, 0, len(family))
	for _, g := range groups {
		if len(g.mem) > 1 {
			rand.Shuffle(len(g.mem), func(i, j int) { g.mem[i], g.mem[j] = g.mem[j], g.mem[i] })
		}
		result = append(result, g.mem...)
	}
	return result, true
}

func TranslateModelForProvider(canonicalID, provider string) string {
	indexOnce.Do(buildIndexes)
	if pm, ok := translationIndex[canonicalID]; ok {
		if modelID, ok := pm[provider]; ok {
			return modelID
		}
	}
	return canonicalID
}

func GetFamilyMembers(canonicalID string) []FamilyMember {
	family, ok := ModelFamilies[canonicalID]
	if !ok {
		return nil
	}
	sorted := make([]FamilyMember, len(family))
	copy(sorted, family)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Priority < sorted[j].Priority })
	return sorted
}
