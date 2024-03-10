package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

func Example() {
	// POST /
	request1, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader("http://example.com"))
	authCookie := http.Cookie{
		Name: "jwt",
		// JWT payload: { "user_id": user_id }
		Value: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxfQ.zCGBEiC4n4X5jij4lK4nSEtrbebYxELZ6OfBwdm6CJg",
	}
	request1.Header.Set("Content-Type", "text/plain")
	request1.AddCookie(&authCookie) // optional

	// POST /api/shorten
	reqBody2, _ := json.Marshal(map[string]string{"url": "http://example.com"})
	request2, _ := http.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(reqBody2))
	request2.Header.Set("Content-Type", "application/json")
	request2.AddCookie(&authCookie) // optional

	// GET /{id}
	request3, _ := http.NewRequest(http.MethodGet, "/123", nil)
	request3.Header.Set("Content-Type", "text/plain")

	// POST /api/shorten/batch
	reqBody4, _ := json.Marshal(
		[]map[string]string{
			{"original_url": "http://example1.com", "correlation_id": "1"},
			{"original_url": "http://example2.com", "correlation_id": "2"},
		},
	)
	request4, _ := http.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(reqBody4))
	request4.Header.Set("Content-Type", "application/json")
	request4.AddCookie(&authCookie) // optional

	// DELETE /api/user/urls
	reqBody5, _ := json.Marshal([]string{"123", "456"})
	request5, _ := http.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader(reqBody5))
	request5.Header.Set("Content-Type", "application/json")
	request5.AddCookie(&authCookie)

	// GET /api/user/urls
	request6, _ := http.NewRequest(http.MethodGet, "/api/user/urls", nil)
	request6.AddCookie(&authCookie)
}
