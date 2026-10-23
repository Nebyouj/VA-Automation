package files

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxFileSize = 10 * 1024 * 1024 // 10 MB

type LocalFileProcessor struct {
	inputDir  string
	outputDir string
}

func NewLocalFileProcessor(inputDir, outputDir string) (*LocalFileProcessor, error) {
	if strings.TrimSpace(inputDir) == "" || strings.TrimSpace(outputDir) == "" {
		return nil, fmt.Errorf("input and output directories are required")
	}

	inputDir, err := filepath.Abs(inputDir)
	if err != nil {
		return nil, fmt.Errorf("resolve input directory: %w", err)
	}

	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		return nil, fmt.Errorf("resolve output directory: %w", err)
	}

	if err := os.MkdirAll(inputDir, 0750); err != nil {
		return nil, fmt.Errorf("create input directory: %w", err)
	}

	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	return &LocalFileProcessor{
		inputDir:  inputDir,
		outputDir: outputDir,
	}, nil
}

func (p *LocalFileProcessor) ProcessFile(
	ctx context.Context,
	fileName, operation, content string,
	jobID int,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if operation == "write_text" {
		if !filepath.IsLocal(fileName) ||
			filepath.Base(fileName) != fileName ||
			fileName == "." {
			return fmt.Errorf(
				"file_name must be a simple file name inside the output directory",
			)
		}

		if len(content) > maxFileSize {
			return fmt.Errorf("content exceeds the 10 MB size limit")
		}

		outputPath := filepath.Join(p.outputDir, fileName)

		if err := writeFile(ctx, outputPath, []byte(content)); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}

		fmt.Printf(
			"File job %d wrote %s (%d bytes)\n",
			jobID,
			fileName,
			len(content),
		)

		return nil
	}

	if !filepath.IsLocal(fileName) ||
		filepath.Base(fileName) != fileName ||
		fileName == "." {
		return fmt.Errorf("file_name must be a simple file name inside the input directory")
	}

	sourcePath := filepath.Join(p.inputDir, fileName)

	info, err := os.Lstat(sourcePath)
	if err != nil {
		return fmt.Errorf("inspect input file: %w", err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symbolic links are not supported")
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("input must be a regular file")
	}

	if info.Size() > maxFileSize {
		return fmt.Errorf("file exceeds the 10 MB size limit")
	}

	data, err := readFile(ctx, sourcePath)
	if err != nil {
		return fmt.Errorf("read input file: %w", err)
	}

	switch operation {
	case "read_text":
		if !utf8.Valid(data) {
			return fmt.Errorf("input file is not valid UTF-8 text")
		}

		fmt.Printf(
			"File job %d read %s (%d bytes):\n%s\n",
			jobID,
			fileName,
			len(data),
			string(data),
		)

	case "text_stats":
		if !utf8.Valid(data) {
			return fmt.Errorf("input file is not valid UTF-8 text")
		}

		text := string(data)
		lines := 0
		words := 0

		if len(text) > 0 {
			lines = strings.Count(text, "\n")
			if !strings.HasSuffix(text, "\n") {
				lines++
			}
			words = len(strings.Fields(text))
		}

		fmt.Printf(
			"File job %d: %s | bytes=%d | lines=%d | words=%d\n",
			jobID,
			fileName,
			len(data),
			lines,
			words,
		)

	case "copy":
		if err := ctx.Err(); err != nil {
			return err
		}

		outputName := filepath.Base(fileName)
		outputPath := filepath.Join(p.outputDir, outputName)

		if err := writeFile(ctx, outputPath, data); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}

		fmt.Printf(
			"File job %d copied %s to %s\n",
			jobID,
			fileName,
			outputPath,
		)

	case "write_text":
		if !filepath.IsLocal(fileName) ||
			filepath.Base(fileName) != fileName ||
			fileName == "." {
			return fmt.Errorf("file_name must be a simple file name inside the output directory")
		}

		if len(content) > maxFileSize {
			return fmt.Errorf("content exceeds the 10 MB size limit")
		}

		outputPath := filepath.Join(p.outputDir, fileName)
		if err := writeFile(ctx, outputPath, []byte(content)); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}

		fmt.Printf(
			"File job %d wrote %s (%d bytes)\n",
			jobID,
			fileName,
			len(content),
		)
		return nil

	default:
		return fmt.Errorf(
			"unsupported file operation %q; supported operations: read_text, text_stats, copy",
			operation,
		)
	}

	return nil
}

func readFile(ctx context.Context, path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxFileSize+1))
	if err != nil {
		return nil, err
	}

	if len(data) > maxFileSize {
		return nil, fmt.Errorf("file exceeds the 10 MB size limit")
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return data, nil
}

func writeFile(ctx context.Context, path string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	file, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0600,
	)
	if err != nil {
		return err
	}

	_, writeErr := file.Write(data)
	closeErr := file.Close()

	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}

	return ctx.Err()
}

// CountWords counts whitespace-separated words.
// Kept separate so word-count behavior can be tested independently.
func CountWords(text string) int {
	return len(strings.FieldsFunc(text, unicode.IsSpace))
}
