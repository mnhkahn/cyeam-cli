package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/arithmetic"
	"github.com/mnhkahn/cyeam-cli/internal/paddleocr"
	"github.com/mnhkahn/cyeam-cli/internal/visionocr"
)

type arithmeticOCRProvider struct {
	Engine string          `json:"engine"`
	Raw    json.RawMessage `json:"raw,omitempty"`
	Error  string          `json:"error,omitempty"`
}
type arithmeticOCREvidence struct {
	Version   int                     `json:"_cyeam_ocr_version"`
	Providers []arithmeticOCRProvider `json:"providers"`
}
type arithmeticRecognizer struct {
	Engine string
	Run    func(context.Context) ([]byte, error)
}

// Both jobs start before either is awaited. A per-provider deadline bounds the
// wait even if a backend ignores cancellation; buffered sends never leak waiters.
func recognizeArithmeticParallel(ctx context.Context, timeout time.Duration, jobs []arithmeticRecognizer) ([]byte, error) {
	type completed struct {
		index int
		value arithmeticOCRProvider
	}
	done := make(chan completed, len(jobs))
	for i, job := range jobs {
		go func(i int, job arithmeticRecognizer) {
			providerCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			finished := make(chan arithmeticOCRProvider, 1)
			go func() {
				raw, err := job.Run(providerCtx)
				value := arithmeticOCRProvider{Engine: job.Engine, Raw: raw}
				if err != nil {
					value.Raw = nil
					value.Error = err.Error()
				}
				finished <- value
			}()
			select {
			case result := <-finished:
				done <- completed{i, result}
			case <-providerCtx.Done():
				done <- completed{i, arithmeticOCRProvider{Engine: job.Engine, Error: providerCtx.Err().Error()}}
			}
		}(i, job)
	}
	evidence := arithmeticOCREvidence{Version: 1, Providers: make([]arithmeticOCRProvider, len(jobs))}
	for range jobs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case result := <-done:
			evidence.Providers[result.index] = result.value
		}
	}
	// Parent cancellation always wins over a partial success.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	usable := false
	for i := range evidence.Providers {
		p := &evidence.Providers[i]
		if p.Error != "" {
			continue
		}
		if _, err := checkArithmeticProvider(*p, arithmetic.CheckOptions{}); err != nil {
			p.Error = err.Error()
			p.Raw = nil
		} else {
			usable = true
		}
	}
	if !usable {
		failures := []error{}
		for _, p := range evidence.Providers {
			failures = append(failures, fmt.Errorf("%s: %s", p.Engine, p.Error))
		}
		return nil, fmt.Errorf("no OCR engine returned usable results: %w", errors.Join(failures...))
	}
	return json.Marshal(evidence)
}

func checkArithmeticProvider(p arithmeticOCRProvider, opts arithmetic.CheckOptions) (arithmetic.CheckReport, error) {
	if p.Engine != "vision" {
		lines, err := paddleocr.Parse(p.Raw)
		if err != nil {
			return arithmetic.CheckReport{}, err
		}
		report, err := arithmetic.Check(lines, opts)
		report.Engine = "paddleocr"
		return report, err
	}
	result, err := visionocr.Parse(p.Raw)
	if err != nil {
		return arithmetic.CheckReport{}, err
	}
	report, err := arithmetic.Check(result.Lines, opts)
	if err != nil {
		return report, err
	}
	report.Engine = "vision"
	// Re-recognition candidates are bounded fixed transforms of malformed answer
	// regions. Keep the original if none passes; never search lower-ranked text.
	reports := []arithmetic.CheckReport{report}
	for _, alt := range result.Alternatives {
		r, e := arithmetic.Check([]paddleocr.Line{alt.Line}, opts)
		if e != nil {
			return report, e
		}
		r.Engine = "vision:" + alt.Kind
		reports = append(reports, r)
	}
	report = arithmetic.MergeChecks(reports...)
	report.Engine = "vision"
	return report, nil
}

func checkArithmeticEvidence(data []byte, opts arithmetic.CheckOptions) (arithmetic.CheckReport, error) {
	var evidence arithmeticOCREvidence
	if err := json.Unmarshal(data, &evidence); err != nil {
		return arithmetic.CheckReport{}, fmt.Errorf("invalid OCR JSON response")
	}
	if evidence.Version == 0 {
		return checkArithmeticProvider(arithmeticOCRProvider{Engine: "paddleocr", Raw: data}, opts)
	}
	if evidence.Version != 1 || len(evidence.Providers) == 0 {
		return arithmetic.CheckReport{}, fmt.Errorf("unsupported or empty OCR evidence")
	}
	reports := []arithmetic.CheckReport{}
	warnings := []string{}
	// Always prefer Vision, independent of which request finished first or the
	// provider order in a saved replay.
	for _, engine := range []string{"vision", "paddleocr"} {
		for _, p := range evidence.Providers {
			if p.Engine != engine {
				continue
			}
			if p.Error != "" {
				warnings = append(warnings, engine+": "+p.Error)
				continue
			}
			r, err := checkArithmeticProvider(p, opts)
			if err != nil {
				warnings = append(warnings, engine+": "+err.Error())
				continue
			}
			reports = append(reports, r)
		}
	}
	if len(reports) == 0 {
		return arithmetic.CheckReport{}, fmt.Errorf("saved evidence contains no usable OCR results")
	}
	report := arithmetic.MergeChecks(reports...)
	report.Warnings = append(report.Warnings, warnings...)
	return report, nil
}
