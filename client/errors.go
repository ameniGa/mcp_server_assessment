package client

import (
	"context"
	"errors"
	"fmt"
)

// wrapDoErr turns a low-level http.Client.Do failure into a clear,
// LLM-consumable message.
func wrapDoErr(service string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s: timed out waiting for a response", service)
	}
	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		err = unwrapped
	}
	return fmt.Errorf("%s: could not reach the API: %w", service, err)
}
