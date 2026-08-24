package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// wrapDoErr turns a low-level http.Client.Do failure into a clear,
// LLM-consumable message.
func wrapDoErr(service string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s: timed out waiting for a response; safe to retry", service)
	}
	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		err = unwrapped
	}
	return fmt.Errorf("%s: could not reach the API: %w", service, err)
}

// classifyStatus turns a non-200 HTTP status into a short, LLM-consumable
// reason that also says whether retrying the same request is likely to help.
func classifyStatus(statusCode int) string {
	switch {
	case statusCode == http.StatusTooManyRequests:
		return "rate limited by upstream (status 429); retrying after a short wait may help"
	case statusCode >= 500:
		return fmt.Sprintf("upstream service unavailable (status %d); safe to retry", statusCode)
	default:
		return fmt.Sprintf("request rejected by upstream (status %d); retrying with the same input will not help", statusCode)
	}
}
