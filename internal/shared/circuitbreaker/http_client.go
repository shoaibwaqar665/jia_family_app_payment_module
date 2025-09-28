package circuitbreaker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPClient wraps an http.Client with circuit breaker protection
type HTTPClient struct {
	client         *http.Client
	circuitBreaker *CircuitBreaker
	name           string
}

// NewHTTPClient creates a new HTTP client with circuit breaker protection
func NewHTTPClient(name string, client *http.Client, config Config) *HTTPClient {
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &HTTPClient{
		client:         client,
		circuitBreaker: NewCircuitBreaker(config),
		name:           name,
	}
}

// Do executes an HTTP request with circuit breaker protection
func (hc *HTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	var response *http.Response
	var err error

	result, err := hc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		// Add timeout to context if not already set
		if _, hasTimeout := ctx.Deadline(); !hasTimeout {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
		}

		// Execute the request
		resp, reqErr := hc.client.Do(req.WithContext(ctx))
		if reqErr != nil {
			return nil, reqErr
		}

		// Check for HTTP error status codes
		if resp.StatusCode >= 400 {
			// Read response body for error details
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()

			if readErr != nil {
				return nil, fmt.Errorf("HTTP %d: failed to read response body: %w", resp.StatusCode, readErr)
			}

			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}

		return resp, nil
	})

	if err != nil {
		return nil, err
	}

	response, ok := result.(*http.Response)
	if !ok {
		return nil, fmt.Errorf("unexpected result type from circuit breaker")
	}

	return response, nil
}

// Get executes a GET request with circuit breaker protection
func (hc *HTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	return hc.Do(ctx, req)
}

// Post executes a POST request with circuit breaker protection
func (hc *HTTPClient) Post(ctx context.Context, url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return hc.Do(ctx, req)
}

// Put executes a PUT request with circuit breaker protection
func (hc *HTTPClient) Put(ctx context.Context, url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "PUT", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return hc.Do(ctx, req)
}

// Delete executes a DELETE request with circuit breaker protection
func (hc *HTTPClient) Delete(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return nil, err
	}
	return hc.Do(ctx, req)
}

// State returns the current circuit breaker state
func (hc *HTTPClient) State() State {
	return hc.circuitBreaker.State()
}

// Metrics returns circuit breaker metrics
func (hc *HTTPClient) Metrics() Metrics {
	return hc.circuitBreaker.GetMetrics()
}

// Reset resets the circuit breaker
func (hc *HTTPClient) Reset() {
	hc.circuitBreaker.Reset()
}

// Name returns the client name
func (hc *HTTPClient) Name() string {
	return hc.name
}
