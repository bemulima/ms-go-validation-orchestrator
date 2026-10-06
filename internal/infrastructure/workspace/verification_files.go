// Package workspace provides bounded read-only authoring fixture transport.
package workspace

import (
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

const (
	maxFiles       = 1000
	maxFileBytes   = 4 << 20
	maxTotalBytes  = 16 << 20
	maxWalkEntries = 10000
	maxWalkDepth   = 128
)

type VerificationFileReader struct{ baseDir string }

func NewVerificationFileReader(baseDir string) VerificationFileReader {
	return VerificationFileReader{baseDir: baseDir}
}

func (reader VerificationFileReader) ReadFiles(ctx context.Context, portableRoot string) ([]domain.WorkspaceFile, error) {
	if ctx.Err() != nil || portableRoot == "" || domain.ValidateSandboxWorkspaceRoot(portableRoot) != nil {
		return nil, domain.ErrVerificationWorkspace
	}
	base, err := openBase(ctx, reader.baseDir)
	if err != nil {
		return nil, domain.ErrVerificationWorkspace
	}
	defer base.Close()
	caseRoot, err := openDirectory(base, strings.TrimPrefix(portableRoot, domain.SandboxWorkspaceRoot+"/"))
	if err != nil {
		return nil, domain.ErrVerificationWorkspace
	}
	defer caseRoot.Close()
	walk := fileWalk{ctx: ctx, files: make([]domain.WorkspaceFile, 0), seen: make(map[string]struct{})}
	if err := walk.directory(caseRoot, "", 0); err != nil {
		return nil, domain.ErrVerificationWorkspace
	}
	if ctx.Err() != nil || len(walk.files) == 0 {
		return nil, domain.ErrVerificationWorkspace
	}
	sort.Slice(walk.files, func(i, j int) bool { return walk.files[i].Path < walk.files[j].Path })
	return walk.files, nil
}

// Resolve the trusted configured mount with descriptor-relative operations.
// Reject symlink components even when os.Root would safely contain them.
func openBase(ctx context.Context, baseDir string) (*os.Root, error) {
	if !filepath.IsAbs(baseDir) || filepath.Clean(baseDir) != baseDir {
		return nil, domain.ErrVerificationWorkspace
	}
	root, err := os.OpenRoot(string(filepath.Separator))
	if err != nil {
		return nil, domain.ErrVerificationWorkspace
	}
	for _, name := range strings.Split(strings.TrimPrefix(baseDir, string(filepath.Separator)), string(filepath.Separator)) {
		if ctx.Err() != nil {
			root.Close()
			return nil, domain.ErrVerificationWorkspace
		}
		if name == "" {
			continue
		}
		next, err := openDirectory(root, name)
		root.Close()
		if err != nil {
			return nil, domain.ErrVerificationWorkspace
		}
		root = next
	}
	return root, nil
}

func openDirectory(parent *os.Root, name string) (*os.Root, error) {
	before, err := parent.Lstat(name)
	if err != nil || !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
		return nil, domain.ErrVerificationWorkspace
	}
	child, err := parent.OpenRoot(name)
	if err != nil {
		return nil, domain.ErrVerificationWorkspace
	}
	opened, err := child.Stat(".")
	after, afterErr := parent.Lstat(name)
	if err != nil || afterErr != nil || !after.IsDir() || after.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, opened) || !os.SameFile(opened, after) {
		child.Close()
		return nil, domain.ErrVerificationWorkspace
	}
	return child, nil
}

type fileWalk struct {
	ctx            context.Context
	files          []domain.WorkspaceFile
	seen           map[string]struct{}
	entries, bytes int
}

func (walk *fileWalk) directory(root *os.Root, prefix string, depth int) error {
	if walk.ctx.Err() != nil || depth > maxWalkDepth {
		return domain.ErrVerificationWorkspace
	}
	directory, err := root.Open(".")
	if err != nil {
		return domain.ErrVerificationWorkspace
	}
	defer directory.Close()
	for {
		if walk.ctx.Err() != nil {
			return domain.ErrVerificationWorkspace
		}
		entries, readErr := directory.ReadDir(64)
		if readErr != nil && readErr != io.EOF {
			return domain.ErrVerificationWorkspace
		}
		for _, entry := range entries {
			walk.entries++
			if walk.ctx.Err() != nil || walk.entries > maxWalkEntries {
				return domain.ErrVerificationWorkspace
			}
			name := entry.Name()
			relative := path.Join(prefix, name)
			if !utf8.ValidString(relative) || domain.ValidateWorkspaceFilePath(relative) != nil || name == "." || name == ".." || strings.ContainsAny(name, "/\\") {
				return domain.ErrVerificationWorkspace
			}
			info, err := root.Lstat(name)
			if err != nil || info.Mode()&os.ModeSymlink != 0 {
				return domain.ErrVerificationWorkspace
			}
			if info.IsDir() {
				child, err := openDirectory(root, name)
				if err != nil {
					return domain.ErrVerificationWorkspace
				}
				err = walk.directory(child, relative, depth+1)
				child.Close()
				if err != nil {
					return domain.ErrVerificationWorkspace
				}
				continue
			}
			if !info.Mode().IsRegular() || len(walk.files) >= maxFiles {
				return domain.ErrVerificationWorkspace
			}
			if _, duplicate := walk.seen[relative]; duplicate {
				return domain.ErrVerificationWorkspace
			}
			// os.Root contains resolution but permits contained symlinks. Check
			// observed path type/identity before and after open; nonblocking
			// avoids a FIFO hang if the node changes after the first Lstat.
			file, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
			if err != nil {
				return domain.ErrVerificationWorkspace
			}
			opened, statErr := file.Stat()
			if statErr != nil || !verificationFileUnchanged(root, name, info, opened) {
				file.Close()
				return domain.ErrVerificationWorkspace
			}
			limit := min(maxFileBytes, maxTotalBytes-walk.bytes)
			content, readErr := readBounded(walk.ctx, file, limit)
			closeErr := file.Close()
			if readErr != nil || closeErr != nil || !utf8.Valid(content) {
				return domain.ErrVerificationWorkspace
			}
			walk.bytes += len(content)
			walk.seen[relative] = struct{}{}
			walk.files = append(walk.files, domain.WorkspaceFile{Path: relative, Content: string(content)})
		}
		if readErr == io.EOF {
			return nil
		}
	}
}

func verificationFileUnchanged(root *os.Root, name string, before, opened os.FileInfo) bool {
	after, err := root.Lstat(name)
	return err == nil && after.Mode()&os.ModeSymlink == 0 && after.Mode().IsRegular() && opened.Mode().IsRegular() && os.SameFile(before, opened) && os.SameFile(opened, after)
}

// Enforce limits on actual reads, including a growth-detection byte, while
// checking cancellation between bounded chunks rather than only before stat.
func readBounded(ctx context.Context, file io.Reader, limit int) ([]byte, error) {
	content := make([]byte, 0, min(limit, 64<<10))
	buffer := make([]byte, 32<<10)
	for {
		if ctx.Err() != nil {
			return nil, domain.ErrVerificationWorkspace
		}
		remaining := limit + 1 - len(content)
		n, err := file.Read(buffer[:min(len(buffer), remaining)])
		content = append(content, buffer[:n]...)
		if len(content) > limit {
			return nil, domain.ErrVerificationWorkspace
		}
		if err == io.EOF {
			return content, nil
		}
		if err != nil || n == 0 {
			return nil, domain.ErrVerificationWorkspace
		}
	}
}
