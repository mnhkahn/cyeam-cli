package cli

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mnhkahn/cyeam-cli/internal/output"
	"github.com/mnhkahn/cyeam-cli/internal/pdf"
	"github.com/spf13/cobra"
)

var renderMarkdownPDF = pdf.RenderMarkdown
var renderHTMLPDF = pdf.RenderHTML
var renderTypstPDF = pdf.RenderTypst
var renderArithmeticPDF = pdf.RenderArithmetic
var renderPinyinPDF = pdf.RenderPinyin

func newPDFCommand() *cobra.Command {
	var outFile, mode, layout, title string
	cmd := &cobra.Command{
		Use:   "pdf [file]",
		Short: "Convert markdown, HTML, or Typst to PDF",
		Long: `Convert markdown, HTML, or Typst content to PDF.

Reads from a file or stdin. Format is auto-detected:
- Files with .typ extension are treated as Typst
- Files with .html/.htm extension or content starting with <!DOCTYPE/<html are treated as HTML
- Everything else is treated as Markdown

Use --mode arithmetic for plain text with one question per nonempty line.
Select --layout standard, grade4, large-number, negative, quantity, or vertical.
Arithmetic mode renders a native A4 worksheet without Typst.
Use --mode pinyin for Chinese text, with pinyin hints, blank character grids,
and a correction area. Spaces or newlines separate words; long input spans pages.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if mode != "auto" && mode != "arithmetic" && mode != "pinyin" {
				return fmt.Errorf("unknown PDF mode %q (use auto, arithmetic, or pinyin)", mode)
			}
			if mode != "arithmetic" && (cmd.Flags().Changed("layout") || cmd.Flags().Changed("title")) {
				return fmt.Errorf("--layout and --title require --mode arithmetic")
			}
			var src []byte
			var err error

			if len(args) == 1 {
				src, err = os.ReadFile(args[0])
				if err != nil {
					return fmt.Errorf("read file: %w", err)
				}
			} else {
				src, err = io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return fmt.Errorf("read stdin: %w", err)
				}
				if len(strings.TrimSpace(string(src))) == 0 {
					return fmt.Errorf("no content provided (stdin is empty)")
				}
			}

			var pdfData []byte
			ext := ""
			if len(args) == 1 {
				ext = strings.ToLower(filepath.Ext(args[0]))
			}
			if mode == "arithmetic" {
				pdfData, err = renderArithmeticPDF(src, pdf.ArithmeticOptions{Layout: layout, Title: title})
			} else if mode == "pinyin" {
				pdfData, err = renderPinyinPDF(src)
			} else if ext == ".typ" {
				pdfData, err = renderTypstPDF(src)
			} else if ext == ".html" || ext == ".htm" || pdf.IsHTML(src) {
				pdfData, err = renderHTMLPDF(src)
			} else {
				pdfData, err = renderMarkdownPDF(src)
			}
			if err != nil {
				return fmt.Errorf("render pdf: %w", err)
			}

			if outFile != "" {
				if err := output.WriteFile(outFile, pdfData); err != nil {
					return err
				}
				_, err := cmd.OutOrStdout().Write([]byte("saved: " + outFile + "\n"))
				return err
			}

			body, _ := json.Marshal(map[string]string{
				"pdf": base64.StdEncoding.EncodeToString(pdfData),
			})
			return output.WriteJSON(cmd.OutOrStdout(), body)
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "auto", "PDF mode: auto, arithmetic, or pinyin")
	cmd.Flags().StringVar(&layout, "layout", "standard", "arithmetic layout: standard, grade4, large-number, negative, quantity, vertical")
	cmd.Flags().StringVar(&title, "title", "", "arithmetic worksheet title (default 宝宝口算)")
	cmd.Flags().StringVarP(&outFile, "out", "o", "", "save PDF to file")
	return cmd
}
