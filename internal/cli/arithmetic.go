package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/arithmetic"
	"github.com/spf13/cobra"
)

func newArithmeticCommand(deps Dependencies) *cobra.Command {
	var opts arithmetic.Options
	var format string
	cmd := &cobra.Command{
		Use: "arithmetic", Short: "Generate and check oral arithmetic exercises",
		Long: `Generate unique oral arithmetic questions using the cyeam.com service.
Default: 20 questions, balanced across the selected grade's types, without answers.
Use 'arithmetic types' to list the latest type IDs and Chinese names from the service.
Use 'arithmetic check <image>' to check photographed work with PaddleOCR.
Text format contains one question per line, suitable for PDF input or tablet tasks.
Use --pretty --format text for raw text without the global JSON envelope.`,
		Example: `  cyeam arithmetic --grade 2 --type multiply,divide --count 20
  cyeam arithmetic --type 表内乘法 --answers --seed 42
  cyeam arithmetic --grade 4 --format worksheet --pretty
  cyeam arithmetic --grade 2 --format text --pretty | cyeam pdf --mode arithmetic -o practice.pdf`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "json" && format != "text" && format != "worksheet" {
				return fmt.Errorf("format must be json, text, or worksheet")
			}
			if opts.Count < 1 || opts.Count > 1000 {
				return fmt.Errorf("--count must be between 1 and 1000")
			}
			if opts.Grade < 0 {
				return fmt.Errorf("grade must be non-negative")
			}
			if deps.Service == nil {
				return fmt.Errorf("arithmetic service is not configured")
			}
			if !cmd.Flags().Changed("seed") {
				opts.Seed = time.Now().UnixNano()
			}
			exercise, err := deps.Service.ArithmeticGenerate(cmd.Context(), opts)
			if err != nil {
				return err
			}
			if format == "worksheet" {
				exercise.Types, err = deps.Service.ArithmeticTypes(cmd.Context(), exercise.Grade)
				if err != nil {
					return err
				}
			}
			if format == "json" {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(exercise)
			}
			return writeArithmetic(cmd.OutOrStdout(), exercise, format)
		},
	}
	cmd.Flags().IntVar(&opts.Grade, "grade", 0, "grade; inferred for a specific type")
	cmd.Flags().StringSliceVar(&opts.Types, "type", nil, "type IDs or Chinese names, comma-separated; default: balanced mix")
	cmd.Flags().IntVar(&opts.Count, "count", 20, "number of unique questions (1–1000)")
	cmd.Flags().BoolVar(&opts.Answers, "answers", false, "include answers separately from questions")
	cmd.Flags().Int64Var(&opts.Seed, "seed", 0, "random seed for reproducible exercises")
	cmd.Flags().StringVar(&format, "format", "json", "output format: json, text, worksheet")
	var grade int
	typesCmd := &cobra.Command{Use: "types", Short: "List supported arithmetic types", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if grade < 0 {
				return fmt.Errorf("grade must be non-negative")
			}
			if deps.Service == nil {
				return fmt.Errorf("arithmetic service is not configured")
			}
			types, err := deps.Service.ArithmeticTypes(cmd.Context(), grade)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(types)
		}}
	typesCmd.Flags().IntVar(&grade, "grade", 0, "filter by grade")
	cmd.AddCommand(typesCmd, newArithmeticCheckCommand())
	return cmd
}

func writeArithmetic(out io.Writer, e arithmetic.Exercise, format string) error {
	var text strings.Builder
	if format == "text" {
		for _, q := range e.Questions {
			fmt.Fprintln(&text, q.Text)
		}
	} else {
		names, seen := []string{}, map[string]bool{}
		for _, q := range e.Questions {
			for _, typ := range e.Types {
				if typ.ID == q.Type && !seen[q.Type] {
					names = append(names, typ.Name)
					seen[q.Type] = true
				}
			}
		}
		fmt.Fprintf(&text, "北师大版口算练习\n%d年级｜%s｜%d题\n姓名：______  日期：______  用时：______分钟  正确：______/%d\n\n", e.Grade, strings.Join(names, "、"), len(e.Questions), len(e.Questions))
		columns := 5
		cells := [][]string{}
		width := 0
		for i, q := range e.Questions {
			switch q.Type {
			case "quantity":
				columns = min(columns, 2)
			case "large-number", "product-change", "estimate", "three-digit-chain":
				columns = min(columns, 3)
			}
			cell := []string{fmt.Sprintf("%d. %s", i+1, strings.ReplaceAll(q.Text, "（ ）", "（      ）"))}
			if strings.HasPrefix(q.Type, "column-") {
				expression := strings.Split(strings.TrimPrefix(q.Text, "列竖式计算："), "＝")[0]
				parts := strings.FieldsFunc(expression, func(r rune) bool { return r == '＋' || r == '－' })
				cell = []string{fmt.Sprintf("%d.", i+1), fmt.Sprintf("%6s", parts[0])}
				operators := []rune{}
				for _, r := range expression {
					if r == '＋' || r == '－' {
						operators = append(operators, r)
					}
				}
				for j, p := range parts[1:] {
					cell = append(cell, fmt.Sprintf("%s%4s", string(operators[j]), p))
				}
				cell = append(cell, "──────", "")
			}
			for _, line := range cell {
				width = max(width, displayWidth(line))
			}
			cells = append(cells, cell)
		}
		for start := 0; start < len(cells); start += columns {
			end := min(start+columns, len(cells))
			height := 1
			for _, cell := range cells[start:end] {
				height = max(height, len(cell))
			}
			for line := 0; line < height; line++ {
				for col := start; col < end; col++ {
					content := ""
					if line < len(cells[col]) {
						content = cells[col][line]
					}
					text.WriteString(content)
					if col < end-1 {
						text.WriteString(strings.Repeat(" ", width-displayWidth(content)+4))
					}
				}
				text.WriteByte('\n')
			}
		}
	}
	if len(e.Questions) > 0 && e.Questions[0].Answer != "" {
		text.WriteString("\n答案\n")
		for i, q := range e.Questions {
			fmt.Fprintf(&text, "%d. %s\n", i+1, q.Answer)
		}
	}
	_, err := io.WriteString(out, text.String())
	return err
}
