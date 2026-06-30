package api

import (
	"bufio"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"
)

const (
	sseBaseDelay  = 1 * time.Second
	sseMaxDelay   = 30 * time.Second
	sseMaxRetries = 6
)

func (c *Client) StreamSSE(ctx context.Context, events chan<- SSEEvent) {
	go func() {
		retries := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			err := c.consumeSSE(ctx, events)
			if err == nil || ctx.Err() != nil {
				return
			}

			retries++
			if retries > sseMaxRetries {
				retries = sseMaxRetries
			}

			delay := time.Duration(float64(sseBaseDelay) * math.Pow(2, float64(retries-1)))
			if delay > sseMaxDelay {
				delay = sseMaxDelay
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}
	}()
}

func (c *Client) consumeSSE(ctx context.Context, events chan<- SSEEvent) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.SSEEventsURL(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	if c.authToken != "" {
		req.Header.Set("codev-web-key", c.authToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	var dataLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, ":") {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
			continue
		}

		if line == "" && len(dataLines) > 0 {
			payload := strings.Join(dataLines, "\n")
			dataLines = nil

			var event SSEEvent
			if err := json.Unmarshal([]byte(payload), &event); err != nil {
				continue
			}

			select {
			case events <- event:
			case <-ctx.Done():
				return nil
			}
		}
	}

	return scanner.Err()
}
