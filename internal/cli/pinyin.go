package cli

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/mnhkahn/cyeam-cli/internal/output"
	"github.com/mnhkahn/cyeam-cli/internal/pinyin"
	py "github.com/mozillazg/go-pinyin"
	"github.com/spf13/cobra"
)

func newPinyinCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pinyin <text>",
		Short: "Get pinyin for Chinese text",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text := args[0]

			argsPY := py.NewArgs()
			argsPY.Style = py.Tone

			type charPinyin struct {
				Char   string `json:"char"`
				Pinyin string `json:"pinyin"`
			}
			var result []charPinyin
			for _, r := range text {
				pys := py.Pinyin(string(r), argsPY)
				cp := charPinyin{Char: string(r)}
				if len(pys) > 0 && len(pys[0]) > 0 {
					cp.Pinyin = pys[0][0]
				}
				result = append(result, cp)
			}

			body, _ := json.Marshal(map[string][]charPinyin{"pinyin": result})
			return output.WriteJSON(cmd.OutOrStdout(), body)
		},
	}
	cmd.AddCommand(newPinyinSheetCommand())
	return cmd
}

func newPinyinSheetCommand() *cobra.Command {
	var outFile string
	cmd := &cobra.Command{
		Use:   "sheet <text>",
		Short: "Generate pinyin practice worksheet PDF",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text := args[0]

			pdfData, err := pinyin.GenerateSheetPDF(text)
			if err != nil {
				return fmt.Errorf("generate pdf: %w", err)
			}

			if outFile != "" {
				if err := output.WriteFile(outFile, pdfData); err != nil {
					return err
				}
				_, err := cmd.OutOrStdout().Write([]byte("saved: " + outFile + "\n"))
				return err
			}

			body, _ := json.Marshal(map[string]string{
				"pinyin": text,
				"pdf":    base64.StdEncoding.EncodeToString(pdfData),
			})
			return output.WriteJSON(cmd.OutOrStdout(), body)
		},
	}
	cmd.Flags().StringVarP(&outFile, "out", "o", "", "save PDF to file")
	return cmd
}