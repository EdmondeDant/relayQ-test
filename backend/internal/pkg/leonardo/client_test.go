package leonardo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type errorTransport struct {
	err error
}

func (t errorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, t.err
}

func TestListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/api/rest/v2/models" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("authorization = %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("accept = %q", got)
		}
		_, _ = fmt.Fprint(w, `{"productionApiAvailableModels":[{"id":"model-id","name":"Model Name","parameters":{"type":"object"}}]}`)
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/api/rest/", "test-key", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].ID != "model-id" || models[0].Name != "Model Name" || string(models[0].Parameters) != `{"type":"object"}` {
		t.Fatalf("models = %#v", models)
	}
}

func TestListModelsErrorDetailsAndRedaction(t *testing.T) {
	key := "secret-leonardo-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", "request-"+key)
		w.Header().Set("Retry-After", "17")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = fmt.Fprintf(w, `{"error":"rejected %s api_key=another-secret","path":"$.%s","code":"rate-limit"}`, key, key)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, key, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListModels(context.Background())
	var apiErr *LeonardoError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T %v", err, err)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests || apiErr.Code != "rate-limit" || apiErr.RequestID != "request-***" || apiErr.RetryAfter != 17*time.Second || !apiErr.RetryableRead {
		t.Fatalf("error = %#v", apiErr)
	}
	if strings.Contains(apiErr.Message, key) || strings.Contains(apiErr.Message, "another-secret") || strings.Contains(apiErr.Path, key) || strings.Contains(apiErr.Error(), key) {
		t.Fatalf("secret leaked: %#v %q", apiErr, apiErr.Error())
	}
}

func TestListModelsRejectsRedirect(t *testing.T) {
	targetCalled := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		targetCalled = true
	}))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "key", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListModels(context.Background())
	var apiErr *LeonardoError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusFound {
		t.Fatalf("error = %T %v", err, err)
	}
	if targetCalled {
		t.Fatal("redirect was followed")
	}
}

func TestListModelsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = fmt.Fprint(w, `{"productionApiAvailableModels":[]}`)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "key", 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListModels(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Client.Timeout") {
		t.Fatalf("error = %v", err)
	}
}

func TestListModelsResponseLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, strings.Repeat("x", MaxResponseBodySize+1))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "key", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListModels(context.Background())
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("error = %v", err)
	}
}

func TestListModelsHTTPDateRetryAfter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", time.Now().Add(5*time.Second).UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprint(w, `{"error":"busy","path":"$","code":"unavailable"}`)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "key", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListModels(context.Background())
	var apiErr *LeonardoError
	if !errors.As(err, &apiErr) || apiErr.RetryAfter < 3*time.Second || apiErr.RetryAfter > 5*time.Second {
		t.Fatalf("error = %#v", apiErr)
	}
}

func TestNewClientValidationAndIsolation(t *testing.T) {
	for _, baseURL := range []string{"ftp://example.com", "https://user:pass@example.com", "https://example.com?api_key=secret"} {
		if _, err := NewClient(baseURL, "key", time.Second); err == nil {
			t.Errorf("base URL %q accepted", baseURL)
		}
	}
	if _, err := NewClient("https://example.com", " ", time.Second); err == nil {
		t.Fatal("empty key accepted")
	}
	original := &http.Client{Timeout: 2 * time.Second}
	client, err := NewClientWithHTTPClient("https://example.com", "key", time.Second, original)
	if err != nil {
		t.Fatal(err)
	}
	if original.Timeout != 2*time.Second || original.CheckRedirect != nil || client.httpClient == original {
		t.Fatal("provided HTTP client was mutated")
	}
}

func TestListModelsTransportErrorRedaction(t *testing.T) {
	key := "secret-leonardo-key"
	client, err := NewClientWithHTTPClient("https://example.com", key, time.Second, &http.Client{
		Transport: errorTransport{err: fmt.Errorf("authorization Bearer %s", key)},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListModels(context.Background())
	if err == nil || strings.Contains(err.Error(), key) {
		t.Fatalf("error = %v", err)
	}
}
