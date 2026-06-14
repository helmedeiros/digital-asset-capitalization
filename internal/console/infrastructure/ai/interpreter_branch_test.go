package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// llamaStub returns an httptest server that fakes Ollama's /api/generate.
// The given response is wrapped into the {"response": "..."} envelope so
// callLLaMA's JSON decode succeeds; tests that want a malformed envelope
// or a non-200 status build their own server directly.
func llamaStub(t *testing.T, response string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/generate", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response": ` + jsonStr(response) + `}`))
	}))
}

// jsonStr quotes a Go string into a JSON string literal so embedded
// quotes and backslashes survive the round trip. Used only to wrap test
// fixtures into the {"response": "..."} envelope; we deliberately avoid
// pulling in json.Marshal noise per fixture line.
func jsonStr(s string) string {
	out := make([]byte, 0, len(s)+2)
	out = append(out, '"')
	for _, r := range s {
		switch r {
		case '"', '\\':
			out = append(out, '\\', byte(r))
		case '\n':
			out = append(out, '\\', 'n')
		case '\r':
			out = append(out, '\\', 'r')
		case '\t':
			out = append(out, '\\', 't')
		default:
			if r < 0x20 {
				continue // skip control chars
			}
			out = append(out, []byte(string(r))...)
		}
	}
	out = append(out, '"')
	return string(out)
}

func TestInterpreter_ExtractJSONFromResponse(t *testing.T) {
	i := NewInterpreter(Config{BaseURL: "http://x", Model: "test"})

	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "json fenced code block",
			in: "Here you go:\n" +
				"```json\n" +
				"{\"command\":\"assets list\"}\n" +
				"```",
			want: "{\"command\":\"assets list\"}",
		},
		{
			name: "direct json object terminates on closing brace",
			in:   "noise\n{\"command\":\"assets list\"}\nfurther text ignored",
			want: "{\"command\":\"assets list\"}",
		},
		{
			// The extractor's terminate-on-closing-brace check only
			// matches "}" or "},", not "]" -- so a direct array
			// captures to the end of the input rather than to the
			// closing bracket. Pin this observable behavior so any
			// future tightening (e.g. terminating on "]" too) shows
			// up here.
			name: "direct json array captures until end of input",
			in:   "[\"a\", \"b\"]\ntrailing line",
			want: "[\"a\", \"b\"]\ntrailing line",
		},
		{
			name: "trailing comma after brace still terminates",
			in:   "prefix\n{\"a\":1},\ntail",
			want: "{\"a\":1},",
		},
		{
			name: "no json present returns empty",
			in:   "Just some prose with no braces.",
			want: "",
		},
		{
			name: "code block fences without json marker do not start capture",
			in:   "```\n{not really json}\n```",
			want: "{not really json}", // direct {…} branch picks it up before the fence
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, i.extractJSONFromResponse(c.in))
		})
	}
}

func TestInterpreter_Interpret_FallbackToExtractedJSON(t *testing.T) {
	// Response wraps the JSON in a fenced code block, so the outer
	// json.Unmarshal in Interpret fails and the extractJSONFromResponse
	// branch produces a parseable command.
	body := "Sure, here is the command:\n" +
		"```json\n" +
		"{\"command\":\"assets list\",\"confidence\":0.92}\n" +
		"```"
	server := llamaStub(t, body)
	defer server.Close()

	interp := NewInterpreter(Config{BaseURL: server.URL, Model: "test"})
	cmd, err := interp.Interpret(context.Background(), "list assets", domain.NewContext("s"))
	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "assets list", cmd.Interpreted)
	assert.InDelta(t, 0.92, cmd.Confidence, 1e-9)
}

func TestInterpreter_Interpret_FallbackToTextParse(t *testing.T) {
	// Plain prose with a leading "assets ..." prefix triggers the
	// parseTextResponse path's pattern match (confidence 0.7).
	body := "Looks like you want:\nassets list\nThat should help."
	server := llamaStub(t, body)
	defer server.Close()

	interp := NewInterpreter(Config{BaseURL: server.URL, Model: "test"})
	cmd, err := interp.Interpret(context.Background(), "list assets", domain.NewContext("s"))
	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "assets list", cmd.Interpreted)
}

func TestInterpreter_AnalyzeIntent_HappyPath(t *testing.T) {
	server := llamaStub(t, `{"action":"list","resource":"asset","target":"Payment Gateway"}`)
	defer server.Close()

	interp := NewInterpreter(Config{BaseURL: server.URL, Model: "test"})
	intent, err := interp.AnalyzeIntent(context.Background(), "list payment gateway assets")
	require.NoError(t, err)
	require.NotNil(t, intent)
	assert.Equal(t, domain.CommandTypeList, intent.Action)
	assert.Equal(t, domain.ResourceTypeAsset, intent.Resource)
	assert.Equal(t, "Payment Gateway", intent.Target)
}

func TestInterpreter_AnalyzeIntent_BadJSONIsAnError(t *testing.T) {
	server := llamaStub(t, "this is not json")
	defer server.Close()

	interp := NewInterpreter(Config{BaseURL: server.URL, Model: "test"})
	intent, err := interp.AnalyzeIntent(context.Background(), "do a thing")
	require.Error(t, err)
	assert.Nil(t, intent)
	assert.Contains(t, err.Error(), "failed to parse intent")
}

func TestInterpreter_callLLaMA_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	interp := NewInterpreter(Config{BaseURL: server.URL, Model: "test"})
	got, err := interp.callLLaMA(context.Background(), "hello")
	require.Error(t, err)
	assert.Empty(t, got)
	assert.Contains(t, err.Error(), "unexpected status code: 500")
}

func TestInterpreter_callLLaMA_MalformedEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer server.Close()

	interp := NewInterpreter(Config{BaseURL: server.URL, Model: "test"})
	got, err := interp.callLLaMA(context.Background(), "hello")
	require.Error(t, err)
	assert.Empty(t, got)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestInterpreter_callLLaMA_TrimsResponseWhitespace(t *testing.T) {
	server := llamaStub(t, "   surrounded by space   ")
	defer server.Close()

	interp := NewInterpreter(Config{BaseURL: server.URL, Model: "test"})
	got, err := interp.callLLaMA(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, "surrounded by space", got)
}

func TestInterpreter_setCommandIntent_ShortCommand(t *testing.T) {
	// A command with fewer than two fields should be a no-op -- not
	// panic, not mutate the command's intent fields.
	interp := NewInterpreter(Config{BaseURL: "http://x", Model: "test"})
	cmd := &domain.Command{}
	interp.setCommandIntent(cmd, InterpretationResult{Command: "single"})
	assert.Equal(t, domain.CommandType(""), cmd.Intent.Action)
	assert.Equal(t, domain.ResourceType(""), cmd.Intent.Resource)
}
