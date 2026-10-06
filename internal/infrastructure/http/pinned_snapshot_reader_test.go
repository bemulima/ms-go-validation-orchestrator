package engines

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestPinnedSnapshotReaderRequestsExactPinnedGenerationAndVerifiesDigest(t *testing.T) {
	t.Parallel()
	files := []domain.WorkspaceFile{{Path: "a.txt", Content: "A"}, {Path: "b.txt", Content: "B"}}
	ref := domain.PracticeSnapshotRefV2{SandboxID: "11111111-1111-4111-8111-111111111111", SnapshotID: "snapshot-1", GenerationID: "gen-1", WorkspaceDigest: digestFiles(t, files)}
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if request.URL.Path != "/internal/v2/sandboxes/"+ref.SandboxID+"/snapshots/"+ref.SnapshotID {
			t.Errorf("path = %s", request.URL.Path)
		}
		if request.URL.Query().Get("generation_id") != ref.GenerationID || request.URL.Query().Get("workspace_digest") != ref.WorkspaceDigest {
			t.Errorf("query = %s", request.URL.RawQuery)
		}
		if request.Header.Get("X-Internal-Token") != "sandbox-token" {
			t.Errorf("sandbox token missing")
		}
		payload, _ := json.Marshal(map[string]any{
			"schema": "practice-pinned-snapshot-content.v2", "sandbox_id": ref.SandboxID,
			"snapshot_id": ref.SnapshotID, "generation_id": ref.GenerationID,
			"workspace_digest": ref.WorkspaceDigest, "revision": 9, "file_count": len(files),
			"pinned_at": "2026-10-02T00:00:00Z", "files": files,
		})
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(payload)))}, nil
	})}

	reader := NewPinnedSnapshotReader("http://sandbox.test", "sandbox-token", client)
	got, err := reader.ReadPinnedSnapshot(context.Background(), ref)
	if err != nil {
		t.Fatalf("ReadPinnedSnapshot() error = %v", err)
	}
	if got.Ref != ref || !reflect.DeepEqual(got.Files, files) || got.Revision != 9 {
		t.Fatalf("snapshot = %+v", got)
	}
}

func TestPinnedSnapshotReaderRejectsIdentityAndDigestMismatch(t *testing.T) {
	t.Parallel()
	files := []domain.WorkspaceFile{{Path: "main.go", Content: "package main"}}
	ref := domain.PracticeSnapshotRefV2{SandboxID: "11111111-1111-4111-8111-111111111111", SnapshotID: "snapshot-1", GenerationID: "gen-1", WorkspaceDigest: digestFiles(t, files)}
	for _, test := range []struct {
		name        string
		mutate      func(map[string]any)
		wantFailure error
	}{
		{name: "wrong identity", mutate: func(payload map[string]any) { payload["generation_id"] = "gen-other" }, wantFailure: domain.ErrPinnedSnapshotCorrupt},
		{name: "wrong digest", mutate: func(payload map[string]any) {
			payload["workspace_digest"] = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		}, wantFailure: domain.ErrPinnedSnapshotDigestMismatch},
		{name: "content digest mismatch", mutate: func(payload map[string]any) {
			payload["files"] = []map[string]string{{"path": "main.go", "content": "package hacked"}}
		}, wantFailure: domain.ErrPinnedSnapshotDigestMismatch},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			client := &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				payload := map[string]any{"schema": "practice-pinned-snapshot-content.v2", "sandbox_id": ref.SandboxID, "snapshot_id": ref.SnapshotID, "generation_id": ref.GenerationID, "workspace_digest": ref.WorkspaceDigest, "revision": 1, "file_count": len(files), "pinned_at": "2026-10-02T00:00:00Z", "files": files}
				test.mutate(payload)
				encoded, _ := json.Marshal(payload)
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(encoded)))}, nil
			})}
			reader := NewPinnedSnapshotReader("http://sandbox.test", "token", client)
			_, err := reader.ReadPinnedSnapshot(context.Background(), ref)
			if !errors.Is(err, test.wantFailure) {
				t.Fatalf("error = %v, want errors.Is(_, %v)", err, test.wantFailure)
			}
		})
	}
}

func TestPinnedSnapshotReaderNormalizesUnavailableAndInvalidResponses(t *testing.T) {
	t.Parallel()
	ref := domain.PracticeSnapshotRefV2{SandboxID: "11111111-1111-4111-8111-111111111111", SnapshotID: "snapshot-1", GenerationID: "gen-1", WorkspaceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	for _, test := range []struct {
		name        string
		status      int
		wantFailure error
	}{
		{name: "unauthorized dependency", status: http.StatusUnauthorized, wantFailure: domain.ErrPinnedSnapshotUnavailable},
		{name: "server unavailable", status: http.StatusServiceUnavailable, wantFailure: domain.ErrPinnedSnapshotUnavailable},
		{name: "identity conflict", status: http.StatusConflict, wantFailure: domain.ErrPinnedSnapshotCorrupt},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: test.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":"sandbox_identity_mismatch"}`))}, nil
			})}
			reader := NewPinnedSnapshotReader("http://sandbox.test", "token", client)
			_, err := reader.ReadPinnedSnapshot(context.Background(), ref)
			if !errors.Is(err, test.wantFailure) {
				t.Fatalf("error = %v, want errors.Is(_, %v)", err, test.wantFailure)
			}
		})
	}
}

func TestPinnedSnapshotReaderRequiresPresentNonNegativeRevision(t *testing.T) {
	t.Parallel()
	files := []domain.WorkspaceFile{{Path: "index.html", Content: "<h1>Frozen genesis</h1>"}}
	ref := domain.PracticeSnapshotRefV2{SandboxID: "11111111-1111-4111-8111-111111111111", SnapshotID: "snapshot-genesis", GenerationID: "gen-genesis", WorkspaceDigest: digestFiles(t, files)}
	for _, test := range []struct {
		name     string
		revision string
		valid    bool
		want     int64
	}{
		{name: "explicit genesis zero", revision: `0`, valid: true},
		{name: "positive", revision: `9`, valid: true, want: 9},
		{name: "missing"},
		{name: "null", revision: `null`},
		{name: "negative", revision: `-1`},
		{name: "string", revision: `"0"`},
		{name: "fraction", revision: `0.5`},
		{name: "boolean", revision: `false`},
		{name: "overflow", revision: `9223372036854775808`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				payload := map[string]any{"schema": "practice-pinned-snapshot-content.v2", "sandbox_id": ref.SandboxID, "snapshot_id": ref.SnapshotID, "generation_id": ref.GenerationID, "workspace_digest": ref.WorkspaceDigest, "file_count": len(files), "pinned_at": "2026-10-02T00:00:00Z", "files": files}
				if test.revision != "" {
					payload["revision"] = json.RawMessage(test.revision)
				}
				encoded, err := json.Marshal(payload)
				if err != nil {
					t.Fatal(err)
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(encoded)))}, nil
			})}
			got, err := NewPinnedSnapshotReader("http://sandbox.test", "token", client).ReadPinnedSnapshot(context.Background(), ref)
			if !test.valid {
				if !errors.Is(err, domain.ErrPinnedSnapshotCorrupt) {
					t.Fatalf("error = %v, want corrupt revision", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("valid retained revision rejected: %v", err)
			}
			if got.Ref != ref || got.Revision != test.want || !reflect.DeepEqual(got.Files, files) {
				t.Fatalf("retained snapshot identity, revision or content changed")
			}
		})
	}
}

func digestFiles(t *testing.T, files []domain.WorkspaceFile) string {
	t.Helper()
	digest, err := domain.WorkspaceFilesDigestV2(files)
	if err != nil {
		t.Fatalf("WorkspaceFilesDigestV2(): %v", err)
	}
	return digest
}
