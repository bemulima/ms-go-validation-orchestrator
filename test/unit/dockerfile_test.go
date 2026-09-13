package unit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDockerfileBuildsNativelyForTargetPlatform(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "Dockerfile"))
	if err != nil {
		t.Fatalf("read Dockerfile: %v", err)
	}
	dockerfile := string(data)
	for _, required := range []string{
		"FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build\n",
		"ARG TARGETOS\n",
		"ARG TARGETARCH\n",
		"RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/ms-go-validation-orchestrator ./cmd/ms-go-validation-orchestrator\n",
		"FROM alpine:3.20\n",
		"COPY --from=build /out/ms-go-validation-orchestrator /usr/local/bin/ms-go-validation-orchestrator\n",
		"ENTRYPOINT [\"ms-go-validation-orchestrator\"]\n",
	} {
		if !strings.Contains(dockerfile, required) {
			t.Errorf("Dockerfile is missing build/runtime invariant: %q", required)
		}
	}
	if strings.Contains(dockerfile, "GOARCH=amd64") {
		t.Error("compiler target must use TARGETARCH rather than a hard-coded architecture")
	}
}
