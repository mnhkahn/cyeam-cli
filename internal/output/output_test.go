package output

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSONAddsTrailingNewline(t *testing.T) {
	var out bytes.Buffer
	if err := WriteJSON(&out, []byte(`{"ok":true}`)); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if out.String() != "{\"ok\":true}\n" {
		t.Fatalf("out = %q", out.String())
	}
}

func TestWriteFileWritesBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "char.png")

	if err := WriteFile(path, []byte{1, 2, 3}); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("file = %v", got)
	}
}
