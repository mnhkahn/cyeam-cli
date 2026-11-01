package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArithmeticReplayRejectsAnotherPhotograph(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photo.png")
	first := []byte("first photograph")
	data, input, err := stampArithmeticInput([]byte(`{"result":{"ocrResults":[]}}`), first, path, 0)
	if err != nil || !input.Verified {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, first, 0600); err != nil {
		t.Fatal(err)
	}
	verified, warning, err := verifyArithmeticInput(data, path)
	if err != nil || warning != "" || !verified.Verified {
		t.Fatal(verified, warning, err)
	}
	if err := os.WriteFile(path, []byte("corrected photograph"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := verifyArithmeticInput(data, path); err == nil || !strings.Contains(err.Error(), "different image") {
		t.Fatal("accepted stale OCR", err)
	}
}
func TestArithmeticOldOCRHasUnverifiedSource(t *testing.T) {
	input, warning, err := verifyArithmeticInput([]byte(`{"ocrResults":[]}`), "photo.png")
	if err != nil || input.Verified || warning == "" {
		t.Fatal(input, warning, err)
	}
}
