// Package api contains a minimal, self-contained HTTP client for the Hydden
// discovery platform. It is a focused port of control's hydden HTTP client
// (control/backend/internal/features/integrations/hydden/http_client.go),
// stripped of control's SettingsProvider / tenant-context coupling: the
// connector supplies the base URL via feature options and the client
// credentials via a standard mesh Grant credential, and this client manages
// its own bearer token.
package api

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Hydden API endpoints. These opaque ssrmquery/search ids are copied verbatim
// from control's hydden client and must match the discovery deployment.
const (
	endpointAuth             = "/auth/api"
	endpointAccounts         = "/api/v1/global/ssrmquery/ASpnJ4bLpFRGBZxEwAEPullOFx5"
	endpointGroups           = "/api/v1/global/search/5giWu96fvwE0N3LVgm60eKfI6X6"
	endpointOwners           = "/api/v1/global/ssrmquery/UVYaMSAx8evNujhC75QELLRej2T"
	endpointOwnerAccounts    = "/api/v1/global/ssrmquery/DUrG0M5i1MYn0H99KwSpezBqLtt"
	endpointGroupMemberships = "/api/v1/global/ssrmquery/W8fSFbTri7TqbXWgdZVpBjLZMNn"
	endpointApplicationRoles = "/api/v1/global/ssrmquery/JipWtKNEWU2BrJN2YY6TOVZvl2N"
	endpointAccountAppRoles  = "/api/v1/global/search/6jZNu3bAmCBJ5rZtN6V1FDQN6ms"

	defaultPageLimit = 1000
	tokenSafetyTTL   = 5 * time.Minute
	requestTimeout   = 60 * time.Second
)

// Row is a single result row from a Hydden search/ssrmquery response.
type Row = map[string]any

// PageFunc receives one page of rows, the 1-based page number, and the
// API-reported total count. Return a non-nil error to stop pagination.
type PageFunc func(page []Row, pageNum, totalCount int) error

// Client is a lightweight Hydden discovery API client.
type Client struct {
	baseURL      string
	clientID     string
	clientSecret string
	httpClient   *http.Client
	logger       *slog.Logger

	maxRetries     int
	initialBackoff time.Duration

	mu          sync.RWMutex
	token       string
	tokenExpiry time.Time
}

// NewClient builds a Hydden client. baseURL is the discovery base URL;
// clientID/clientSecret are the discovery API credentials (from the mesh
// Grant credential).
func NewClient(baseURL, clientID, clientSecret string, logger *slog.Logger) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableCompression = false
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		baseURL:        strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		clientID:       clientID,
		clientSecret:   clientSecret,
		httpClient:     &http.Client{Transport: transport, Timeout: requestTimeout},
		logger:         logger,
		maxRetries:     3,
		initialBackoff: time.Second,
	}
}

type searchResponse struct {
	Rows       []any `json:"rows"`
	TotalCount int   `json:"totalCount"`
}

// getToken returns a cached bearer token or fetches a new one via /auth/api.
func (c *Client) getToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		tok := c.token
		c.mu.RUnlock()
		return tok, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return c.token, nil
	}

	if c.baseURL == "" {
		return "", fmt.Errorf("discovery base_url is required")
	}
	if c.clientID == "" || c.clientSecret == "" {
		return "", fmt.Errorf("discovery client credentials are required")
	}

	reqBody, _ := json.Marshal(map[string]string{"id": c.clientID, "secret": c.clientSecret})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+endpointAuth, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.executeWithRetry(ctx, req, reqBody)
	if err != nil {
		return "", fmt.Errorf("discovery auth request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("discovery auth failed with status %d: %s", resp.StatusCode, string(b))
	}

	var authResp struct {
		AccessToken string `json:"accessToken"`
		Token       string `json:"token"`
		ExpiresIn   int    `json:"expiresIn"`
		ExpiresAt   int64  `json:"expiresAt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("decode discovery auth response: %w", err)
	}
	token := authResp.AccessToken
	if token == "" {
		token = authResp.Token
	}
	if token == "" {
		return "", fmt.Errorf("no token in discovery auth response")
	}

	switch {
	case authResp.ExpiresAt > 0:
		c.tokenExpiry = time.Unix(authResp.ExpiresAt, 0).Add(-tokenSafetyTTL)
	case authResp.ExpiresIn > 0:
		c.tokenExpiry = time.Now().Add(time.Duration(authResp.ExpiresIn)*time.Second - tokenSafetyTTL)
	default:
		c.tokenExpiry = time.Now().Add(55 * time.Minute)
	}
	c.token = token
	return token, nil
}

// invalidateToken drops the cached token so the next call re-authenticates.
func (c *Client) invalidateToken() {
	c.mu.Lock()
	c.token = ""
	c.mu.Unlock()
}

// doAuthenticated sends an authenticated request, refreshing the token once on 401.
func (c *Client) doAuthenticated(ctx context.Context, method, url string, body []byte) (*http.Response, error) {
	for attempt := 0; attempt < 2; attempt++ {
		token, err := c.getToken(ctx)
		if err != nil {
			return nil, err
		}
		var r io.Reader
		if body != nil {
			r = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, r)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		req.Header.Set("User-Agent", "mesh-discovery/1.0")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.executeWithRetry(ctx, req, body)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 {
			_ = resp.Body.Close()
			c.invalidateToken()
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("discovery: exhausted auth retry attempts")
}

// executeWithRetry performs an HTTP request with exponential backoff on
// transient failures. body is the buffered request body so retries can resend.
func (c *Client) executeWithRetry(ctx context.Context, req *http.Request, body []byte) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 && body != nil {
			req.Body = io.NopCloser(bytes.NewReader(body))
		}
		resp, err := c.httpClient.Do(req)
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		if err == nil && status >= 200 && status < 300 {
			return resp, nil
		}
		lastErr = err

		retry := err != nil
		switch status {
		case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			retry = true
		}
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if !retry || attempt == c.maxRetries {
			if lastErr != nil {
				return nil, lastErr
			}
			// Non-retryable HTTP error: re-issue once to hand the caller the
			// response/body for status inspection.
			if body != nil {
				req.Body = io.NopCloser(bytes.NewReader(body))
			}
			return c.httpClient.Do(req)
		}

		backoff := c.initialBackoff * (1 << uint(attempt))
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, lastErr
}

// decodeBody returns a reader that transparently decompresses gzip responses.
func decodeBody(resp *http.Response) (io.ReadCloser, error) {
	if resp.Header.Get("Content-Encoding") == "gzip" {
		r, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("create gzip reader: %w", err)
		}
		return r, nil
	}
	return resp.Body, nil
}

// pageBody builds a paginated search request body. The viewTime key casing is
// preserved per-endpoint to match the discovery API contract.
func pageBody(viewTimeKey, viewTimeVal string, offset, limit int) map[string]any {
	b := map[string]any{
		"offset":       offset,
		"limit":        limit,
		"filterModel":  map[string]any{},
		"sortModel":    []any{},
		"rowGroupCols": []any{},
		"groupKeys":    []any{},
	}
	b[viewTimeKey] = viewTimeVal
	return b
}

// forEachPage drives offset/limit pagination against a search endpoint,
// invoking cb for each non-empty page.
func (c *Client) forEachPage(ctx context.Context, endpoint, viewTimeKey, viewTimeVal string, cb PageFunc) error {
	url := c.baseURL + endpoint
	offset, pageNum, fetched := 0, 0, 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		bodyBytes, _ := json.Marshal(pageBody(viewTimeKey, viewTimeVal, offset, defaultPageLimit))
		resp, err := c.doAuthenticated(ctx, http.MethodPost, url, bodyBytes)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return fmt.Errorf("discovery %s returned status %d: %s", endpoint, resp.StatusCode, string(b))
		}
		reader, err := decodeBody(resp)
		if err != nil {
			_ = resp.Body.Close()
			return err
		}
		var sr searchResponse
		decErr := json.NewDecoder(reader).Decode(&sr)
		if reader != resp.Body {
			_ = reader.Close()
		}
		_ = resp.Body.Close()
		if decErr != nil {
			return fmt.Errorf("decode discovery %s response: %w", endpoint, decErr)
		}

		page := toRows(sr.Rows)
		pageNum++
		fetched += len(page)
		if len(page) > 0 {
			if err := cb(page, pageNum, sr.TotalCount); err != nil {
				return err
			}
		}

		if len(sr.Rows) < defaultPageLimit {
			return nil
		}
		if sr.TotalCount > 0 && fetched >= sr.TotalCount {
			return nil
		}
		offset += defaultPageLimit
	}
}

func toRows(raw []any) []Row {
	rows := make([]Row, 0, len(raw))
	for _, r := range raw {
		if m, ok := r.(map[string]any); ok {
			rows = append(rows, m)
		}
	}
	return rows
}

// ForEachAccountPage streams account rows.
func (c *Client) ForEachAccountPage(ctx context.Context, cb PageFunc) error {
	return c.forEachPage(ctx, endpointAccounts, "ViewTime", "0", cb)
}

// ForEachGroupPage streams group rows.
func (c *Client) ForEachGroupPage(ctx context.Context, cb PageFunc) error {
	return c.forEachPage(ctx, endpointGroups, "ViewTime", "0", cb)
}

// ForEachGroupMembershipPage streams account<->group membership rows.
func (c *Client) ForEachGroupMembershipPage(ctx context.Context, cb PageFunc) error {
	return c.forEachPage(ctx, endpointGroupMemberships, "viewTime", time.Now().UTC().Format(time.RFC3339), cb)
}

// ForEachOwnerPage streams owner (identity) rows.
func (c *Client) ForEachOwnerPage(ctx context.Context, cb PageFunc) error {
	return c.forEachPage(ctx, endpointOwners, "viewTime", "0", cb)
}

// ForEachOwnerAccountPage streams owner<->account relationship rows.
func (c *Client) ForEachOwnerAccountPage(ctx context.Context, cb PageFunc) error {
	return c.forEachPage(ctx, endpointOwnerAccounts, "viewTime", time.Now().UTC().Format(time.RFC3339), cb)
}

// ForEachApplicationRolePage streams application-role (entitlement) rows.
func (c *Client) ForEachApplicationRolePage(ctx context.Context, cb PageFunc) error {
	return c.forEachPage(ctx, endpointApplicationRoles, "ViewTime", "0", cb)
}

// GetAccountAppRoles returns all app-role memberships for a single account.
// The endpoint returns the full per-account result set in one shot, so no
// pagination is needed. accountExternalID is the Hydden account id.
func (c *Client) GetAccountAppRoles(ctx context.Context, accountExternalID string) ([]Row, error) {
	bodyBytes, _ := json.Marshal(map[string]any{"Id": accountExternalID})
	resp, err := c.doAuthenticated(ctx, http.MethodPost, c.baseURL+endpointAccountAppRoles, bodyBytes)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("discovery GetAccountAppRoles returned status %d: %s", resp.StatusCode, string(b))
	}
	reader, err := decodeBody(resp)
	if err != nil {
		_ = resp.Body.Close()
		return nil, err
	}
	var sr searchResponse
	decErr := json.NewDecoder(reader).Decode(&sr)
	if reader != resp.Body {
		_ = reader.Close()
	}
	_ = resp.Body.Close()
	if decErr != nil {
		return nil, fmt.Errorf("decode discovery GetAccountAppRoles response: %w", decErr)
	}
	return toRows(sr.Rows), nil
}
