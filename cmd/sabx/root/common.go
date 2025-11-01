package root

import (
	"context"
	"time"
)

const requestTimeout = 15 * time.Second

func timeoutContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, requestTimeout)
}
