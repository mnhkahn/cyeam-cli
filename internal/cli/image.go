package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	img "github.com/mnhkahn/cyeam-cli/internal/image"
	"github.com/mnhkahn/cyeam-cli/internal/output"
	"github.com/spf13/cobra"
)

func newImageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Image tools - format conversion and resizing",
	}
	cmd.AddCommand(newImageConvertCommand())
	return cmd
}

func newImageConvertCommand() *cobra.Command {
	var (
		format    string
		outFile   string
		quality   int
		width     int
		height    int
		keepRatio bool
	)
	cmd := &cobra.Command{
		Use:   "convert <input>",
		Short: "Convert an image to another format",
		Long: `Convert an image between formats, optionally resizing.

Input is a file path or "-" for stdin. Source format (jpg/png/gif/webp) is
auto-detected; SVG input is rasterized and recognized by the .svg extension.

Target formats: jpg, png, webp, gif, ico, base64.
- webp output is lossless (VP8L); --quality applies to jpg only.
- ico produces a multi-size favicon (16/32/48 px).
- base64 prints a PNG data URL to stdout.

Without --out, the result is written to "<input-name>.<ext>" in the current
directory (except base64, which prints to stdout).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format = strings.ToLower(format)
			if !img.ValidFormat(format) {
				return fmt.Errorf("unsupported --format %q (supported: %s)", format, strings.Join(img.Formats, ", "))
			}
			if quality < 1 || quality > 100 {
				return fmt.Errorf("--quality must be 1-100, got %d", quality)
			}

			input := args[0]
			var src []byte
			var err error
			fromStdin := input == "-"
			if fromStdin {
				src, err = io.ReadAll(cmd.InOrStdin())
			} else {
				src, err = os.ReadFile(input)
			}
			if err != nil {
				return fmt.Errorf("read input: %w", err)
			}
			if len(src) == 0 {
				return fmt.Errorf("no input data")
			}

			isSVG := looksLikeSVG(input, src)
			decoded, err := img.Decode(bytes.NewReader(src), isSVG, width, height)
			if err != nil {
				return err
			}
			// SVG rasterization already honors width/height; bitmaps resize here.
			if !isSVG {
				decoded = img.Resize(decoded, width, height, keepRatio)
			}

			var data []byte
			if format == "ico" {
				data, err = img.EncodeICO(decoded)
			} else {
				data, err = img.Encode(decoded, format, quality)
			}
			if err != nil {
				return err
			}

			// base64: print the data URL text (respecting -o if given).
			if format == "base64" && outFile == "" {
				_, err := cmd.OutOrStdout().Write(append(data, '\n'))
				return err
			}

			dest := outFile
			if dest == "" {
				if fromStdin {
					return fmt.Errorf("--out is required when reading from stdin")
				}
				dest = deriveOutPath(input, format)
			}
			if err := output.WriteFile(dest, data); err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write([]byte("saved: " + dest + "\n"))
			return err
		},
	}
	cmd.Flags().StringVarP(&format, "format", "f", "", "target format: jpg, png, webp, gif, ico, base64 (required)")
	cmd.Flags().StringVarP(&outFile, "out", "o", "", "output file path")
	cmd.Flags().IntVarP(&quality, "quality", "q", 90, "jpg quality 1-100")
	cmd.Flags().IntVarP(&width, "width", "w", 0, "target width in px")
	cmd.Flags().IntVarP(&height, "height", "H", 0, "target height in px")
	cmd.Flags().BoolVar(&keepRatio, "keep-ratio", true, "preserve aspect ratio when only one dimension is set")
	cmd.MarkFlagRequired("format")
	return cmd
}

func looksLikeSVG(input string, src []byte) bool {
	if strings.EqualFold(filepath.Ext(input), ".svg") {
		return true
	}
	head := src
	if len(head) > 512 {
		head = head[:512]
	}
	return bytes.Contains(head, []byte("<svg"))
}

func deriveOutPath(input, format string) string {
	ext := format
	if format == "base64" {
		ext = "txt"
	}
	base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	if base == "" {
		base = "converted-image"
	}
	return base + "." + ext
}
