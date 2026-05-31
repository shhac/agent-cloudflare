package cli

import (
	"fmt"
	"time"

	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
)

func sinceWindow(since string, now time.Time) (time.Time, time.Time, error) {
	if since == "" {
		since = "1h"
	}
	duration, err := time.ParseDuration(since)
	if err != nil {
		return time.Time{}, time.Time{}, agenterrors.Wrap(err, agenterrors.FixableByAgent).
			WithHint("Use a duration such as 15m, 1h, or 24h")
	}
	if duration <= 0 {
		return time.Time{}, time.Time{}, agenterrors.New(fmt.Sprintf("invalid --since %q", since), agenterrors.FixableByAgent).
			WithHint("Use a positive duration such as 15m, 1h, or 24h")
	}
	end := now.UTC()
	return end.Add(-duration), end, nil
}
