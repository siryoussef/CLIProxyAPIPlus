package registry

import "testing"

func TestParseOpencodeModelsFiltersToSupportedFreeChatModels(t *testing.T) {
	data := []byte(`{"data":[
		{"id":"minimax-m2.5","created":1790848163,"owned_by":"opencode"},
		{"id":"big-pickle","created":1790848163,"owned_by":"opencode"},
		{"id":"deepseek-v4-flash-free","created":1790848163,"owned_by":"opencode"},
		{"id":"mimo-v2.6-flash-free","created":1790848163,"owned_by":"opencode"},
		{"id":"jev-1.13-free","created":1790848163,"owned_by":"opencode"},
		{"id":"muse-spark-1.3-contributor-free","created":1790848163,"owned_by":"opencode"}
	]}`)

	models, err := parseOpencodeModels(data)
	if err != nil {
		t.Fatalf("parseOpencodeModels returned error: %v", err)
	}

	want := map[string]bool{
		"big-pickle":             false,
		"deepseek-v4-flash-free": false,
		"mimo-v2.6-flash-free":   false,
	}
	for _, model := range models {
		if _, ok := want[model.ID]; !ok {
			t.Errorf("unexpected model %q", model.ID)
			continue
		}
		want[model.ID] = true
	}
	for id, found := range want {
		if !found {
			t.Errorf("expected supported free model %q", id)
		}
	}
	if len(models) != len(want) {
		t.Fatalf("got %d models, want %d", len(models), len(want))
	}
}

func TestParseOpencodeModelsRejectsCatalogWithoutSupportedModels(t *testing.T) {
	_, err := parseOpencodeModels([]byte(`{"data":[{"id":"minimax-m2.5"}]}`))
	if err == nil {
		t.Fatal("parseOpencodeModels accepted a catalog without supported free chat models")
	}
}
