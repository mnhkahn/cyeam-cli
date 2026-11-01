package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/paddleocr"
)

func recognizeArithmeticWithRetry(ctx context.Context, client *paddleocr.Client, upload []byte, timeout time.Duration, retries int, timings bool, out io.Writer) ([]byte, error) {
	for attempt := 1; attempt <= retries+1; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		var timing paddleocr.RequestTiming
		if timings {
			client.Timing = &timing
		}
		data, err := client.RecognizeRaw(attemptCtx, upload)
		cancel()
		if timings {
			fmt.Fprintf(out, "OCR第%d次耗时：准备/连接 %.3fs，上传 %.3fs，等待响应 %.3fs，下载 %.3fs，总计 %.3fs\n", attempt, timing.PrepareConnectMS/1000, timing.UploadMS/1000, timing.WaitMS/1000, timing.DownloadMS/1000, timing.TotalMS/1000)
		}
		if err == nil {
			return data, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		if attempt == retries+1 {
			return nil, fmt.Errorf("OCR timed out after %d attempts (%s each): %w", attempt, timeout, err)
		}
		fmt.Fprintf(out, "OCR第%d次请求超时（%s），正在重试（%d/%d）\n", attempt, timeout, attempt+1, retries+1)
	}
	panic("unreachable")
}
