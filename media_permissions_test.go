package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMediaPermissionsAllowOnlyExplicitSameOriginCapture(t *testing.T) {
	response := httptest.NewRecorder()
	mediaPermissions(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(response, httptest.NewRequest("GET", "https://wails.localhost/", nil))
	if got := response.Header().Get("Permissions-Policy"); got != "microphone=(self), camera=(), display-capture=(self), geolocation=()" {
		t.Fatalf("media policy blocks explicit screen capture or permits unwanted origins: %q", got)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store, max-age=0" {
		t.Fatal("stale frontend cache allowed", got)
	}
	if response.Code != http.StatusNoContent {
		t.Fatal("asset handler not called")
	}
}
