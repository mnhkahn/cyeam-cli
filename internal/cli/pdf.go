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

func newPDFCommand() *cobra.Command {
	var outFile string
	cmd := &cobra.Command{
		Use:   "pdf [file]",
		Short: "Convert markdown, HTML, or Typst to PDF",
		Long: `Convert markdown, HTML, or Typst content to PDF.

Reads from a file or stdin. Format is auto-detected:
- Files with .typ extension are treated as Typst
- Files with .html/.htm extension or content starting with <!DOCTYPE/<html are treated as HTML
- Everything else is treated as Markdown`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			if ext == ".typ" {
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
	cmd.Flags().StringVarP(&outFile, "out", "o", "", "save PDF to file")
	return cmd
}
