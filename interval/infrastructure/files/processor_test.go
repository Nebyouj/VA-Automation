package files

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestProcessor(t *testing.T) (*LocalFileProcessor, string, string) {
	t.Helper()

	root := t.TempDir()
	inputDir := filepath.Join(root, "input")
	outputDir := filepath.Join(root, "output")

	processor, err := NewLocalFileProcessor(inputDir, outputDir)
	if err != nil {
		t.Fatalf("NewLocalFileProcessor() error = %v", err)
	}

	return processor, inputDir, outputDir
}

func writeInput(t *testing.T, inputDir, name, content string) {
	t.Helper()

	err := os.WriteFile(
		filepath.Join(inputDir, name),
		[]byte(content),
		0600,
	)
	if err != nil {
		t.Fatalf("write input file: %v", err)
	}
}

func TestProcessFileReadText(t *testing.T) {
	processor, inputDir, _ := newTestProcessor(t)
	writeInput(t, inputDir, "sample.txt", "Hello from Go\n")

	err := processor.ProcessFile(
		context.Background(),
		"sample.txt",
		"read_text",
		"",
		1,
	)
	if err != nil {
		t.Fatalf("ProcessFile() error = %v", err)
	}
}

func TestProcessFileTextStats(t *testing.T) {
	processor, inputDir, _ := newTestProcessor(t)
	writeInput(t, inputDir, "sample.txt", "one two\nthree\n")

	err := processor.ProcessFile(
		context.Background(),
		"sample.txt",
		"text_stats",
		"",
		2,
	)
	if err != nil {
		t.Fatalf("ProcessFile() error = %v", err)
	}

	if got := CountWords("one two\nthree\tfour"); got != 4 {
		t.Errorf("CountWords() = %d, want 4", got)
	}

	if got := CountWords(" \n\t "); got != 0 {
		t.Errorf("CountWords() for whitespace = %d, want 0", got)
	}
}

func TestProcessFileCopy(t *testing.T) {
	processor, inputDir, outputDir := newTestProcessor(t)
	const content = "Content to copy\n"
	writeInput(t, inputDir, "sample.txt", content)

	err := processor.ProcessFile(
		context.Background(),
		"sample.txt",
		"copy",
		"",
		3,
	)
	if err != nil {
		t.Fatalf("ProcessFile() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(outputDir, "sample.txt"))
	if err != nil {
		t.Fatalf("read copied file: %v", err)
	}

	if string(got) != content {
		t.Errorf("copied content = %q, want %q", string(got), content)
	}
}

func TestProcessFileDoesNotOverwriteOutput(t *testing.T) {
	processor, inputDir, outputDir := newTestProcessor(t)
	writeInput(t, inputDir, "sample.txt", "new content")

	outputPath := filepath.Join(outputDir, "sample.txt")
	const original = "keep this content"
	if err := os.WriteFile(outputPath, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	err := processor.ProcessFile(
		context.Background(),
		"sample.txt",
		"copy",
		"",
		4,
	)
	if err == nil {
		t.Fatal("expected copy to fail when output already exists")
	}

	got, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != original {
		t.Errorf("existing output changed: got %q, want %q", got, original)
	}
}

func TestProcessFileRejectsUnsupportedOperation(t *testing.T) {
	processor, inputDir, _ := newTestProcessor(t)
	writeInput(t, inputDir, "sample.txt", "content")

	err := processor.ProcessFile(
		context.Background(),
		"sample.txt",
		"delete_everything",
		"",
		5,
	)
	if err == nil {
		t.Fatal("expected unsupported operation to fail")
	}
	if !strings.Contains(err.Error(), "unsupported file operation") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestProcessFileRejectsMissingFile(t *testing.T) {
	processor, _, _ := newTestProcessor(t)

	err := processor.ProcessFile(
		context.Background(),
		"missing.txt",
		"read_text",
		"",
		6,
	)
	if err == nil {
		t.Fatal("expected missing file to fail")
	}
}

func TestProcessFileRejectsPathTraversal(t *testing.T) {
	processor, _, _ := newTestProcessor(t)

	for _, name := range []string{
		"../secret.txt",
		"..\\secret.txt",
		"..",
	} {
		err := processor.ProcessFile(
			context.Background(),
			name,
			"read_text",
			"",
			7,
		)
		if err == nil {
			t.Errorf("expected path %q to be rejected", name)
		}
	}
}

func TestProcessFileRejectsCancelledContext(t *testing.T) {
	processor, inputDir, _ := newTestProcessor(t)
	writeInput(t, inputDir, "sample.txt", "content")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := processor.ProcessFile(ctx, "sample.txt", "read_text", "", 8)
	if err == nil {
		t.Fatal("expected cancelled context to fail")
	}
}

func TestProcessFileRejectsOversizedFile(t *testing.T) {
	processor, inputDir, _ := newTestProcessor(t)

	file, err := os.Create(filepath.Join(inputDir, "large.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if err := file.Truncate(maxFileSize + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	err = processor.ProcessFile(
		context.Background(),
		"large.txt",
		"read_text",
		"",
		9,
	)
	if err == nil {
		t.Fatal("expected oversized file to be rejected")
	}
}

func TestProcessFileRejectsSymbolicLink(t *testing.T) {
	processor, inputDir, _ := newTestProcessor(t)

	target := filepath.Join(inputDir, "target.txt")
	if err := os.WriteFile(target, []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(inputDir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symbolic links are unavailable in this environment: %v", err)
	}

	err := processor.ProcessFile(
		context.Background(),
		"link.txt",
		"read_text",
		"",
		10,
	)
	if err == nil {
		t.Fatal("expected symbolic link to be rejected")
	}
}
