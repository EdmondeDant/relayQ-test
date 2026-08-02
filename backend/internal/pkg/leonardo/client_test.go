package leonardo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testGenerationID = "123e4567-e89b-42d3-a456-426614174000"

type errorTransport struct {
	err error
}

func (t errorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, t.err
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type errorReadCloser struct {
	err error
}

func (r errorReadCloser) Read([]byte) (int, error) {
	return 0, r.err
}

func (errorReadCloser) Close() error {
	return nil
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

func TestCreateGenerationOnceAndParseCosts(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/api/rest/v2/generations" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer key" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("headers = %#v", r.Header)
		}
		_, _ = fmt.Fprintf(w, `{"generationId":%q,"cost":{"amount":0.12,"unit":"USD"},"apiCreditCost":9}`, testGenerationID)
	}))
	defer server.Close()
	client, err := NewClient(server.URL+"/api/rest", "key", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.CreateGeneration(context.Background(), CreateGenerationRequest{
		Model: "verified-slug", Parameters: map[string]any{"prompt": "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || response.GenerationID != testGenerationID || response.Cost == nil || response.Cost.Amount != 0.12 || response.Cost.Unit != "USD" || response.APICreditCost == nil || *response.APICreditCost != 9 {
		t.Fatalf("calls = %d, response = %#v", calls, response)
	}
}

func TestCreateGenerationAPICreditCostMissingAndZero(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		nil  bool
	}{
		{name: "missing", body: fmt.Sprintf(`{"generationId":%q}`, testGenerationID), nil: true},
		{name: "zero", body: fmt.Sprintf(`{"generationId":%q,"apiCreditCost":0}`, testGenerationID)},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = fmt.Fprint(w, test.body)
			}))
			defer server.Close()
			client, err := NewClient(server.URL, "key", time.Second)
			if err != nil {
				t.Fatal(err)
			}
			response, err := client.CreateGeneration(context.Background(), CreateGenerationRequest{Model: "verified-slug", Parameters: map[string]any{}})
			if err != nil {
				t.Fatal(err)
			}
			if test.nil && response.APICreditCost != nil {
				t.Fatalf("apiCreditCost = %v", *response.APICreditCost)
			}
			if !test.nil && (response.APICreditCost == nil || *response.APICreditCost != 0) {
				t.Fatalf("apiCreditCost = %v", response.APICreditCost)
			}
		})
	}
}

func TestCreateGenerationTransportFailures(t *testing.T) {
	key := "secret-leonardo-key"
	for _, test := range []struct {
		name           string
		wrote          bool
		wantUnknown    bool
		wantNotWritten bool
	}{
		{name: "before write", wantNotWritten: true},
		{name: "after write", wrote: true, wantUnknown: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			client, err := NewClientWithHTTPClient("https://example.com", key, time.Second, &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				if test.wrote {
					trace := httptrace.ContextClientTrace(req.Context())
					trace.WroteRequest(httptrace.WroteRequestInfo{})
				}
				return nil, fmt.Errorf("authorization Bearer %s", key)
			})})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.CreateGeneration(context.Background(), CreateGenerationRequest{Model: "verified-slug", Parameters: map[string]any{}})
			var apiErr *LeonardoError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error = %T %v", err, err)
			}
			if calls.Load() != 1 || apiErr.SafeToRetry != test.wantNotWritten || strings.Contains(apiErr.Error(), key) {
				t.Fatalf("calls = %d, error = %#v", calls.Load(), apiErr)
			}
			if errors.Is(err, ErrGenerationRequestNotWritten) != test.wantNotWritten {
				t.Fatalf("not-written classification = %v, error = %#v", errors.Is(err, ErrGenerationRequestNotWritten), apiErr)
			}
			if (apiErr.SubmissionStatus == SubmissionUnknown) != test.wantUnknown || (apiErr.SideEffectStatus == SideEffectUnknown) != test.wantUnknown {
				t.Fatalf("error = %#v", apiErr)
			}
		})
	}
}

func TestCreateGenerationResponseFailuresAreUnknown(t *testing.T) {
	key := "secret-leonardo-key"
	for _, test := range []struct {
		name string
		body io.ReadCloser
	}{
		{name: "body read", body: errorReadCloser{err: fmt.Errorf("api_key=%s", key)}},
		{name: "decode", body: io.NopCloser(strings.NewReader("{"))},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			client, err := NewClientWithHTTPClient("https://example.com", key, time.Second, &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				trace := httptrace.ContextClientTrace(req.Context())
				trace.WroteRequest(httptrace.WroteRequestInfo{})
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: test.body, Request: req}, nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.CreateGeneration(context.Background(), CreateGenerationRequest{Model: "verified-slug", Parameters: map[string]any{}})
			var apiErr *LeonardoError
			if !errors.As(err, &apiErr) || calls.Load() != 1 || apiErr.SubmissionStatus != SubmissionUnknown || apiErr.SideEffectStatus != SideEffectUnknown || apiErr.SafeToRetry || strings.Contains(apiErr.Error(), key) {
				t.Fatalf("calls = %d, error = %#v", calls.Load(), apiErr)
			}
		})
	}
}

func TestCreateGenerationUnknownIsNotRetried(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"error":"upstream failed","path":"$","code":"internal"}`)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "key", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.CreateGeneration(context.Background(), CreateGenerationRequest{Model: "verified-slug", Parameters: map[string]any{}})
	var apiErr *LeonardoError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T %v", err, err)
	}
	if calls != 1 || apiErr.SubmissionStatus != SubmissionUnknown || apiErr.SideEffectStatus != SideEffectUnknown || apiErr.SafeToRetry || apiErr.RetryableRead {
		t.Fatalf("calls = %d, error = %#v", calls, apiErr)
	}
}

func TestCreateGenerationInvalidSuccessIsUnknown(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = fmt.Fprint(w, `{"generationId":"not-a-uuid","cost":{"amount":1,"unit":"USD"}}`)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "key", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.CreateGeneration(context.Background(), CreateGenerationRequest{Model: "verified-slug", Parameters: map[string]any{}})
	var apiErr *LeonardoError
	if !errors.As(err, &apiErr) || calls != 1 || apiErr.SubmissionStatus != SubmissionUnknown || apiErr.SideEffectStatus != SideEffectUnknown || apiErr.SafeToRetry {
		t.Fatalf("calls = %d, error = %#v", calls, apiErr)
	}
}

func TestCreateGenerationRejectsRedirect(t *testing.T) {
	targetCalls := 0
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { targetCalls++ }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "key", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.CreateGeneration(context.Background(), CreateGenerationRequest{Model: "verified-slug", Parameters: map[string]any{}})
	var apiErr *LeonardoError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusTemporaryRedirect || targetCalls != 0 || apiErr.SafeToRetry {
		t.Fatalf("target calls = %d, error = %#v", targetCalls, apiErr)
	}
}

func TestGetGenerationStatusesAndImages(t *testing.T) {
	statuses := []string{"PENDING", "COMPLETE", "FAILED"}
	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/rest/v1/generations/"+testGenerationID {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				_, _ = fmt.Fprintf(w, `{"generations_by_pk":{"id":%q,"status":%q,"generated_images":[{"id":"image-1","url":"https://example.com/image.png","nsfw":true}]}}`, testGenerationID, status)
			}))
			defer server.Close()
			client, err := NewClient(server.URL+"/api/rest", "key", time.Second)
			if err != nil {
				t.Fatal(err)
			}
			generation, err := client.GetGeneration(context.Background(), testGenerationID)
			if err != nil {
				t.Fatal(err)
			}
			if generation.ID != testGenerationID || generation.Status != status || len(generation.GeneratedImages) != 1 || generation.GeneratedImages[0].ID != "image-1" || generation.GeneratedImages[0].URL != "https://example.com/image.png" || !generation.GeneratedImages[0].NSFW {
				t.Fatalf("generation = %#v", generation)
			}
		})
	}
}

func TestGetGenerationRejectsInvalidIDAndStatus(t *testing.T) {
	var calls atomic.Int32
	client, err := NewClientWithHTTPClient("https://example.com", "key", time.Second, &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, errors.New("unexpected request")
	})})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", "generation-1", "../other"} {
		if _, err := client.GetGeneration(context.Background(), id); err == nil {
			t.Fatalf("invalid ID %q accepted", id)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("GET calls = %d", calls.Load())
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"generations_by_pk":{"id":%q,"status":"UNKNOWN"}}`, testGenerationID)
	}))
	defer server.Close()
	client, err = NewClient(server.URL, "key", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetGeneration(context.Background(), testGenerationID); err == nil {
		t.Fatal("unknown status accepted")
	}
}

func TestGetGenerationDoesNotFollowRedirect(t *testing.T) {
	var sourceCalls atomic.Int32
	var targetCalls atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { targetCalls.Add(1) }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sourceCalls.Add(1)
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	client, err := NewClient(source.URL, "key", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetGeneration(context.Background(), testGenerationID)
	var apiErr *LeonardoError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusTemporaryRedirect || sourceCalls.Load() != 1 || targetCalls.Load() != 0 {
		t.Fatalf("source=%d target=%d error=%#v", sourceCalls.Load(), targetCalls.Load(), apiErr)
	}
	if apiErr.SubmissionStatus != "" || apiErr.SideEffectStatus != "" || errors.Is(err, ErrGenerationRequestNotWritten) {
		t.Fatalf("GET error has create semantics: %#v", apiErr)
	}
}

func TestGetGenerationSanitizesReadDecodeStatusAndTransportErrors(t *testing.T) {
	for _, test := range []struct {
		name      string
		transport http.RoundTripper
	}{
		{name: "transport", transport: transportFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("Authorization secret-key") })},
		{name: "read", transport: transportFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: errorReadCloser{err: errors.New("Cookie secret-key")}}, nil
		})},
		{name: "decode", transport: transportFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{`))}, nil
		})},
		{name: "status", transport: transportFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"generations_by_pk":{"id":%q,"status":"secret-key"}}`, testGenerationID)))}, nil
		})},
	} {
		t.Run(test.name, func(t *testing.T) {
			client, err := NewClientWithHTTPClient("https://example.com", "secret-key", time.Second, &http.Client{Transport: test.transport})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.GetGeneration(context.Background(), testGenerationID)
			if err == nil || strings.Contains(err.Error(), "secret-key") || errors.Is(err, ErrGenerationRequestNotWritten) {
				t.Fatalf("error = %v", err)
			}
			var apiErr *LeonardoError
			if errors.As(err, &apiErr) && (apiErr.SubmissionStatus != "" || apiErr.SideEffectStatus != "") {
				t.Fatalf("GET error has create semantics: %#v", apiErr)
			}
		})
	}
}
