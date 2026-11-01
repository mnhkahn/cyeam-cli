package paddleocr

import (
	"context"
	"net/http/httptrace"
	"sync/atomic"
	"time"
)

// RequestTiming measures client-observed phases. Wait includes remote processing
// and network round-trip time; it cannot separate server queueing from inference.
type RequestTiming struct {
	PrepareConnectMS float64 `json:"prepare_connect_ms"`
	UploadMS         float64 `json:"upload_ms"`
	WaitMS           float64 `json:"wait_ms"`
	DownloadMS       float64 `json:"download_ms"`
	TotalMS          float64 `json:"total_ms"`
}

func traceRequest(ctx context.Context, started time.Time, dst *RequestTiming) (context.Context, func()) {
	if dst == nil {
		return ctx, func() {}
	}
	var connected, written, first atomic.Int64
	elapsed := func() int64 { return time.Since(started).Nanoseconds() }
	trace := &httptrace.ClientTrace{
		GotConn:              func(httptrace.GotConnInfo) { connected.Store(elapsed()) },
		WroteRequest:         func(httptrace.WroteRequestInfo) { written.Store(elapsed()) },
		GotFirstResponseByte: func() { first.Store(elapsed()) },
	}
	done := func() {
		total := elapsed()
		c, w, f := connected.Load(), written.Load(), first.Load()
		if c == 0 {
			c = total
		}
		if w == 0 {
			w = total
		}
		if f == 0 {
			f = total
		}
		ms := func(n int64) float64 { return float64(max(n, 0)) / 1e6 }
		*dst = RequestTiming{PrepareConnectMS: ms(c), UploadMS: ms(w - c), WaitMS: ms(f - w), DownloadMS: ms(total - f), TotalMS: ms(total)}
	}
	return httptrace.WithClientTrace(ctx, trace), done
}
