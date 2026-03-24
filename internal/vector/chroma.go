package vector

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// CheckHealth verifies that the configured Chroma endpoint responds to a heartbeat request.
func CheckHealth(ctx context.Context, chromaURL string, httpClient *http.Client) error {
	baseURL := strings.TrimRight(chromaURL, "/")
	paths := []string{"/api/v1/heartbeat", "/api/v2/heartbeat"}

	var lastErr error
	for _, path := range paths {
		pathCtx, cancel := context.WithTimeout(ctx, time.Second)
		req, err := http.NewRequestWithContext(pathCtx, http.MethodGet, baseURL+path, nil)
		if err != nil {
			cancel()
			return fmt.Errorf("build chroma request: %w", err)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		_ = resp.Body.Close()
		cancel()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return fmt.Errorf("chroma healthcheck failed: %w", lastErr)
}
