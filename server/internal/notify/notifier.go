package notify

import (
	"context"
	"errors"
)

type Notifier interface {
	Send(ctx context.Context, recipient string, text string) error
	ValidateConfig() error
}

var ErrInvalidConfig = errors.New("invalid notifier config")
