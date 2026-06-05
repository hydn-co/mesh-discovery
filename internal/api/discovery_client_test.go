package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestServer stands up a fake Hydden discovery API. authHits and the path
// handlers let tests assert on token reuse and request routing.
func newTestServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) (*Client, *int) {
	t.Helper()
	authHits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == endpointAuth {
			authHits++
			_ = json.NewEncoder(w).Encode(map[string]any{"accessToken": "tok", "expiresIn": 3600})
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			http.Error(w, "missing bearer", http.StatusUnauthorized)
			return
		}
		handler(w, r)
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "cid", "sec", nil)
	return client, &authHits
}

func TestShouldAuthenticateThenStreamAccountPages(t *testing.T) {
	client, authHits := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))
		// Single short page (< page limit) terminates pagination.
		_ = json.NewEncoder(w).Encode(searchResponse{
			Rows:       []any{map[string]any{"Id": "acc-1"}, map[string]any{"Id": "acc-2"}},
			TotalCount: 2,
		})
	})

	var got []Row
	err := client.ForEachAccountPage(context.Background(), func(page []Row, pageNum, total int) error {
		assert.Equal(t, 1, pageNum)
		assert.Equal(t, 2, total)
		got = append(got, page...)
		return nil
	})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "acc-1", got[0]["Id"])
	// The bearer token is cached and reused across the auth + data calls.
	assert.Equal(t, 1, *authHits)
}

func TestShouldStreamFetchedEntities(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, endpointDatastoreFetch, r.URL.Path)
		_, _ = io.WriteString(w, `[
			{"id":"acc-1","type":"principal.account.user","entity":{"dept":"IT"}},
			{"id":"acc-2","type":"principal.account.user","tombstoned":true}
		]`)
	})

	var got []*FetchedEntity
	err := client.FetchEntities(context.Background(), "ds1", "principal.account.user", func(e *FetchedEntity) error {
		got = append(got, e)
		return nil
	})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "acc-1", got[0].ID)
	assert.Equal(t, "IT", got[0].Entity["dept"])
	assert.True(t, got[1].Tombstoned)
}

func TestShouldGetAccountDetailsType(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.True(t, strings.HasPrefix(r.URL.Path, "/internal/v1/datastore/entity/"))
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "principal.account.user"})
	})

	resp, err := client.GetAccountDetails(context.Background(), "ds1", "acc-1")
	require.NoError(t, err)
	assert.Equal(t, "principal.account.user", resp["type"])
}

func TestShouldSurfaceNonOKStatus(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})

	err := client.FetchEntities(context.Background(), "ds1", "t", func(*FetchedEntity) error { return nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}
