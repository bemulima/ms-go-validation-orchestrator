package engines

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

const maxPinnedSnapshotResponseBytes int64 = 4 << 20

type PinnedSnapshotHTTPReader struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewPinnedSnapshotReader(baseURL, token string, client *http.Client) PinnedSnapshotHTTPReader {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return PinnedSnapshotHTTPReader{baseURL: strings.TrimRight(baseURL, "/"), token: token, client: client}
}

type pinnedSnapshotContentV2 struct {
	Schema          string                 `json:"schema"`
	SandboxID       string                 `json:"sandbox_id"`
	SnapshotID      string                 `json:"snapshot_id"`
	GenerationID    string                 `json:"generation_id"`
	WorkspaceDigest string                 `json:"workspace_digest"`
	Revision        int64                  `json:"revision"`
	FileCount       int                    `json:"file_count"`
	PinnedAt        string                 `json:"pinned_at"`
	Files           []domain.WorkspaceFile `json:"files"`
}

func (reader PinnedSnapshotHTTPReader) ReadPinnedSnapshot(
	ctx context.Context,
	ref domain.PracticeSnapshotRefV2,
) (domain.PinnedSnapshotV2, error) {
	if err := domain.ValidatePracticeSnapshotRefV2(ref); err != nil {
		return domain.PinnedSnapshotV2{}, fmt.Errorf("%w: invalid requested snapshot reference", domain.ErrPinnedSnapshotCorrupt)
	}
	if reader.baseURL == "" || reader.token == "" {
		return domain.PinnedSnapshotV2{}, domain.ErrPinnedSnapshotUnavailable
	}
	endpoint := reader.baseURL + "/internal/v2/sandboxes/" + url.PathEscape(ref.SandboxID) + "/snapshots/" + url.PathEscape(ref.SnapshotID)
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return domain.PinnedSnapshotV2{}, fmt.Errorf("%w: invalid Sandbox URL", domain.ErrPinnedSnapshotUnavailable)
	}
	query := parsedURL.Query()
	query.Set("generation_id", ref.GenerationID)
	query.Set("workspace_digest", ref.WorkspaceDigest)
	parsedURL.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return domain.PinnedSnapshotV2{}, fmt.Errorf("%w: create Sandbox snapshot request", domain.ErrPinnedSnapshotUnavailable)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Internal-Token", reader.token)
	response, err := reader.client.Do(request)
	if err != nil {
		return domain.PinnedSnapshotV2{}, fmt.Errorf("%w: Sandbox snapshot request failed: %v", domain.ErrPinnedSnapshotUnavailable, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxPinnedSnapshotResponseBytes+1))
	if err != nil {
		return domain.PinnedSnapshotV2{}, fmt.Errorf("%w: read Sandbox snapshot response", domain.ErrPinnedSnapshotUnavailable)
	}
	if int64(len(body)) > maxPinnedSnapshotResponseBytes {
		return domain.PinnedSnapshotV2{}, fmt.Errorf("%w: Sandbox snapshot response exceeds size limit", domain.ErrPinnedSnapshotCorrupt)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		if isUnavailableSnapshotStatus(response.StatusCode) {
			return domain.PinnedSnapshotV2{}, fmt.Errorf("%w: Sandbox snapshot returned status %d", domain.ErrPinnedSnapshotUnavailable, response.StatusCode)
		}
		return domain.PinnedSnapshotV2{}, fmt.Errorf("%w: Sandbox snapshot returned status %d", domain.ErrPinnedSnapshotCorrupt, response.StatusCode)
	}

	var content pinnedSnapshotContentV2
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&content); err != nil {
		return domain.PinnedSnapshotV2{}, fmt.Errorf("%w: decode Sandbox snapshot response: %v", domain.ErrPinnedSnapshotCorrupt, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return domain.PinnedSnapshotV2{}, fmt.Errorf("%w: Sandbox snapshot response has trailing JSON", domain.ErrPinnedSnapshotCorrupt)
	}
	if content.Schema != domain.PracticePinnedSnapshotContentSchemaV2 || content.SandboxID != ref.SandboxID || content.SnapshotID != ref.SnapshotID || content.GenerationID != ref.GenerationID {
		return domain.PinnedSnapshotV2{}, domain.ErrPinnedSnapshotCorrupt
	}
	if content.WorkspaceDigest != ref.WorkspaceDigest {
		return domain.PinnedSnapshotV2{}, domain.ErrPinnedSnapshotDigestMismatch
	}
	if content.Revision < 1 || content.FileCount < 0 || content.Files == nil || content.FileCount != len(content.Files) {
		return domain.PinnedSnapshotV2{}, domain.ErrPinnedSnapshotCorrupt
	}
	if _, err := time.Parse(time.RFC3339Nano, content.PinnedAt); err != nil {
		return domain.PinnedSnapshotV2{}, domain.ErrPinnedSnapshotCorrupt
	}
	files := append([]domain.WorkspaceFile(nil), content.Files...)
	sort.Slice(files, func(left, right int) bool { return files[left].Path < files[right].Path })
	digest, err := domain.WorkspaceFilesDigestV2(files)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidRequest) {
			return domain.PinnedSnapshotV2{}, fmt.Errorf("%w: invalid Sandbox file path", domain.ErrPinnedSnapshotCorrupt)
		}
		return domain.PinnedSnapshotV2{}, fmt.Errorf("%w: verify Sandbox snapshot files", domain.ErrPinnedSnapshotCorrupt)
	}
	if digest != ref.WorkspaceDigest {
		return domain.PinnedSnapshotV2{}, domain.ErrPinnedSnapshotDigestMismatch
	}
	return domain.PinnedSnapshotV2{
		Ref:      ref,
		Revision: content.Revision,
		Files:    files,
	}, nil
}

func isUnavailableSnapshotStatus(status int) bool {
	return status == http.StatusNotFound || status == http.StatusUnauthorized || status == http.StatusForbidden || status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}
