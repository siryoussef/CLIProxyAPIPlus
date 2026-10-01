package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

const opencodeModelsURL = "https://opencode.ai/zen/v1/models"

type opencodeCatalogResponse struct {
	Data []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

var opencodeCatalog = struct {
	sync.RWMutex
	models []*ModelInfo
}{models: defaultOpencodeModels()}

// GetOpencodeModels returns the latest supported free OpenCode Zen models.
func GetOpencodeModels() []*ModelInfo {
	opencodeCatalog.RLock()
	defer opencodeCatalog.RUnlock()
	return cloneOpencodeModels(opencodeCatalog.models)
}

func defaultOpencodeModels() []*ModelInfo {
	ids := []string{
		"deepseek-v4-flash-free",
		"big-pickle",
		"mimo-v2.6-flash-free",
		"space-bunny-free",
		"longcat-2.5-preview-free",
		"mimo-v2.5-free",
		"ling-3.0-flash-fin-free",
		"nemotron-3-ultra-free",
		"nemotron-3.5-lightning-free",
	}
	models := make([]*ModelInfo, 0, len(ids))
	for _, id := range ids {
		models = append(models, opencodeModelInfo(id, 0, "opencode"))
	}
	return models
}

func cloneOpencodeModels(models []*ModelInfo) []*ModelInfo {
	cloned := make([]*ModelInfo, 0, len(models))
	for _, model := range models {
		if model == nil {
			continue
		}
		modelCopy := *model
		cloned = append(cloned, &modelCopy)
	}
	return cloned
}

func opencodeModelInfo(id string, created int64, ownedBy string) *ModelInfo {
	if ownedBy == "" {
		ownedBy = "opencode"
	}
	return &ModelInfo{
		ID:                  id,
		DisplayName:         id,
		ContextLength:       128000,
		MaxCompletionTokens: 4096,
		OwnedBy:             ownedBy,
		Type:                "opencode",
		Object:              "model",
		Created:             created,
	}
}

func isOpencodeFreeChatModel(id string) bool {
	lower := strings.ToLower(strings.TrimSpace(id))
	if lower != "big-pickle" && !strings.HasSuffix(lower, "-free") {
		return false
	}
	// These free models use different Zen APIs than this executor's chat-completions endpoint.
	if strings.HasPrefix(lower, "jev-") || strings.Contains(lower, "contributor-free") {
		return false
	}
	return true
}

func parseOpencodeModels(data []byte) ([]*ModelInfo, error) {
	var catalog opencodeCatalogResponse
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("decode OpenCode Zen models: %w", err)
	}

	models := make([]*ModelInfo, 0, len(catalog.Data))
	seen := make(map[string]struct{}, len(catalog.Data))
	for _, item := range catalog.Data {
		id := strings.TrimSpace(item.ID)
		if !isOpencodeFreeChatModel(id) {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		models = append(models, opencodeModelInfo(id, item.Created, item.OwnedBy))
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("OpenCode Zen catalog contains no supported free chat models")
	}
	return models, nil
}

func fetchOpencodeModels(ctx context.Context) ([]*ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opencodeModelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create OpenCode Zen models request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch OpenCode Zen models: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch OpenCode Zen models: unexpected HTTP status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read OpenCode Zen models: %w", err)
	}
	return parseOpencodeModels(data)
}

func updateOpencodeModels(models []*ModelInfo) bool {
	if len(models) == 0 {
		return false
	}
	opencodeCatalog.Lock()
	defer opencodeCatalog.Unlock()
	if !modelSectionChanged(opencodeCatalog.models, models) {
		return false
	}
	opencodeCatalog.models = cloneOpencodeModels(models)
	return true
}
