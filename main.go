package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type Url struct {
	ID          int       `json:"id"`
	RedirectUrl string    `json:"redirectUrl"`
	ShortenUrl  string    `json:"shortenUrl"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var IdCount int = 1
var DB = make(map[string]Url)

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

func insertUrlDB(url string) string {
	ShortenUrl, err := GenerateHash(url)
	if err != nil {
		return ""
	}

	DB[ShortenUrl] = Url{
		ID:          IdCount,
		RedirectUrl: url,
		ShortenUrl:  ShortenUrl,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	IdCount++
	return ShortenUrl
}

func getUrlDB(id string) (Url, error) {
	url, ok := DB[id]
	if !ok {
		return Url{}, errors.New("URL not found")
	}
	return url, nil
}

func UrlHandler(w http.ResponseWriter, r *http.Request) {
	// capture body with url {"url": "https://example.com"}
	var data struct {
		Url string `json:"url"`
	}
	// since body is in json then it should be decoded to struct form
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "invalid body syntax", http.StatusBadRequest)
		return
	}
	// data is converted to struct now, save it in DB
	ShortenUrl := insertUrlDB(data.Url)
	res := struct {
		Url string `json:"url"`
	}{Url: ShortenUrl}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Set the Content-Type header
	w.Header().Set("Content-Type", "application/json")

	res := map[string]string{
		"hello": "world",
	}

	// 2. Encode and write the JSON response directly to the writer
	json.NewEncoder(w).Encode(res)
}

func redirectURLHandler(w http.ResponseWriter, r *http.Request) {
	// catch the part after url
	id := r.URL.Path[len("/redirect/"):]
	url, err := getUrlDB(id)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusNotFound)
	}
	http.Redirect(w, r, url.RedirectUrl, http.StatusFound)
}

func start_server() {
	fmt.Println("Server starting on http://localhost:8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		fmt.Println("Error starting server on http://localhost:8000", err)
		return
	}
}

func main() {
	http.HandleFunc("/health", HealthHandler)
	http.HandleFunc("/url", UrlHandler)
	http.HandleFunc("/redirect/", redirectURLHandler)

	start_server()
}
