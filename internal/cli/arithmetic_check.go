package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/arithmetic"
	"github.com/mnhkahn/cyeam-cli/internal/paddleocr"
	"github.com/spf13/cobra"
	_ "golang.org/x/image/webp"
)

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
func newArithmeticCheckCommand() *cobra.Command {
	var envFile, replay, save string
	var opts arithmetic.CheckOptions
	var timeout time.Duration
	cmd := &cobra.Command{
		Use: "check <image>", Short: "Check photographed arithmetic using PaddleOCR and local calculation",
		Long: `Upload a local image to the configured PaddleOCR service, then calculate and compare
recognized horizontal equations and their answers. Supports +, -, ×, ÷, parentheses,
exact decimals/fractions, and quotient/remainder answers. Ignores scratch work below.
Unclear, unsupported, low-confidence, or unscored OCR results require review.
No handwriting digit substitutions or AI grading are performed.
Run cyeam ocr config set --url <endpoint> to save your own credentials.
Alternatively set OCR_API_URL/OCR_API_TOKEN or pass --ocr-env to a dotenv file.
Without your own endpoint and token, image checking is unavailable; no other OCR backend is used.
Use --save-ocr to retain OCR evidence and --ocr-result to replay it without uploading.`,
		Example: `  cyeam arithmetic check homework.jpg --ocr-env ~/code/wangxizhi/.env --pretty
  cyeam arithmetic check homework.jpg --expected-count 18 --save-ocr ocr.json
  cyeam arithmetic check homework.jpg --ocr-result ocr.json --pretty`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if math.IsNaN(opts.MinConfidence) || opts.MinConfidence < 0 || opts.MinConfidence > 1 {
				return fmt.Errorf("--min-confidence must be between 0 and 1")
			}
			if opts.ExpectedCount < 0 || opts.ExpectedCount > 1000 || cmd.Flags().Changed("expected-count") && opts.ExpectedCount == 0 {
				return fmt.Errorf("--expected-count must be between 1 and 1000")
			}
			if timeout <= 0 || timeout > 5*time.Minute {
				return fmt.Errorf("--timeout must be between 0 and 5m")
			}
			if save != "" && replay != "" {
				return fmt.Errorf("--save-ocr and --ocr-result cannot be combined")
			}
			if save != "" {
				savePath, _ := filepath.Abs(save)
				for _, input := range []string{args[0], envFile} {
					if input == "" {
						continue
					}
					inputPath, _ := filepath.Abs(input)
					saveInfo, e1 := os.Stat(save)
					inputInfo, e2 := os.Stat(input)
					if savePath == inputPath || e1 == nil && e2 == nil && os.SameFile(saveInfo, inputInfo) {
						return fmt.Errorf("--save-ocr must not overwrite the input image or OCR env file")
					}
				}
			}
			var data []byte
			var err error
			if replay != "" {
				data, err = readCheckFile(replay, 32<<20)
			} else {
				var src []byte
				src, err = readCheckFile(args[0], 20<<20)
				if err != nil {
					return err
				}
				cfg, _, decodeErr := image.DecodeConfig(bytes.NewReader(src))
				if decodeErr != nil || cfg.Width <= 0 || cfg.Height <= 0 {
					return fmt.Errorf("input must be a readable PNG, JPEG, GIF, or WebP image")
				}
				client, configErr := paddleocr.Config(envFile)
				if configErr != nil {
					return configErr
				}
				ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
				defer cancel()
				client.HTTP.Timeout = timeout
				data, err = client.RecognizeRaw(ctx, src)
			}
			if err != nil {
				return err
			}
			lines, err := paddleocr.Parse(data)
			if err != nil {
				return err
			}
			report, err := arithmetic.Check(lines, opts)
			if err != nil {
				return err
			}
			if save != "" {
				if err := os.WriteFile(save, data, 0600); err != nil {
					return fmt.Errorf("save OCR evidence: %w", err)
				}
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			if pretty {
				return writeArithmeticCheck(cmd.OutOrStdout(), report)
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(report)
		},
	}
	cmd.Flags().StringVar(&envFile, "ocr-env", "", "dotenv file with OCR_API_URL and OCR_API_TOKEN")
	cmd.Flags().StringVar(&replay, "ocr-result", "", "replay an existing PaddleOCR JSON result; do not upload")
	cmd.Flags().StringVar(&save, "save-ocr", "", "save raw PaddleOCR JSON evidence to a file")
	cmd.Flags().Float64Var(&opts.MinConfidence, "min-confidence", 0.9, "minimum recognition confidence for automatic judgment")
	cmd.Flags().IntVar(&opts.ExpectedCount, "expected-count", 0, "expected number of questions, to detect omissions")
	cmd.Flags().DurationVar(&timeout, "timeout", 90*time.Second, "OCR request timeout (maximum 5m)")
	return cmd
}
func writeArithmeticCheck(out io.Writer, r arithmetic.CheckReport) error {
	var text strings.Builder
	fmt.Fprintf(&text, "识别 %d 题：正确 %d，错误 %d，待核对 %d\n", r.Detected, r.Correct, r.Incorrect, r.NeedsReview)
	fmt.Fprintf(&text, "按OCR识别值比较：相符 %d，不符 %d，无法比较 %d\n", r.MatchingValues, r.MismatchingValues, r.Detected-r.MatchingValues-r.MismatchingValues)
	reasons := map[string]string{
		"OCR did not provide equation coordinates":       "缺少题目坐标",
		"OCR did not provide recognition confidence":     "服务未提供识别置信度",
		"OCR confidence is below the threshold":          "识别置信度不足",
		"multiple possible answers on this row":          "同一行有多个可能答案",
		"answer is missing or not an unambiguous number": "答案缺失或无法明确识别",
		"duplicate question number":                      "题号重复",
	}
	for i, q := range r.Questions {
		number := q.Number
		if number == 0 {
			number = i + 1
		}
		state := "待核对"
		if q.Status == "correct" {
			state = "正确"
		} else if q.Status == "incorrect" {
			state = "错误"
		}
		answer := q.SubmittedAnswer
		if answer == "" {
			answer = "未识别"
		}
		fmt.Fprintf(&text, "(%d) %s = %s  %s", number, q.Expression, answer, state)
		if q.ExpectedAnswer != "" {
			fmt.Fprintf(&text, "；计算结果：%s", q.ExpectedAnswer)
		}
		if q.Status == "needs_review" && q.Comparison != "" {
			match := "相符"
			if q.Comparison == "incorrect" {
				match = "不符"
			}
			fmt.Fprintf(&text, "；识别值与计算结果%s", match)
		}
		if q.Reason != "" {
			reason := reasons[q.Reason]
			if reason == "" {
				reason = q.Reason
			}
			fmt.Fprintf(&text, "（%s）", reason)
		}
		text.WriteByte('\n')
	}
	if r.ExpectedCount > 0 && r.Detected != r.ExpectedCount {
		fmt.Fprintf(&text, "预计 %d 题，实际识别 %d 题。\n", r.ExpectedCount, r.Detected)
	}
	if len(r.MissingNumbers) > 0 {
		fmt.Fprintf(&text, "未识别的题号：%v\n", r.MissingNumbers)
	}
	if r.Detected == 0 {
		text.WriteString("未识别到支持的算式，不能据此判断全部正确。\n")
	}
	text.WriteString("统计仅包含识别到的算式；手写内容可能存在漏识别或误识别。\n")
	_, err := io.WriteString(out, text.String())
	return err
}
