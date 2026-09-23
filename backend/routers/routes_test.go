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
