package testutil

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Golden compares output against a golden file.
// If the GOLDEN_UPDATE environment variable is set, updates the golden file.
func Golden(t *testing.T, name string, got []byte) {
	t.Helper()

	goldenPath := filepath.Join("testdata", name+".golden")

	if os.Getenv("GOLDEN_UPDATE") != "" {
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatalf("failed to create testdata dir: %v", err)
		}
		if err := os.WriteFile(goldenPath, got, 0644); err != nil {
			t.Fatalf("failed to update golden file: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v\nGot:\n%s", goldenPath, err, got)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("output mismatch for %s\nWant:\n%s\nGot:\n%s", name, want, got)
	}
}

// GoldenString is like Golden but takes a string.
func GoldenString(t *testing.T, name string, got string) {
	t.Helper()
	Golden(t, name, []byte(got))
}
