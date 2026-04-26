package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileRead reads a file and returns its contents.
func FileRead(ctx context.Context, path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %q: %w", path, err)
	}
	return string(b), nil
}

// FileWrite writes content to a file.
func FileWrite(ctx context.Context, path, content string) (string, error) {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("mkdir %q: %w", dir, err)
		}
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write %q: %w", path, err)
	}
	return fmt.Sprintf("Wrote %d bytes to %s", len(content), path), nil
}

// FileGlob returns matching file paths.
func FileGlob(ctx context.Context, pattern string) (string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("glob %q: %w", pattern, err)
	}
	return strings.Join(matches, "\n"), nil
}

// FileGrep searches for a pattern in files matching a glob.
func FileGrep(ctx context.Context, pattern, globPattern string) (string, error) {
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return "", fmt.Errorf("glob %q: %w", globPattern, err)
	}
	var out strings.Builder
	for _, path := range matches {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		b, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			continue
		}
		lines := strings.Split(string(b), "\n")
		for i, line := range lines {
			if strings.Contains(line, pattern) {
				out.WriteString(fmt.Sprintf("%s:%d: %s\n", path, i+1, line))
			}
		}
	}
	if out.Len() == 0 {
		return "No matches found.", nil
	}
	return out.String(), nil
}
