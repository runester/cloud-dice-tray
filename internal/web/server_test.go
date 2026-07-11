package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestIndex(t *testing.T) {
	server := newTestServer(t)
	response := request(t, server, http.MethodGet, "/", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Dice workbench") {
		t.Fatalf("status %d body %q", response.Code, response.Body.String())
	}
}

func TestValidate(t *testing.T) {
	server := newTestServer(t)
	valid := request(t, server, http.MethodPost, "/dice/validate", "sum(4d6)")
	if valid.Code != http.StatusOK || !strings.Contains(valid.Body.String(), "No dice were rolled") {
		t.Fatalf("valid response: status %d body %q", valid.Code, valid.Body.String())
	}

	invalid := request(t, server, http.MethodPost, "/dice/validate", "26d6")
	if invalid.Code != http.StatusOK || !strings.Contains(invalid.Body.String(), "dice count must be between") || !strings.Contains(invalid.Body.String(), "dice_count_out_of_range") {
		t.Fatalf("invalid response: status %d body %q", invalid.Code, invalid.Body.String())
	}

	semanticError := request(t, server, http.MethodPost, "/dice/validate", "3d6 + 2")
	if semanticError.Code != http.StatusOK || !strings.Contains(semanticError.Body.String(), "aggregate multiple dice") || !strings.Contains(semanticError.Body.String(), "list_in_arithmetic") {
		t.Fatalf("semantic response: status %d body %q", semanticError.Code, semanticError.Body.String())
	}
}

func TestRoll(t *testing.T) {
	server := newTestServer(t)
	response := request(t, server, http.MethodPost, "/dice/roll", "2 + 3")
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, "<strong>5</strong>") || !strings.Contains(body, "No dice rolled") {
		t.Fatalf("status %d body %q", response.Code, body)
	}
}

func TestRollErrorReturnsSwappableFeedback(t *testing.T) {
	server := newTestServer(t)
	response := request(t, server, http.MethodPost, "/dice/roll", "1 / 0")
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, "cannot divide by zero") || !strings.Contains(body, "division_by_zero") {
		t.Fatalf("status %d body %q", response.Code, body)
	}
}

func TestOutputEscapesExpression(t *testing.T) {
	server := newTestServer(t)
	response := request(t, server, http.MethodPost, "/dice/roll", "1 < 2")
	if strings.Contains(response.Body.String(), "< 2") {
		t.Fatalf("response contains unescaped user input: %q", response.Body.String())
	}
}

func TestStaticAsset(t *testing.T) {
	server := newTestServer(t)
	response := request(t, server, http.MethodGet, "/static/app.css", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), ".workbench") {
		t.Fatalf("status %d body %q", response.Code, response.Body.String())
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	server, err := New()
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func request(t *testing.T, server *Server, method, path, expression string) *httptest.ResponseRecorder {
	t.Helper()
	var body *strings.Reader
	if method == http.MethodPost {
		body = strings.NewReader(url.Values{"expression": {expression}}.Encode())
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, body)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, req)
	return response
}
