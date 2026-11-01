package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
)

// OCR evidence is bound to image bytes, so a cached earlier submission cannot
// accidentally be graded as a different or corrected photograph.
type arithmeticInput struct {
	UploadSHA256 string `json:"upload_sha256,omitempty"`
	UploadWidth  int    `json:"upload_width,omitempty"`
	UploadHeight int    `json:"upload_height,omitempty"`
	Mode         string `json:"mode,omitempty"`
	Rotation     int    `json:"rotation,omitempty"`
	Name         string `json:"name,omitempty"`
	SHA256       string `json:"sha256,omitempty"`
	Verified     bool   `json:"verified"`
}

func imageDigest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func stampArithmeticInput(data, src, upload []byte, name string, rotation int) ([]byte, arithmeticInput, error) {
	input := arithmeticInput{Rotation: rotation, Name: filepath.Base(name), SHA256: imageDigest(src), Verified: true}
	input.UploadSHA256 = imageDigest(upload)
	input.Mode = "ocr"
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(upload)); err == nil {
		input.UploadWidth = cfg.Width
		input.UploadHeight = cfg.Height
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
		return nil, input, fmt.Errorf("invalid OCR JSON response")
	}
	value, _ := json.Marshal(input)
	fields["_cyeam_input"] = value
	stamped, err := json.Marshal(fields)
	return stamped, input, err
}
func verifyArithmeticInput(data []byte, imagePath string) (arithmeticInput, string, error) {
	var fields struct {
		Input *arithmeticInput `json:"_cyeam_input"`
	}
	if err := json.Unmarshal(data, &fields); err != nil {
		return arithmeticInput{}, "", fmt.Errorf("invalid OCR JSON response")
	}
	if fields.Input == nil || fields.Input.SHA256 == "" {
		return arithmeticInput{}, "Saved OCR has no image fingerprint; its input image cannot be verified.", nil
	}
	input := *fields.Input
	input.Verified = false
	src, err := readCheckFile(imagePath, 20<<20)
	if err != nil {
		if _, statErr := os.Stat(imagePath); os.IsNotExist(statErr) {
			return input, "Replay image is unavailable; only the saved OCR is being inspected.", nil
		}
		return input, "", err
	}
	if imageDigest(src) != input.SHA256 {
		return input, "", fmt.Errorf("saved OCR belongs to a different image; run check again without --ocr-result")
	}
	input.Verified = true
	return input, "", nil
}
