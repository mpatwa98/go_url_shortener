package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	HealthHandler(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}

	var payload map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode health payload: %v", err)
	}
	if payload["hello"] != "world" {
		t.Fatalf("expected hello=world, got %#v", payload)
	}
}

func TestCreateURLReturnsCreatedForValidURL(t *testing.T) {
	repo := NewURLRepository()
	handler := NewURLHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/url", bytes.NewBufferString(`{"url":"https://example.com"}`))
	res := httptest.NewRecorder()

	handler.Create(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, res.Code, res.Body.String())
	}

	var payload struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.URL == "" {
		t.Fatal("expected shortened URL in response body")
	}
}

func TestCreateURLRejectsInvalidURL(t *testing.T) {
	repo := NewURLRepository()
	handler := NewURLHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/url", bytes.NewBufferString(`{"url":"not-a-valid-url"}`))
	res := httptest.NewRecorder()

	handler.Create(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, res.Code, res.Body.String())
	}
}

func TestCreateURLRejectsEmptyURL(t *testing.T) {
	repo := NewURLRepository()
	handler := NewURLHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/url", bytes.NewBufferString(`{"url":"   "}`))
	res := httptest.NewRecorder()

	handler.Create(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, res.Code, res.Body.String())
	}
}

func TestCreateURLRejectsMalformedJSON(t *testing.T) {
	repo := NewURLRepository()
	handler := NewURLHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/url", strings.NewReader(`{"url":`))
	res := httptest.NewRecorder()

	handler.Create(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, res.Code, res.Body.String())
	}
}

func TestCreateURLRejectsWrongMethod(t *testing.T) {
	repo := NewURLRepository()
	handler := NewURLHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/url", nil)
	res := httptest.NewRecorder()

	handler.Create(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d: %s", http.StatusMethodNotAllowed, res.Code, res.Body.String())
	}
}

func TestCreateURLReturnsSameShortCodeForDuplicateURL(t *testing.T) {
	repo := NewURLRepository()
	handler := NewURLHandler(repo)
	payload := `{"url":"https://duplicate.example.com"}`

	firstReq := httptest.NewRequest(http.MethodPost, "/url", bytes.NewBufferString(payload))
	firstRes := httptest.NewRecorder()
	handler.Create(firstRes, firstReq)

	secondReq := httptest.NewRequest(http.MethodPost, "/url", bytes.NewBufferString(payload))
	secondRes := httptest.NewRecorder()
	handler.Create(secondRes, secondReq)

	if firstRes.Code != http.StatusCreated || secondRes.Code != http.StatusCreated {
		t.Fatalf("expected both calls to return %d, got %d and %d", http.StatusCreated, firstRes.Code, secondRes.Code)
	}

	var firstBody, secondBody struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(firstRes.Body.Bytes(), &firstBody); err != nil {
		t.Fatalf("failed to decode first response: %v", err)
	}
	if err := json.Unmarshal(secondRes.Body.Bytes(), &secondBody); err != nil {
		t.Fatalf("failed to decode second response: %v", err)
	}
	if firstBody.URL != secondBody.URL {
		t.Fatalf("expected same short code for duplicate URLs, got %q and %q", firstBody.URL, secondBody.URL)
	}
}

func TestRedirectHandlerReturnsFoundForKnownShortCode(t *testing.T) {
	repo := NewURLRepository()
	handler := NewURLHandler(repo)
	shortURL, err := repo.Create("https://redirect.example.com")
	if err != nil {
		t.Fatalf("failed to create test url: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/redirect/"+shortURL, nil)
	res := httptest.NewRecorder()

	handler.Redirect(res, req)

	if res.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusFound, res.Code, res.Body.String())
	}
	if got := res.Header().Get("Location"); got != "https://redirect.example.com" {
		t.Fatalf("expected Location header %q, got %q", "https://redirect.example.com", got)
	}
}

func TestRedirectHandlerReturnsNotFoundForUnknownShortCode(t *testing.T) {
	repo := NewURLRepository()
	handler := NewURLHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/redirect/unknown", nil)
	res := httptest.NewRecorder()

	handler.Redirect(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, res.Code, res.Body.String())
	}
}

func TestRedirectHandlerRejectsMissingShortCode(t *testing.T) {
	repo := NewURLRepository()
	handler := NewURLHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/redirect/", nil)
	res := httptest.NewRecorder()

	handler.Redirect(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, res.Code, res.Body.String())
	}
}

func TestRedirectHandlerRejectsWrongMethod(t *testing.T) {
	repo := NewURLRepository()
	handler := NewURLHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/redirect/abc12345", nil)
	res := httptest.NewRecorder()

	handler.Redirect(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d: %s", http.StatusMethodNotAllowed, res.Code, res.Body.String())
	}
}

func TestURLRepositoryCreateAndGet(t *testing.T) {
	repo := NewURLRepository()

	shortURL, err := repo.Create("https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := repo.Get(shortURL)
	if !ok {
		t.Fatal("expected URL to exist")
	}

	if got.RedirectURL != "https://example.com" {
		t.Fatalf("expected redirect URL %q, got %q",
			"https://example.com",
			got.RedirectURL,
		)
	}
}
