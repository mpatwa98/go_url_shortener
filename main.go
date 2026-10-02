package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type URL struct {
	ID          int       `json:"id"`
	RedirectURL string    `json:"redirect_url"`
	ShortenURL  string    `json:"shorten_url"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type URLRepository struct {
	urls   map[string]URL
	nextID int
}

func NewURLRepository() *URLRepository {
	return &URLRepository{
		urls:   make(map[string]URL),
		nextID: 1,
	}
}

func GenerateHash(url string) (string, error) {
	hasher := md5.New()

	_, err := hasher.Write([]byte(url))
	if err != nil {
		return "", fmt.Errorf("error in creating hash: %w", err)
	}

	data := hasher.Sum(nil)
	hash := hex.EncodeToString(data)

	return hash[:8], nil
}

// IsURLValid checks whether a raw string is a valid absolute HTTP/HTTPS URL.
func IsURLValid(rawURL string) bool {
	// ParseRequestURI validates format according to RFC 3986 / RFC 2616
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}

	// Ensure the scheme is http or https
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return false
	}

	// Ensure host is not empty (net/url allows scheme without host like "http:///path")
	if strings.TrimSpace(u.Host) == "" {
		return false
	}

	return true
}

func (r *URLRepository) Create(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if !IsURLValid(rawURL) {
		return "", fmt.Errorf("invalid URL")
	}

	shortURL, err := GenerateHash(rawURL)
	if err != nil {
		return "", fmt.Errorf("error generating short code: %w", err)
	}

	if existing, ok := r.urls[shortURL]; ok {
		return existing.ShortenURL, nil
	}

	id := r.nextID
	r.nextID++

	r.urls[shortURL] = URL{
		ID:          id,
		RedirectURL: rawURL,
		ShortenURL:  shortURL,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return shortURL, nil
}

func (r *URLRepository) Get(shortCode string) (URL, bool) {
	url, ok := r.urls[shortCode]
	return url, ok
}

type URLHandler struct {
	repo *URLRepository
}

func NewURLHandler(repo *URLRepository) *URLHandler {
	return &URLHandler{
		repo: repo,
	}
}

func (h *URLHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		URL string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "invalid body syntax", http.StatusBadRequest)
		return
	}

	data.URL = strings.TrimSpace(data.URL)
	if data.URL == "" || !IsURLValid(data.URL) {
		http.Error(w, "invalid URL", http.StatusBadRequest)
		return
	}

	shortURL, err := h.repo.Create(data.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	res := struct {
		URL string `json:"url"`
	}{
		URL: shortURL,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func (h *URLHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if len(r.URL.Path) <= len("/redirect/") {
		http.Error(w, "missing short code", http.StatusBadRequest)
		return
	}

	id := r.URL.Path[len("/redirect/"):]
	if id == "" {
		http.Error(w, "missing short code", http.StatusBadRequest)
		return
	}

	url, ok := h.repo.Get(id)
	if !ok {
		http.Error(w, "invalid request", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, url.RedirectURL, http.StatusFound)
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	res := map[string]string{
		"hello": "world",
	}
	json.NewEncoder(w).Encode(res)
}

func main() {
	repo := NewURLRepository()
	handler := NewURLHandler(repo)

	http.HandleFunc("/health", HealthHandler)
	http.HandleFunc("/url", handler.Create)
	http.HandleFunc("/redirect/", handler.Redirect)

	log.Fatal(http.ListenAndServe(":8000", nil))
}
