package output

import (
	"io"
	"os"
)

func WriteJSON(out io.Writer, body []byte) error {
	if _, err := out.Write(body); err != nil {
		return err
	}
	_, err := out.Write([]byte("\n"))
	return err
}

func WriteFile(path string, body []byte) error {
	return os.WriteFile(path, body, 0644)
}
