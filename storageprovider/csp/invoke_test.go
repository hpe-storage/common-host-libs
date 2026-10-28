// Copyright 2026 Hewlett Packard Enterprise Development LP
package csp

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hpe-storage/common-host-libs/connectivity"
	"github.com/hpe-storage/common-host-libs/storageprovider"
	"github.com/stretchr/testify/assert"
)

const (
	unauthorizedBody = `{"errors":[{"code":"Unauthorized","message":"session renewal required"}]}`
	volumeGroupsPath = "/containers/v1/volume_groups"
	tokensPath       = "/containers/v1/tokens"
)

type scriptedReply struct {
	status int
	body   string
}

// fakeCSP replays scripted replies for one path (the last reply repeats) and counts calls per path.
type fakeCSP struct {
	mu      sync.Mutex
	path    string
	replies []scriptedReply
	calls   map[string]int
}

func (f *fakeCSP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls[r.URL.Path]++
	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case tokensPath:
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"session_token":"new-token"}`))
	case f.path:
		idx := f.calls[r.URL.Path] - 1
		if idx >= len(f.replies) {
			idx = len(f.replies) - 1
		}
		w.WriteHeader(f.replies[idx].status)
		_, _ = w.Write([]byte(f.replies[idx].body))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func newTestProvider(t *testing.T, path string, authToken string, replies ...scriptedReply) (*ContainerStorageProvider, *fakeCSP) {
	fake := &fakeCSP{path: path, replies: replies, calls: map[string]int{}}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)

	provider := &ContainerStorageProvider{
		Credentials: &storageprovider.Credentials{Username: "user", Password: "pass", Backend: "10.0.0.1"},
		Client:      connectivity.NewHTTPClient(srv.URL),
		AuthToken:   authToken,
	}
	return provider, fake
}

func TestInvoke_401ThenCreated_NoError(t *testing.T) {
	provider, fake := newTestProvider(t, volumeGroupsPath, "stale-token",
		scriptedReply{http.StatusUnauthorized, unauthorizedBody},
		scriptedReply{http.StatusCreated, `{"id":"volumegroup-1","name":"volumegroup-1"}`},
	)

	vg, err := provider.CreateVolumeGroup("volumegroup-1", "desc", nil)

	assert.NoError(t, err)
	if assert.NotNil(t, vg) {
		assert.Equal(t, "volumegroup-1", vg.ID)
	}
	assert.Equal(t, 1, fake.calls[tokensPath])
	assert.Equal(t, 2, fake.calls[volumeGroupsPath])
	assert.Equal(t, "new-token", provider.AuthToken)
}

func TestInvoke_Two401sThenCreated_NoError(t *testing.T) {
	provider, fake := newTestProvider(t, volumeGroupsPath, "stale-token",
		scriptedReply{http.StatusUnauthorized, unauthorizedBody},
		scriptedReply{http.StatusUnauthorized, unauthorizedBody},
		scriptedReply{http.StatusCreated, `{"id":"volumegroup-1","name":"volumegroup-1"}`},
	)

	vg, err := provider.CreateVolumeGroup("volumegroup-1", "desc", nil)

	assert.NoError(t, err)
	assert.NotNil(t, vg)
	assert.Equal(t, 2, fake.calls[tokensPath])
	assert.Equal(t, 3, fake.calls[volumeGroupsPath])
}

func TestInvoke_401Then400_ReportsNewError(t *testing.T) {
	provider, _ := newTestProvider(t, volumeGroupsPath, "stale-token",
		scriptedReply{http.StatusUnauthorized, unauthorizedBody},
		scriptedReply{http.StatusBadRequest, `{"errors":[{"code":"Bad Request","message":"boom"}]}`},
	)

	vg, err := provider.CreateVolumeGroup("volumegroup-1", "desc", nil)

	assert.Nil(t, vg)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "status code 400")
		assert.Contains(t, err.Error(), "boom")
		assert.NotContains(t, err.Error(), "session renewal required")
	}
}

func TestInvoke_Persistent401_Bounded(t *testing.T) {
	provider, fake := newTestProvider(t, volumeGroupsPath, "stale-token",
		scriptedReply{http.StatusUnauthorized, unauthorizedBody},
	)

	vg, err := provider.CreateVolumeGroup("volumegroup-1", "desc", nil)

	assert.Nil(t, vg)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "status code 401")
	}
	assert.Equal(t, maxReloginAttempts, fake.calls[tokensPath])
	assert.Equal(t, maxReloginAttempts+1, fake.calls[volumeGroupsPath])
}

func TestInvoke_200_NoRelogin(t *testing.T) {
	provider, fake := newTestProvider(t, "/containers/v1/volumes/vol-1", "valid-token",
		scriptedReply{http.StatusOK, `{"id":"vol-1","name":"pvc-1"}`},
	)

	vol, err := provider.GetVolume("vol-1")

	assert.NoError(t, err)
	if assert.NotNil(t, vol) {
		assert.Equal(t, "vol-1", vol.ID)
	}
	assert.Equal(t, 0, fake.calls[tokensPath])
}

func TestInvoke_EmptyToken_LogsInFirst(t *testing.T) {
	provider, fake := newTestProvider(t, volumeGroupsPath, "",
		scriptedReply{http.StatusCreated, `{"id":"volumegroup-1","name":"volumegroup-1"}`},
	)

	vg, err := provider.CreateVolumeGroup("volumegroup-1", "desc", nil)

	assert.NoError(t, err)
	assert.NotNil(t, vg)
	assert.Equal(t, 1, fake.calls[tokensPath])
	assert.Equal(t, 1, fake.calls[volumeGroupsPath])
}
