package reading

import (
	"encoding/json"
	"fmt"
)

// ParseReport recomputes the grade and feedback from saved inputs rather than
// trusting editable generated fields in an intermediate report.
func ParseReport(data []byte) (Report, error) {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return Report{}, fmt.Errorf("invalid reading report JSON")
	}
	if len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		data = envelope.Data
	}
	var evidence struct {
		SchemaVersion    int     `json:"schema_version"`
		Reference        string  `json:"reference"`
		Transcript       string  `json:"transcript"`
		ReferenceSHA256  string  `json:"reference_sha256"`
		TranscriptSHA256 string  `json:"transcript_sha256"`
		Options          Options `json:"options"`
	}
	if err := json.Unmarshal(data, &evidence); err != nil || evidence.SchemaVersion != SchemaVersion {
		return Report{}, fmt.Errorf("unsupported or invalid reading report")
	}
	if evidence.ReferenceSHA256 != digest(evidence.Reference) || evidence.TranscriptSHA256 != digest(evidence.Transcript) {
		return Report{}, fmt.Errorf("reading report input fingerprint mismatch")
	}
	return Review(evidence.Reference, evidence.Transcript, evidence.Options)
}
