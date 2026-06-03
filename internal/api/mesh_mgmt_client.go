package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MeshMgmtClient is a minimal client for mesh-core's management API, used by the
// orchestrator to register derived per-datasource providers/connectors. It
// authenticates with a mesh system credential via the client_credentials grant,
// mirroring control's mesh.Client (control/backend/internal/mesh/client.go).
type MeshMgmtClient struct {
	baseURL      string
	clientID     string
	clientSecret string
	httpClient   *http.Client

	mu          sync.RWMutex
	token       string
	tokenExpiry time.Time
}

// systemTokenTenantID is the well-known tenant used for system-principal tokens
// (matches control's meshSystemTokenTenantID).
const systemTokenTenantID = "00000000-0000-0000-0000-000000000000"

// NewMeshMgmtClient builds a mesh-core management client.
func NewMeshMgmtClient(baseURL, clientID, clientSecret string) *MeshMgmtClient {
	return &MeshMgmtClient{
		baseURL:      strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: requestTimeout},
	}
}

// DeriveProviderFeature describes a feature on a derived provider.
type DeriveProviderFeature struct {
	DisplayName    string          `json:"display_name"`
	Description    string          `json:"description,omitempty"`
	Type           string          `json:"type"`
	ResumeBehavior string          `json:"resume_behavior"`
	Schedulable    bool            `json:"schedulable"`
	OptionsSchema  json.RawMessage `json:"options_schema,omitempty"`
	Requirements   []string        `json:"requirements,omitempty"`
}

// DeriveProviderRequest registers a provider that reuses an existing executable.
type DeriveProviderRequest struct {
	DisplayName       string                           `json:"display_name"`
	Description       string                           `json:"description,omitempty"`
	ExecutableName    string                           `json:"executable_name"`
	ManifestVersion   string                           `json:"manifest_version"`
	ManifestSourceRef string                           `json:"manifest_source_ref"`
	Features          map[string]DeriveProviderFeature `json:"features"`
}

// DeriveProvider creates/updates a derived provider that reuses ExecutableName.
func (c *MeshMgmtClient) DeriveProvider(
	ctx context.Context,
	tenantID, providerID uuid.UUID,
	req DeriveProviderRequest,
) error {
	path := fmt.Sprintf("/api/v1/tenants/%s/providers/%s/derive", tenantID, providerID)
	return c.do(ctx, http.MethodPost, path, req, nil)
}

// PutConnectorRequest creates/updates a connector instance of a provider.
type PutConnectorRequest struct {
	Name         string    `json:"name"`
	ProviderID   uuid.UUID `json:"provider_id"`
	Requirements []string  `json:"requirements,omitempty"`
	IsSystem     bool      `json:"is_system,omitempty"`
}

// PutConnector creates/updates a connector.
func (c *MeshMgmtClient) PutConnector(
	ctx context.Context,
	tenantID, connectorID uuid.UUID,
	req PutConnectorRequest,
) error {
	path := fmt.Sprintf("/api/v1/tenants/%s/connectors/%s", tenantID, connectorID)
	return c.do(ctx, http.MethodPut, path, req, nil)
}

// PutConnectorFeatureRequest bakes options (and an optional secret) into a
// connector's feature.
type PutConnectorFeatureRequest struct {
	Enabled  bool            `json:"enabled"`
	SecretID *uuid.UUID      `json:"secret_id,omitempty"`
	Options  json.RawMessage `json:"options,omitempty"`
}

// PutConnectorFeature configures a connector feature with baked options.
func (c *MeshMgmtClient) PutConnectorFeature(
	ctx context.Context,
	tenantID, connectorID, featureID uuid.UUID,
	req PutConnectorFeatureRequest,
) error {
	path := fmt.Sprintf("/api/v1/tenants/%s/connectors/%s/features/%s", tenantID, connectorID, featureID)
	return c.do(ctx, http.MethodPut, path, req, nil)
}

// RequestInvocation queues a feature run.
func (c *MeshMgmtClient) RequestInvocation(
	ctx context.Context,
	tenantID, connectorID, featureID uuid.UUID,
	runSequence int,
) error {
	path := fmt.Sprintf(
		"/api/v1/tenants/%s/connectors/%s/features/%s/invocations/%d/request",
		tenantID, connectorID, featureID, runSequence,
	)
	return c.do(ctx, http.MethodPost, path, map[string]any{}, nil)
}

type tokenRequest struct {
	GrantType    string `json:"grant_type"`
	TenantID     string `json:"tenant_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func (c *MeshMgmtClient) getToken(ctx context.Context) (string, error) {
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
	if c.baseURL == "" || c.clientID == "" || c.clientSecret == "" {
		return "", fmt.Errorf(
			"mesh-core base URL and client credentials are required (set MESH_CORE_BASE_URL, MESH_CLIENT_ID, MESH_CLIENT_SECRET)",
		)
	}

	body, _ := json.Marshal(tokenRequest{
		GrantType:    "client_credentials",
		TenantID:     systemTokenTenantID,
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/auth/token", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("mesh-core token request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("mesh-core token request returned %d: %s", resp.StatusCode, string(b))
	}
	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("decode mesh-core token response: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("empty access_token in mesh-core token response")
	}
	ttl := time.Duration(tr.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	c.token = tr.AccessToken
	c.tokenExpiry = time.Now().Add(ttl - tokenSafetyTTL)
	return c.token, nil
}

func (c *MeshMgmtClient) do(ctx context.Context, method, path string, body, dest any) error {
	token, err := c.getToken(ctx)
	if err != nil {
		return err
	}
	var reader io.Reader
	if body != nil {
		raw, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return fmt.Errorf("marshal request body: %w", marshalErr)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("mesh-core %s %s failed: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mesh-core %s %s returned %d: %s", method, path, resp.StatusCode, string(b))
	}
	if dest != nil {
		return json.NewDecoder(resp.Body).Decode(dest)
	}
	return nil
}
