package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dylangeraci/flowforge/internal/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/tidwall/gjson"
)

func setupWorkflowRouter(t *testing.T) (*chi.Mux, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)

	userID := newULID()
	testutil.CreateTestUser(t, db, userID, userID+"@test.com")

	h := NewWorkflowHandler(db)
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(testutil.AuthContext(userID)))
		})
	})
	r.Post("/workflows", h.Create)
	r.Get("/workflows", h.List)
	r.Get("/workflows/{id}", h.GetByID)
	r.Put("/workflows/{id}", h.Update)
	r.Delete("/workflows/{id}", h.Delete)

	return r, userID
}

func createTestWorkflow(t *testing.T, r *chi.Mux) string {
	t.Helper()
	body := `{"name":"test-wf","description":"test","steps":[{"action":"log","config":{"message":"hi"},"name":"step1"}]}`
	req := httptest.NewRequest("POST", "/workflows", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create workflow: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	return gjson.GetBytes(w.Body.Bytes(), "id").String()
}

func TestWorkflowCreate(t *testing.T) {
	r, _ := setupWorkflowRouter(t)
	id := createTestWorkflow(t, r)
	if id == "" {
		t.Fatal("expected non-empty workflow id")
	}
}

func TestWorkflowCreate_ValidationErrors(t *testing.T) {
	r, _ := setupWorkflowRouter(t)

	tests := []struct {
		name string
		body string
	}{
		{"missing name", `{"steps":[{"action":"log","name":"s"}]}`},
		{"no steps", `{"name":"test"}`},
		{"invalid action", `{"name":"test","steps":[{"action":"bad","name":"s"}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/workflows", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
		})
	}
}

func TestWorkflowGetByID(t *testing.T) {
	r, _ := setupWorkflowRouter(t)
	id := createTestWorkflow(t, r)

	req := httptest.NewRequest("GET", "/workflows/"+id, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if gjson.GetBytes(w.Body.Bytes(), "name").String() != "test-wf" {
		t.Error("expected name=test-wf")
	}
	if gjson.GetBytes(w.Body.Bytes(), "steps.#").Int() != 1 {
		t.Error("expected 1 step")
	}
}

func TestWorkflowGetByID_NotFound(t *testing.T) {
	r, _ := setupWorkflowRouter(t)

	req := httptest.NewRequest("GET", "/workflows/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestWorkflowList(t *testing.T) {
	r, _ := setupWorkflowRouter(t)
	createTestWorkflow(t, r)
	createTestWorkflow(t, r)

	req := httptest.NewRequest("GET", "/workflows", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	total := gjson.GetBytes(w.Body.Bytes(), "total").Int()
	if total < 2 {
		t.Errorf("expected at least 2 workflows, got %d", total)
	}
}

func TestWorkflowUpdate(t *testing.T) {
	r, _ := setupWorkflowRouter(t)
	id := createTestWorkflow(t, r)

	body := `{"name":"updated-wf","description":"updated","steps":[{"action":"delay","config":{"duration_ms":100},"name":"delay-step"},{"action":"log","config":{"message":"done"},"name":"log-step"}]}`
	req := httptest.NewRequest("PUT", "/workflows/"+id, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := w.Body.Bytes()
	if gjson.GetBytes(result, "name").String() != "updated-wf" {
		t.Error("expected name=updated-wf")
	}
	if gjson.GetBytes(result, "steps.#").Int() != 2 {
		t.Error("expected 2 steps after update")
	}
}

func TestWorkflowUpdate_NotFound(t *testing.T) {
	r, _ := setupWorkflowRouter(t)

	body := `{"name":"x","steps":[{"action":"log","config":{},"name":"s"}]}`
	req := httptest.NewRequest("PUT", "/workflows/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestWorkflowDelete(t *testing.T) {
	r, _ := setupWorkflowRouter(t)
	id := createTestWorkflow(t, r)

	req := httptest.NewRequest("DELETE", "/workflows/"+id, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	req = httptest.NewRequest("GET", "/workflows/"+id, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w.Code)
	}
}

func TestWorkflowDelete_NotFound(t *testing.T) {
	r, _ := setupWorkflowRouter(t)

	req := httptest.NewRequest("DELETE", "/workflows/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestWorkflowCreate_InvalidJSON(t *testing.T) {
	r, _ := setupWorkflowRouter(t)

	req := httptest.NewRequest("POST", "/workflows", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	errObj := resp["error"].(map[string]interface{})
	if errObj["code"] != "INVALID_JSON" {
		t.Errorf("expected INVALID_JSON error code")
	}
}
