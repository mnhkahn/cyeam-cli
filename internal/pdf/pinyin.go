package pdf

import "github.com/mnhkahn/cyeam-cli/internal/pinyin"

// RenderPinyin makes a paginated worksheet with pinyin hints and empty character grids.
func RenderPinyin(src []byte) ([]byte, error) {
	font, err := selectPDFFont([]byte("看拼音写字改错："))
	if err != nil {
		return nil, err
	}
	return pinyin.GenerateSheetPDFWithHeaderFont(string(src), font)
}
