package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tidwall/gjson"
)

func TestExecuteLog(t *testing.T) {
	config := json.RawMessage(`{"message":"hello world","level":"info"}`)
	out, err := ExecuteAction(context.Background(), "log", config, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gjson.GetBytes(out, "logged").Bool() {
		t.Error("expected logged=true")
	}
	if gjson.GetBytes(out, "message").String() != "hello world" {
		t.Error("expected message=hello world")
	}
	if gjson.GetBytes(out, "level").String() != "info" {
		t.Error("expected level=info")
	}
}

func TestExecuteLog_DefaultLevel(t *testing.T) {
	config := json.RawMessage(`{"message":"test"}`)
	out, err := ExecuteAction(context.Background(), "log", config, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gjson.GetBytes(out, "level").String() != "info" {
		t.Error("expected default level=info")
	}
}

func TestExecuteDelay(t *testing.T) {
	config := json.RawMessage(`{"duration_ms":1}`)
	out, err := ExecuteAction(context.Background(), "delay", config, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gjson.GetBytes(out, "delayed_ms").Int() != 1 {
		t.Error("expected delayed_ms=1")
	}
}

func TestExecuteDelay_InvalidDuration(t *testing.T) {
	config := json.RawMessage(`{"duration_ms":0}`)
	_, err := ExecuteAction(context.Background(), "delay", config, json.RawMessage(`{}`))
	if err == nil {
		t.Error("expected error for invalid duration")
	}
}

func TestExecuteDelay_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	config := json.RawMessage(`{"duration_ms":60000}`)
	_, err := ExecuteAction(ctx, "delay", config, json.RawMessage(`{}`))
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestExecuteHTTPCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("X-Custom") != "test" {
			t.Errorf("expected X-Custom header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	config, _ := json.Marshal(map[string]interface{}{
		"url":     srv.URL,
		"method":  "POST",
		"headers": map[string]string{"X-Custom": "test"},
		"body":    `{"foo":"bar"}`,
	})

	out, err := ExecuteAction(context.Background(), "http_call", config, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gjson.GetBytes(out, "status").Int() != 200 {
		t.Errorf("expected status 200, got %d", gjson.GetBytes(out, "status").Int())
	}
	if gjson.GetBytes(out, "body").String() != `{"ok":true}` {
		t.Errorf("unexpected body: %s", gjson.GetBytes(out, "body").String())
	}
}

func TestExecuteHTTPCall_MissingURL(t *testing.T) {
	config := json.RawMessage(`{"method":"GET"}`)
	_, err := ExecuteAction(context.Background(), "http_call", config, json.RawMessage(`{}`))
	if err == nil {
		t.Error("expected error for missing URL")
	}
}

func TestExecuteTransform(t *testing.T) {
	config := json.RawMessage(`{"expression":"name","output_path":"result"}`)
	runCtx := json.RawMessage(`{"name":"Alice","age":30}`)

	out, err := ExecuteAction(context.Background(), "transform", config, runCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gjson.GetBytes(out, "result").String() != "Alice" {
		t.Errorf("expected result=Alice, got %s", gjson.GetBytes(out, "result").String())
	}
}

func TestExecuteTransform_WithInputPath(t *testing.T) {
	config := json.RawMessage(`{"input_path":"data","expression":"items.#","output_path":"count"}`)
	runCtx := json.RawMessage(`{"data":{"items":["a","b","c"]}}`)

	out, err := ExecuteAction(context.Background(), "transform", config, runCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gjson.GetBytes(out, "count").Int() != 3 {
		t.Errorf("expected count=3, got %d", gjson.GetBytes(out, "count").Int())
	}
}

func TestExecuteTransform_Passthrough(t *testing.T) {
	config := json.RawMessage(`{"output_path":"data"}`)
	runCtx := json.RawMessage(`{"x":1}`)

	out, err := ExecuteAction(context.Background(), "transform", config, runCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gjson.GetBytes(out, "data.x").Int() != 1 {
		t.Errorf("expected data.x=1, got %s", string(out))
	}
}

func TestExecuteUnknownAction(t *testing.T) {
	_, err := ExecuteAction(context.Background(), "unknown", json.RawMessage(`{}`), json.RawMessage(`{}`))
	if err == nil {
		t.Error("expected error for unknown action")
	}
}
