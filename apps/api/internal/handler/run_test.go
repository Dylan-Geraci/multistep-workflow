package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dylangeraci/flowforge/internal/metrics"
	"github.com/dylangeraci/flowforge/internal/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/tidwall/gjson"
)

func setupRunRouter(t *testing.T) (*chi.Mux, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	rdb := testutil.SetupTestRedis(t)

	userID := newULID()
	testutil.CreateTestUser(t, db, userID, userID+"@test.com")

	wh := NewWorkflowHandler(db)
	rh := NewRunHandler(db, rdb, metrics.New())

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(testutil.AuthContext(userID)))
		})
	})
	r.Post("/workflows", wh.Create)
	r.Post("/workflows/{id}/runs", rh.Create)
	r.Get("/workflows/{id}/runs", rh.List)
	r.Get("/runs/{id}", rh.GetByID)
	r.Post("/runs/{id}/cancel", rh.Cancel)

	return r, userID
}

func createTestRun(t *testing.T, r *chi.Mux) (workflowID, runID string) {
	t.Helper()
	// Create workflow first
	body := `{"name":"run-test-wf","description":"test","steps":[{"action":"log","config":{"message":"hi"},"name":"step1"}]}`
	req := httptest.NewRequest("POST", "/workflows", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create workflow: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	workflowID = gjson.GetBytes(w.Body.Bytes(), "id").String()

	// Create run
	req = httptest.NewRequest("POST", "/workflows/"+workflowID+"/runs", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create run: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	runID = gjson.GetBytes(w.Body.Bytes(), "id").String()
	return
}

func TestRunCreate(t *testing.T) {
	r, _ := setupRunRouter(t)
	_, runID := createTestRun(t, r)
	if runID == "" {
		t.Fatal("expected non-empty run id")
	}
}

func TestRunCreate_WorkflowNotFound(t *testing.T) {
	r, _ := setupRunRouter(t)
	req := httptest.NewRequest("POST", "/workflows/nonexistent/runs", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRunGetByID(t *testing.T) {
	r, _ := setupRunRouter(t)
	_, runID := createTestRun(t, r)

	req := httptest.NewRequest("GET", "/runs/"+runID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if gjson.GetBytes(w.Body.Bytes(), "status").String() != "pending" {
		t.Error("expected status=pending")
	}
}

func TestRunGetByID_NotFound(t *testing.T) {
	r, _ := setupRunRouter(t)
	req := httptest.NewRequest("GET", "/runs/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRunList(t *testing.T) {
	r, _ := setupRunRouter(t)
	workflowID, _ := createTestRun(t, r)

	req := httptest.NewRequest("GET", "/workflows/"+workflowID+"/runs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if gjson.GetBytes(w.Body.Bytes(), "total").Int() < 1 {
		t.Error("expected at least 1 run")
	}
}

func TestRunCancel(t *testing.T) {
	r, _ := setupRunRouter(t)
	_, runID := createTestRun(t, r)

	req := httptest.NewRequest("POST", "/runs/"+runID+"/cancel", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if gjson.GetBytes(w.Body.Bytes(), "status").String() != "cancelled" {
		t.Error("expected status=cancelled")
	}
}

func TestRunCancel_AlreadyCancelled(t *testing.T) {
	r, _ := setupRunRouter(t)
	_, runID := createTestRun(t, r)

	// Cancel first time
	req := httptest.NewRequest("POST", "/runs/"+runID+"/cancel", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first cancel: expected 200, got %d", w.Code)
	}

	// Cancel second time — should 409
	req = httptest.NewRequest("POST", "/runs/"+runID+"/cancel", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestRunCancel_NotFound(t *testing.T) {
	r, _ := setupRunRouter(t)
	req := httptest.NewRequest("POST", "/runs/nonexistent/cancel", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
