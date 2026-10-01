// Package nextdns is a small client for the NextDNS API (https://nextdns.github.io/api/).
package nextdns

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL is the address of the NextDNS API.
const DefaultBaseURL = "https://api.nextdns.io"

// Client sends authenticated requests to the NextDNS API.
type Client struct {
	BaseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient returns a client that authenticates with the given API key.
func NewClient(apiKey string) *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			// NextDNS can take close to a minute to answer some requests.
			Timeout:       2 * time.Minute,
			CheckRedirect: checkRedirect,
		},
	}
}

// checkRedirect only follows redirects to the same host. Go sends X-Api-Key
// again when it follows a redirect, so this keeps the key from leaking.
func checkRedirect(req *http.Request, via []*http.Request) error {
	if req.URL.Scheme != via[0].URL.Scheme || req.URL.Host != via[0].URL.Host {
		return fmt.Errorf("refusing redirect to %s: it would send the API key to another host", req.URL.Host)
	}
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	return nil
}

// ErrorDetail is one entry of the "errors" list in a NextDNS response.
type ErrorDetail struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Source struct {
		Parameter string `json:"parameter"`
	} `json:"source"`
}

// APIError is returned when NextDNS reports an error.
type APIError struct {
	StatusCode int
	Errors     []ErrorDetail
	// Body holds the raw response when it did not contain a list of errors.
	Body string
}

func (e *APIError) Error() string {
	if len(e.Errors) == 0 {
		return fmt.Sprintf("NextDNS API error (HTTP %d): %s", e.StatusCode, e.Body)
	}

	messages := make([]string, 0, len(e.Errors))
	for _, d := range e.Errors {
		msg := d.Code
		if d.Detail != "" {
			msg += ": " + d.Detail
		}
		if d.Source.Parameter != "" {
			msg += " (parameter: " + d.Source.Parameter + ")"
		}
		messages = append(messages, msg)
	}

	return fmt.Sprintf("NextDNS API error (HTTP %d): %s", e.StatusCode, strings.Join(messages, "; "))
}

// response is the envelope NextDNS wraps every answer in.
type response struct {
	Data   json.RawMessage `json:"data"`
	Errors []ErrorDetail   `json:"errors"`
}

// Do sends a request to path (for example "/profiles") and decodes the
// response's "data" field into out. body and out may be nil.
func (c *Client) Do(ctx context.Context, method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("%s %s: reading response: %w", method, path, err)
	}

	// Always try to read the body, whatever the status code: NextDNS sends
	// useful errors with 4xx and 5xx, and sometimes with HTTP 200 too.
	var parsed response
	parseErr := json.Unmarshal(raw, &parsed)

	if len(parsed.Errors) > 0 {
		return &APIError{StatusCode: res.StatusCode, Errors: parsed.Errors}
	}
	if res.StatusCode >= 400 {
		return &APIError{StatusCode: res.StatusCode, Body: strings.TrimSpace(string(raw))}
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if parseErr != nil {
		return fmt.Errorf("%s %s: decoding response: %w", method, path, parseErr)
	}
	if err := json.Unmarshal(parsed.Data, out); err != nil {
		return fmt.Errorf("%s %s: decoding response data: %w", method, path, err)
	}

	return nil
}
