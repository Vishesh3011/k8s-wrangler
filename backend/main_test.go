package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type healthResponse struct {
	Status string `json:"status"`
}

func TestGetMongoDBUrl_Defaults(t *testing.T) {
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_PORT")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("DB_NAME")

	url, _ := getMongoDBUrl()
	if url != "mongodb://admin:password@0.0.0.0:27017/tasksdb?authSource=admin&authMechanism=SCRAM-SHA-256" {
		t.Fatalf("expected default URL, got %q", url)
	}
}

func TestGetMongoDBUrl_FromEnv(t *testing.T) {
	t.Setenv("DB_HOST", "mongo.example.com")
	t.Setenv("DB_PORT", "12345")
	t.Setenv("DB_USER", "testuser")
	t.Setenv("DB_PASSWORD", "testpass")
	t.Setenv("DB_NAME", "mytasks")

	url, _ := getMongoDBUrl()
	if url != "mongodb://testuser:testpass@mongo.example.com:12345/mytasks?authSource=admin&authMechanism=SCRAM-SHA-256" {
		t.Fatalf("expected env URL, got %q", url)
	}
}

func TestHealthCheck_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	healthCheck(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp healthResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}

	if resp.Status != "i am alive!!!" {
		t.Fatalf("expected health status 'i am alive!!!', got %q", resp.Status)
	}
}

func TestHealthCheck_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rr := httptest.NewRecorder()

	healthCheck(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestAddTaskHandler_InvalidJson(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/tasks/add",
		bytes.NewBufferString("{invalid json}"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	addTaskHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestAddTaskHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/tasks/add", nil)
	rr := httptest.NewRecorder()

	addTaskHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestGetTasksHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/tasks", nil)
	rr := httptest.NewRecorder()

	getTasksHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}
