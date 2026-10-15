package notify

import (
	"context"
	"errors"
)

type Notifier interface {
	Send(ctx context.Context, recipient string, text string, keyboard *InlineKeyboard) error
	ValidateConfig() error
}

var ErrInvalidConfig = errors.New("invalid notifier config")
