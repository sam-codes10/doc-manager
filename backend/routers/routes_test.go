package routers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSwaggerEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := InitRouters()

	// In httptest, RequestURI must be explicitly set because gin-swagger matches against RequestURI
	req, _ := http.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	req.RequestURI = "/swagger/index.html"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /swagger/index.html, got %d: %s", w.Code, w.Body.String())
	}

	// Test doc.json
	reqDoc, _ := http.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
	reqDoc.RequestURI = "/swagger/doc.json"
	wDoc := httptest.NewRecorder()
	r.ServeHTTP(wDoc, reqDoc)

	if wDoc.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /swagger/doc.json, got %d: %s", wDoc.Code, wDoc.Body.String())
	}
}

func TestDocumentEventsRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := InitRouters()

	// An invalid UUID should reach the handler and return 400 Bad Request (not 404 Route Not Found)
	req, _ := http.NewRequest(http.MethodGet, "/api/documents/invalid-uuid/events", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request for invalid UUID, got %d", w.Code)
	}
}

func TestCORSHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := InitRouters()

	// Preflight OPTIONS request
	req, _ := http.NewRequest(http.MethodOptions, "/api/documents", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 for OPTIONS request, got %d", w.Code)
	}
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin: *, got %q", origin)
	}
}
