package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/mnhkahn/cyeam-cli/internal/output"
	"github.com/spf13/cobra"
	_ "golang.org/x/image/webp"
)

const maxHomeworkImageBytes = 10 << 20

func readCheckFile(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("input exceeds %d MiB", limit>>20)
	}
	return data, nil
}

func readHomeworkImage(path string) ([]byte, error) {
	data, err := readCheckFile(path, maxHomeworkImageBytes)
	if err != nil {
		return nil, err
	}
	mime := http.DetectContentType(data)
	if mime != "image/jpeg" && mime != "image/png" && mime != "image/webp" {
		return nil, fmt.Errorf("image must be JPG, PNG, or WebP")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width < 1 || config.Height < 1 || int64(config.Width)*int64(config.Height) > 40000000 {
		return nil, fmt.Errorf("image is invalid or exceeds 40 megapixels")
	}
	return data, nil
}

func readHomeworkWords(path string) ([]string, error) {
	data, err := readCheckFile(path, 64<<10)
	if err != nil {
		return nil, err
	}
	raw := strings.NewReplacer("\r\n", "\n", "，", "\n", ",", "\n", ";", "\n", "；", "\n").Replace(string(data))
	words := []string{}
	tokens := 0
	for _, word := range strings.Split(raw, "\n") {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}
		if len(word) > 150 {
			return nil, fmt.Errorf("dictation word exceeds 150 bytes")
		}
		words = append(words, word)
		tokens += len(strings.Fields(word))
	}
	if len(words) < 1 || len(words) > 100 {
		return nil, fmt.Errorf("dictation words must contain 1–100 items")
	}
	if tokens > 400 {
		return nil, fmt.Errorf("dictation words contain more than 400 tokens")
	}
	return words, nil
}

func homeworkInputFromJSON(data []byte, expectedKind string) (json.RawMessage, error) {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("invalid homework report JSON")
	}
	if len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		data = envelope.Data
	}
	var result struct {
		Input json.RawMessage `json:"input"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid homework report JSON")
	}
	if len(result.Input) > 0 && string(result.Input) != "null" {
		data = result.Input
	}
	var input struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(data, &input); err != nil || input.Kind == "" {
		return nil, fmt.Errorf("homework report has no valid input")
	}
	if input.Kind != expectedKind {
		return nil, fmt.Errorf("homework input kind is %q, expected %q", input.Kind, expectedKind)
	}
	return json.RawMessage(data), nil
}

func readHomeworkInput(path, expectedKind string) (json.RawMessage, error) {
	data, err := readCheckFile(path, 16<<20)
	if err != nil {
		return nil, err
	}
	return homeworkInputFromJSON(data, expectedKind)
}

type homeworkResultView struct {
	Input struct {
		Kind string `json:"kind"`
	} `json:"input"`
	Report struct {
		Grade       string   `json:"grade"`
		Provisional bool     `json:"provisional"`
		Correct     int      `json:"correct"`
		Incorrect   int      `json:"incorrect"`
		Pending     int      `json:"pending"`
		Total       int      `json:"total"`
		Accuracy    *float64 `json:"accuracy"`
		Items       []struct {
			Number     int    `json:"number"`
			Expression string `json:"expression"`
			Raw        string `json:"raw"`
			State      string `json:"state"`
			Status     string `json:"status"`
			Hint       string `json:"hint"`
		} `json:"items"`
		Feedback string `json:"feedback"`
	} `json:"report"`
}

func homeworkResultViewFromJSON(data []byte) (homeworkResultView, error) {
	var result homeworkResultView
	if err := json.Unmarshal(data, &result); err != nil {
		return result, fmt.Errorf("invalid homework response: %w", err)
	}
	return result, nil
}

func writeHomeworkResult(cmd *cobra.Command, data json.RawMessage) error {
	pretty, _ := cmd.Flags().GetBool("pretty")
	if !pretty {
		return output.WriteJSON(cmd.OutOrStdout(), data)
	}
	result, err := homeworkResultViewFromJSON(data)
	if err != nil {
		return err
	}
	if result.Input.Kind == "reading" {
		_, err = fmt.Fprint(cmd.OutOrStdout(), result.Report.Feedback)
		return err
	}
	var text strings.Builder
	grade := result.Report.Grade
	if grade == "" {
		grade = "暂不评级"
	} else if result.Report.Provisional {
		grade = "暂评·" + grade
	}
	fmt.Fprintf(&text, "成绩：%s\n", grade)
	fmt.Fprintf(&text, "正确 %d，需订正 %d，待核对 %d / 共 %d 项\n", result.Report.Correct, result.Report.Incorrect, result.Report.Pending, result.Report.Total)
	if result.Report.Accuracy != nil {
		fmt.Fprintf(&text, "可判定正确率：%.1f%%\n", *result.Report.Accuracy)
	}
	for _, item := range result.Report.Items {
		status := "待核对"
		if item.Status == "correct" {
			status = "正确"
		} else if item.Status == "incorrect" {
			status = "需订正"
		}
		raw := item.Raw
		if raw == "" {
			raw = "未识别"
			if item.State == "blank" {
				raw = "未作答"
			}
		}
		if result.Input.Kind == "arithmetic" {
			fmt.Fprintf(&text, "(%d) %s = %s  %s", item.Number, item.Expression, raw, status)
		} else {
			fmt.Fprintf(&text, "(%d) %s  %s", item.Number, raw, status)
		}
		if item.Hint != "" && item.Status != "correct" {
			fmt.Fprintf(&text, "（%s）", item.Hint)
		}
		text.WriteByte('\n')
	}
	_, err = io.WriteString(cmd.OutOrStdout(), text.String())
	return err
}
