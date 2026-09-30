package executor

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/thinking"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/util"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// OpencodeExecutor handles requests to the OpenCode Zen API.
type OpencodeExecutor struct {
	cfg *config.Config
}

// NewOpencodeExecutor creates a new OpenCode executor instance.
func NewOpencodeExecutor(cfg *config.Config) *OpencodeExecutor {
	return &OpencodeExecutor{cfg: cfg}
}

// Identifier returns the unique identifier for this executor.
func (e *OpencodeExecutor) Identifier() string { return "opencode" }

// CountTokens returns the token count for the given request.
func (e *OpencodeExecutor) CountTokens(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, fmt.Errorf("opencode: count tokens not supported")
}

// Refresh validates the Opencode token.
func (e *OpencodeExecutor) Refresh(ctx context.Context, auth *cliproxyauth.Auth) (*cliproxyauth.Auth, error) {
	if auth == nil {
		return nil, fmt.Errorf("missing auth")
	}
	return auth, nil
}

func opencodeUserAgent() string {
	return fmt.Sprintf("opencode/1.18.31 (%s %s; %s)", runtime.GOOS, runtime.GOARCH, runtime.Version())
}

func canonicalSessionID(signal string) string {
	sum := sha256.Sum256([]byte("ses\x00" + signal))
	timePart := hex.EncodeToString(sum[:6])
	
	const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var randPart strings.Builder
	for i := 0; i < 14; i++ {
		b := sum[6+i]
		randPart.WriteByte(base62Alphabet[b%62])
	}
	
	return "ses_" + timePart + randPart.String()
}

func randomOpencodeMsgID() string {
	signal := fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
	sum := sha256.Sum256([]byte("msg\x00" + signal))
	timePart := hex.EncodeToString(sum[:6])
	
	const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var randPart strings.Builder
	for i := 0; i < 14; i++ {
		b := sum[6+i]
		randPart.WriteByte(base62Alphabet[b%62])
	}
	
	return "msg_" + timePart + randPart.String()
}

// PrepareRequest prepares the HTTP request before execution.
func (e *OpencodeExecutor) PrepareRequest(req *http.Request, auth *cliproxyauth.Auth) error {
	if req == nil {
		return nil
	}
	apiKey := "public"
	if auth != nil && func() string { _, v := auth.AccountInfo(); return v }() != "" {
		apiKey = func() string { _, v := auth.AccountInfo(); return v }()
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", opencodeUserAgent())
	req.Header.Set("x-opencode-client", "cli")
	req.Header.Set("x-opencode-project", "global")
	
	var attrs map[string]string
	if auth != nil {
		attrs = auth.Attributes
	}
	util.ApplyCustomHeadersFromAttrs(req, attrs)
	return nil
}

// HttpRequest executes a raw HTTP request.
func (e *OpencodeExecutor) HttpRequest(ctx context.Context, auth *cliproxyauth.Auth, req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("opencode executor: request is nil")
	}
	if ctx == nil {
		ctx = req.Context()
	}
	httpReq := req.WithContext(ctx)
	if err := e.PrepareRequest(httpReq, auth); err != nil {
		return nil, err
	}
	httpClient := newProxyAwareHTTPClient(ctx, e.cfg, auth, 0)
	return httpClient.Do(httpReq)
}

// anonymousCoreTools are the tool names the OpenCode Zen free tier requires
// to be present in every agent-shaped request. Requests missing any of these
// names are rejected with 403 FreeTierError.
// Source: https://github.com/jasonxu114514/opencode2api (gateway/upstream.go)
var anonymousCoreTools = []string{"bash", "edit", "glob", "grep", "read"}

// isFreeModel checks if the model is served by the Zen free tier
// (no-auth "Bearer public" lane). All models on the Zen endpoint require the
// agent-shaped gate, but -free suffix / big-pickle always land on the
// anonymous lane which additionally enforces the tool gate.
func isFreeModel(model string) bool {
	lower := strings.ToLower(model)
	return strings.HasSuffix(lower, "-free") || lower == "big-pickle"
}

// injectGateTools ensures the request body has stream:true and contains all
// five anonymous core tools required by the OpenCode Zen free-tier gate.
// Missing tools are synthesized with minimal definitions; existing tools and
// all other body fields are preserved unchanged.
func injectGateTools(payload []byte) ([]byte, error) {
	var body map[string]interface{}
	if err := json.Unmarshal(payload, &body); err != nil {
		return payload, nil
	}

	// Free tier gate requires stream: true.
	body["stream"] = true

	tools, _ := body["tools"].([]interface{})

	// Build a set of present tool names (handle both OpenAI and Anthropic shapes).
	present := make(map[string]bool, len(tools))
	for _, toolObj := range tools {
		tmap, ok := toolObj.(map[string]interface{})
		if !ok {
			continue
		}
		// OpenAI chat-completions shape: tool.function.name
		if fobj, ok := tmap["function"].(map[string]interface{}); ok {
			if name, ok := fobj["name"].(string); ok && name != "" {
				present[name] = true
			}
		}
		// Anthropic shape: tool.name
		if name, ok := tmap["name"].(string); ok && name != "" {
			present[name] = true
		}
	}

	// Inject any missing core tools.
	for _, name := range anonymousCoreTools {
		if present[name] {
			continue
		}
		tools = append(tools, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        name,
				"description": "Agent tool " + name,
				"parameters": map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		})
	}

	body["tools"] = tools
	return json.Marshal(body)
}

func (e *OpencodeExecutor) doExecution(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options, forceStream bool) (*http.Response, *usageReporter, sdktranslator.Format, sdktranslator.Format, []byte, error) {
	baseModel := thinking.ParseSuffix(req.Model).ModelName

	reporter := newUsageReporter(ctx, e.Identifier(), baseModel, auth)

	from := opts.SourceFormat
	to := sdktranslator.FromString("openai")
	endpoint := "/zen/v1/chat/completions"
	
	apiKey := "public"
	if auth != nil && func() string { _, v := auth.AccountInfo(); return v }() != "" {
		apiKey = func() string { _, v := auth.AccountInfo(); return v }()
	}

	stream := opts.Stream
	if forceStream {
		stream = true
	}

	originalPayloadSource := req.Payload
	if len(opts.OriginalRequest) > 0 {
		originalPayloadSource = opts.OriginalRequest
	}
	originalTranslated := sdktranslator.TranslateRequest(from, to, baseModel, originalPayloadSource, stream)
	translated := sdktranslator.TranslateRequest(from, to, baseModel, req.Payload, stream)
	
	if isFreeModel(baseModel) {
		if newPayload, err := injectGateTools(translated); err == nil {
			translated = newPayload
		}
	}

	requestedModel := payloadRequestedModel(opts, req.Model)
	translated = applyPayloadConfigWithRoot(e.cfg, baseModel, to.String(), "", translated, originalTranslated, requestedModel)

	translated, err := thinking.ApplyThinking(translated, req.Model, from.String(), to.String(), e.Identifier())
	if err != nil {
		return nil, reporter, from, to, translated, err
	}

	url := "https://opencode.ai" + endpoint
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(translated))
	if err != nil {
		return nil, reporter, from, to, translated, err
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("User-Agent", opencodeUserAgent())
	httpReq.Header.Set("x-opencode-client", "cli")
	httpReq.Header.Set("x-opencode-project", "global")
	httpReq.Header.Set("x-opencode-request", randomOpencodeMsgID())
	
	// Create a stable session ID based on the original request ID or context
	sessionSeed := fmt.Sprintf("%d", time.Now().UnixNano())
	httpReq.Header.Set("x-opencode-session", canonicalSessionID(sessionSeed))

	var attrs map[string]string
	if auth != nil {
		attrs = auth.Attributes
	}
	util.ApplyCustomHeadersFromAttrs(httpReq, attrs)
	if forceStream {
		httpReq.Header.Set("Accept", "text/event-stream")
		httpReq.Header.Set("Cache-Control", "no-cache")
	}

	var authID, authLabel, authType, authValue string
	if auth != nil {
		authID = auth.ID
		authLabel = auth.Label
		authType, authValue = auth.AccountInfo()
	}
	recordAPIRequest(ctx, e.cfg, upstreamRequestLog{
		URL:       url,
		Method:    http.MethodPost,
		Headers:   httpReq.Header.Clone(),
		Body:      translated,
		Provider:  e.Identifier(),
		AuthID:    authID,
		AuthLabel: authLabel,
		AuthType:  authType,
		AuthValue: authValue,
	})

	httpClient := newProxyAwareHTTPClient(ctx, e.cfg, auth, 0)
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		recordAPIResponseError(ctx, e.cfg, err)
		return nil, reporter, from, to, translated, err
	}

	recordAPIResponseMetadata(ctx, e.cfg, httpResp.StatusCode, httpResp.Header.Clone())
	
	return httpResp, reporter, from, to, translated, nil
}

// Execute performs a non-streaming request.
func (e *OpencodeExecutor) Execute(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (resp cliproxyexecutor.Response, err error) {
	// Free tier gate requires stream to be true
	forceStream := isFreeModel(thinking.ParseSuffix(req.Model).ModelName)
	
	httpResp, reporter, from, to, translated, err := e.doExecution(ctx, auth, req, opts, forceStream)
	if err != nil {
		if reporter != nil {
			reporter.trackFailure(ctx, &err)
		}
		return resp, err
	}
	defer reporter.trackFailure(ctx, &err)
	defer httpResp.Body.Close()
	
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		b, _ := io.ReadAll(httpResp.Body)
		appendAPIResponseChunk(ctx, e.cfg, b)
		err = statusErr{code: httpResp.StatusCode, msg: string(b)}
		return resp, err
	}

	var body []byte
	if forceStream {
		// Collect the stream into a single payload
		body, err = e.collectStreamResponse(httpResp.Body)
		if err != nil {
			return resp, err
		}
	} else {
		body, err = io.ReadAll(httpResp.Body)
		if err != nil {
			recordAPIResponseError(ctx, e.cfg, err)
			return resp, err
		}
	}
	
	appendAPIResponseChunk(ctx, e.cfg, body)
	reporter.publish(ctx, parseOpenAIUsage(body))
	reporter.ensurePublished(ctx)

	var param any
	out := sdktranslator.TranslateNonStream(ctx, to, from, req.Model, opts.OriginalRequest, translated, body, &param)
	resp = cliproxyexecutor.Response{Payload: []byte(out)}
	return resp, nil
}

func (e *OpencodeExecutor) collectStreamResponse(body io.Reader) ([]byte, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(nil, 52_428_800)
	var fullContent strings.Builder
	var lastChunk []byte
	
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		lastChunk = []byte(data)
		content := gjson.Get(data, "choices.0.delta.content").String()
		fullContent.WriteString(content)
	}
	
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	
	// Create a synthetic OpenAI non-stream response from the last chunk
	if len(lastChunk) > 0 {
		synth, err := sjson.SetBytes(lastChunk, "choices.0.message.role", "assistant")
		if err == nil {
			synth, err = sjson.SetBytes(synth, "choices.0.message.content", fullContent.String())
			if err == nil {
				synth, _ = sjson.DeleteBytes(synth, "choices.0.delta")
				return synth, nil
			}
		}
	}
	
	// Fallback empty response
	return []byte(`{"choices":[{"message":{"role":"assistant","content":""}}]}`), nil
}

// ExecuteStream performs a streaming request.
func (e *OpencodeExecutor) ExecuteStream(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (_ *cliproxyexecutor.StreamResult, err error) {
	httpResp, reporter, from, to, translated, err := e.doExecution(ctx, auth, req, opts, true)
	if err != nil {
		if reporter != nil {
			reporter.trackFailure(ctx, &err)
		}
		return nil, err
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		b, _ := io.ReadAll(httpResp.Body)
		appendAPIResponseChunk(ctx, e.cfg, b)
		httpResp.Body.Close()
		err = statusErr{code: httpResp.StatusCode, msg: string(b)}
		reporter.trackFailure(ctx, &err)
		return nil, err
	}

	out := make(chan cliproxyexecutor.StreamChunk)
	go func() {
		defer close(out)
		defer httpResp.Body.Close()
		defer reporter.ensurePublished(ctx)

		scanner := bufio.NewScanner(httpResp.Body)
		scanner.Buffer(nil, 52_428_800)
		var param any
		for scanner.Scan() {
			line := scanner.Bytes()
			appendAPIResponseChunk(ctx, e.cfg, line)
			if detail, ok := parseOpenAIStreamUsage(line); ok {
				reporter.publish(ctx, detail)
			}
			if len(line) == 0 {
				continue
			}
			if !bytes.HasPrefix(line, []byte("data:")) {
				continue
			}
			chunks := sdktranslator.TranslateStream(ctx, to, from, req.Model, opts.OriginalRequest, translated, bytes.Clone(line), &param)
			for i := range chunks {
				out <- cliproxyexecutor.StreamChunk{Payload: []byte(chunks[i])}
			}
		}
		if errScan := scanner.Err(); errScan != nil {
			recordAPIResponseError(ctx, e.cfg, errScan)
			out <- cliproxyexecutor.StreamChunk{Err: errScan}
		}
	}()

	return &cliproxyexecutor.StreamResult{
		Headers: httpResp.Header.Clone(),
		Chunks:  out,
	}, nil
}
