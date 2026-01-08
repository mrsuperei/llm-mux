package executor

import (
	"context"
	"net/http"
	"sync"
	"time"
)

var antigravityEndpoints = []string{
	"https://cloudcode-pa.googleapis.com",
	"https://oauth2.googleapis.com",
}

func PrewarmAntigravityConnections(ctx context.Context) {
	var wg sync.WaitGroup
	timeout := 5 * time.Second

	for _, endpoint := range antigravityEndpoints {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			prewarmHTTP(ctx, url, timeout)
		}(endpoint)
	}

	wg.Wait()
}

func prewarmHTTP(ctx context.Context, baseURL string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, baseURL, nil)
	if err != nil {
		return
	}

	client := &http.Client{
		Transport: SharedTransport,
		Timeout:   timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}
