package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mnhkahn/cyeam-cli/internal/output"
	"github.com/mnhkahn/cyeam-cli/internal/wiring"
	"github.com/spf13/cobra"
)

func newWiringCommand() *cobra.Command {
	var out, baseSVG string
	render := &cobra.Command{
		Use:   "render <sketch.fzz|sketch.fz>",
		Short: "Export an interactive breadboard SVG",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sketch := args[0]
			if out == "" {
				out = strings.TrimSuffix(filepath.Base(sketch), filepath.Ext(sketch)) + ".interactive.svg"
			}
			if !strings.EqualFold(filepath.Ext(out), ".svg") {
				return fmt.Errorf("the output path must end in .svg")
			}
			data, err := wiring.Render(cmd.Context(), sketch, baseSVG)
			if err != nil {
				return err
			}
			if err := output.WriteFile(out, data); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "saved:", out)
			return err
		},
	}
	render.Flags().StringVarP(&out, "out", "o", "", "output SVG path (default: <sketch>.interactive.svg)")
	render.Flags().StringVar(&baseSVG, "base-svg", "", "matching breadboard SVG previously exported by fritzing-render")
	cmd := &cobra.Command{
		Use:   "wiring",
		Short: "Render Fritzing wiring diagrams",
	}
	cmd.AddCommand(render)
	return cmd
}
