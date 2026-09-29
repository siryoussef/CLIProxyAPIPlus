package registry

// GetOpencodeModels returns a list of static OpenCode models,
// representing the models available in the OpenCode Zen free tier.
func GetOpencodeModels() []*ModelInfo {
	return []*ModelInfo{
		{
			ID:                "deepseek-v4-flash-free",
			DisplayName:       "DeepSeek V4 Flash (Free)",
			ContextLength:     128000,
			MaxCompletionTokens: 4096,
			OwnedBy:           "opencode",
			Type:              "opencode",
			Object:            "model",
			Created:           1790624717,
		},
		{
			ID:                "big-pickle",
			DisplayName:       "Big Pickle (Free)",
			ContextLength:     128000,
			MaxCompletionTokens: 4096,
			OwnedBy:           "opencode",
			Type:              "opencode",
			Object:            "model",
			Created:           1790624717,
		},
		{
			ID:                "minimax-m2.5-free",
			DisplayName:       "MiniMax M2.5 (Free)",
			ContextLength:     128000,
			MaxCompletionTokens: 4096,
			OwnedBy:           "opencode",
			Type:              "opencode",
			Object:            "model",
			Created:           1790624717,
		},
		{
			ID:                "nemotron-3-super-free",
			DisplayName:       "Nemotron 3 Super (Free)",
			ContextLength:     128000,
			MaxCompletionTokens: 4096,
			OwnedBy:           "opencode",
			Type:              "opencode",
			Object:            "model",
			Created:           1790624717,
		},
		{
			ID:                "qwen3.6-plus-free",
			DisplayName:       "Qwen 3.6 Plus (Free)",
			ContextLength:     128000,
			MaxCompletionTokens: 4096,
			OwnedBy:           "opencode",
			Type:              "opencode",
			Object:            "model",
			Created:           1790624717,
		},
	}
}