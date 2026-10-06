package workspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

func fixtureRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "case"), 0700); err != nil {
		t.Fatal(err)
	}
	return root
}

func putFile(t *testing.T, root, name string, content []byte) {
	t.Helper()
	p := filepath.Join(root, "case", name)
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, content, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestVerificationReaderPreservesTextAndSortedRelativePaths(t *testing.T) {
	root := fixtureRoot(t)
	putFile(t, root, "z.txt", []byte("Привет\n"))
	putFile(t, root, "a/index.html", []byte("<main>private fixture</main>"))
	putFile(t, root, "b.txt", []byte{'x', 0, 'y'}) // Content string permits UTF-8 U+0000; paths do not.
	files, err := NewVerificationFileReader(root).ReadFiles(context.Background(), "/workspaces/case")
	if err != nil || len(files) != 3 || files[0].Path != "a/index.html" || files[1].Path != "b.txt" || files[1].Content != "x\x00y" || files[2].Path != "z.txt" || files[2].Content != "Привет\n" {
		t.Fatalf("text projection: %+v err=%v", files, err)
	}
}

func TestVerificationReaderRejectsUnsafeOrUnsupportedNodes(t *testing.T) {
	for _, name := range []string{"root symlink", "file symlink inside", "file symlink outside", "fifo", "invalid UTF8", "noncanonical filename", "empty tree"} {
		t.Run(name, func(t *testing.T) {
			root := fixtureRoot(t)
			putFile(t, root, "good.txt", []byte("private fixture marker"))
			p := filepath.Join(root, "case", "unsafe")
			var err error
			switch name {
			case "root symlink":
				err = os.Rename(filepath.Join(root, "case"), filepath.Join(root, "actual"))
				if err == nil {
					err = os.Symlink("actual", filepath.Join(root, "case"))
				}
			case "file symlink inside":
				err = os.Symlink("good.txt", p)
			case "file symlink outside":
				err = os.Symlink(root, p)
			case "fifo":
				err = syscall.Mkfifo(p, 0600)
			case "invalid UTF8":
				putFile(t, root, "unsafe", []byte{0xff})
			case "noncanonical filename":
				putFile(t, root, "unsafe\\name", []byte("x"))
			case "empty tree":
				err = os.Remove(filepath.Join(root, "case", "good.txt"))
			}
			if err != nil {
				t.Fatal(err)
			}
			files, err := NewVerificationFileReader(root).ReadFiles(context.Background(), "/workspaces/case")
			if !errors.Is(err, domain.ErrVerificationWorkspace) || files != nil || strings.Contains(err.Error(), root) || strings.Contains(err.Error(), "marker") {
				t.Fatalf("unsafe projection: files=%d err=%v", len(files), err)
			}
		})
	}
}

func TestVerificationReaderRejectsInvalidMountRootAndCancellation(t *testing.T) {
	root := fixtureRoot(t)
	putFile(t, root, "good", []byte("text"))
	link := filepath.Join(root, "mount-link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	for _, base := range []string{"", "relative", root + "/../", root + "/missing", link} {
		if _, err := NewVerificationFileReader(base).ReadFiles(context.Background(), "/workspaces/case"); !errors.Is(err, domain.ErrVerificationWorkspace) {
			t.Fatalf("invalid mount accepted: %v", err)
		}
	}
	for _, portable := range []string{"", root, "/workspaces/case/child", "/workspaces/../case"} {
		if _, err := NewVerificationFileReader(root).ReadFiles(context.Background(), portable); !errors.Is(err, domain.ErrVerificationWorkspace) {
			t.Fatalf("invalid portable root accepted: %v", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewVerificationFileReader(root).ReadFiles(ctx, "/workspaces/case"); !errors.Is(err, domain.ErrVerificationWorkspace) {
		t.Fatalf("cancelled read accepted: %v", err)
	}
}

func TestVerificationReaderEnforcesFileAndAggregateLimits(t *testing.T) {
	for _, test := range []struct {
		name        string
		count, size int
	}{
		{"file count", maxFiles + 1, 1}, {"file bytes", 1, maxFileBytes + 1}, {"total bytes", 5, maxFileBytes},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := fixtureRoot(t)
			for i := 0; i < test.count; i++ {
				putFile(t, root, fmt.Sprintf("%04d.txt", i), bytes.Repeat([]byte("a"), test.size))
			}
			if files, err := NewVerificationFileReader(root).ReadFiles(context.Background(), "/workspaces/case"); !errors.Is(err, domain.ErrVerificationWorkspace) || files != nil {
				t.Fatalf("limit exceeded without failure: files=%d err=%v", len(files), err)
			}
		})
	}
}

func TestVerificationReaderBoundsDirectoryWalk(t *testing.T) {
	for _, name := range []string{"entries", "depth"} {
		t.Run(name, func(t *testing.T) {
			root := fixtureRoot(t)
			putFile(t, root, "good.txt", []byte("text"))
			if name == "entries" {
				for i := 0; i < maxWalkEntries; i++ {
					if err := os.Mkdir(filepath.Join(root, "case", fmt.Sprintf("dir-%05d", i)), 0700); err != nil {
						t.Fatal(err)
					}
				}
			} else {
				if err := os.MkdirAll(filepath.Join(root, "case", strings.Repeat("d/", maxWalkDepth+1)), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := NewVerificationFileReader(root).ReadFiles(context.Background(), "/workspaces/case"); !errors.Is(err, domain.ErrVerificationWorkspace) {
				t.Fatalf("unbounded walk accepted: %v", err)
			}
		})
	}
}

func TestBoundedReadUsesActualBytesAndCancellation(t *testing.T) {
	if _, err := readBounded(context.Background(), strings.NewReader("growing"), 3); !errors.Is(err, domain.ErrVerificationWorkspace) {
		t.Fatalf("actual excess accepted: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readBounded(ctx, strings.NewReader("text"), 10); !errors.Is(err, domain.ErrVerificationWorkspace) {
		t.Fatalf("cancelled read accepted: %v", err)
	}
}

func TestVerificationFileGuardRejectsPathMutationBeforeRead(t *testing.T) {
	for _, kind := range []string{"symlink to same opened inode", "different regular file"} {
		t.Run(kind, func(t *testing.T) {
			rootPath := fixtureRoot(t)
			putFile(t, rootPath, "source.txt", []byte("private original"))
			root, err := os.OpenRoot(filepath.Join(rootPath, "case"))
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			before, err := root.Lstat("source.txt")
			if err != nil {
				t.Fatal(err)
			}
			file, err := root.Open("source.txt")
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			opened, err := file.Stat()
			if err != nil {
				t.Fatal(err)
			}
			if !verificationFileUnchanged(root, "source.txt", before, opened) {
				t.Fatal("stable opened file rejected")
			}
			if err := os.Rename(filepath.Join(rootPath, "case", "source.txt"), filepath.Join(rootPath, "case", "retained.txt")); err != nil {
				t.Fatal(err)
			}
			if kind == "symlink to same opened inode" {
				err = os.Symlink("retained.txt", filepath.Join(rootPath, "case", "source.txt"))
			} else {
				err = os.WriteFile(filepath.Join(rootPath, "case", "source.txt"), []byte("replacement"), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			if verificationFileUnchanged(root, "source.txt", before, opened) {
				t.Fatal("changed path accepted before reading opened descriptor")
			}
		})
	}
}
