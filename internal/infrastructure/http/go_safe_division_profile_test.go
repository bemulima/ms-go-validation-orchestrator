package engines

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

func TestGoSafeDivisionCapabilityRequestsExactProfile(t *testing.T) {
	const profile = "go-safe-division.v1"
	seen := false
	c := NewGoTestsClient("http://unit.invalid", "owner-token")
	c.HTTP = &http.Client{Transport: goProviderRoundTrip(func(r *http.Request) (*http.Response, error) {
		seen = true
		if r.URL.Path != "/internal/go-tests/capability" || r.URL.Query().Get("profile_id") != profile || r.Header.Get("Authorization") != "Bearer owner-token" {
			t.Errorf("wrong capability request: path=%q profile=%q", r.URL.Path, r.URL.Query().Get("profile_id"))
		}
		b, _ := json.Marshal(domain.GoTestManifest{Schema: "go-test-capability.v1", ProfileID: profile, BundleVersion: "safe-division.v1"})
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(b))}, nil
	})}
	m, err := c.CapabilityForProfile(context.Background(), profile)
	if err != nil || !seen || m.ProfileID != profile || m.BundleVersion != "safe-division.v1" {
		t.Fatalf("capability=%+v seen=%v err=%v", m, seen, err)
	}
}
