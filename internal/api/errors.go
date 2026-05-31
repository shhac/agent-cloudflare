package api

import (
	"encoding/json"
	"fmt"
	"strings"

	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
)

func classifyHTTPError(status int, body []byte) *agenterrors.APIError {
	msg, code := extractErrorMessage(status, body)
	hints := []string{}
	if code != 0 {
		hints = append(hints, fmt.Sprintf("Cloudflare code: %d", code))
	}

	switch {
	case status == 401:
		return withHint(agenterrors.New("Authentication failed: "+msg, agenterrors.FixableByHuman),
			append(hints, "Check the stored profile with 'agent-cloudflare profiles check' or re-add it with a valid API token")...)
	case status == 403:
		return withHint(agenterrors.New("Permission denied: "+msg, agenterrors.FixableByHuman),
			append(hints, "The token may need a narrower or broader Cloudflare permission group for this account or zone")...)
	case status == 404:
		return withHint(agenterrors.New("Not found: "+msg, agenterrors.FixableByAgent),
			append(hints, "Check the account ID, zone ID, zone name, or resource ID; list commands can rediscover valid IDs")...)
	case status == 429:
		return withHint(agenterrors.New("Rate limited: "+msg, agenterrors.FixableByRetry),
			append(hints, "Wait and retry; reduce --per-page or scope the command to one account/zone")...)
	case status >= 500:
		return withHint(agenterrors.New("Cloudflare API error: "+msg, agenterrors.FixableByRetry),
			append(hints, "Cloudflare returned a server error; retry later")...)
	default:
		return withHint(agenterrors.New(msg, agenterrors.FixableByAgent), hints...)
	}
}

func extractErrorMessage(status int, body []byte) (string, int) {
	var env struct {
		Errors []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		if len(body) > 0 && len(body) <= 200 {
			return fmt.Sprintf("HTTP %d: %s", status, string(body)), 0
		}
		return fmt.Sprintf("HTTP %d", status), 0
	}
	if len(env.Errors) > 0 {
		code := env.Errors[0].Code
		if env.Errors[0].Message != "" {
			return env.Errors[0].Message, code
		}
		return fmt.Sprintf("HTTP %d", status), code
	}
	if env.Error != "" {
		return env.Error, 0
	}
	if env.Message != "" {
		return env.Message, 0
	}
	return fmt.Sprintf("HTTP %d", status), 0
}

func withHint(err *agenterrors.APIError, parts ...string) *agenterrors.APIError {
	filtered := []string{}
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	if len(filtered) > 0 {
		err.Hint = strings.Join(filtered, "; ")
	}
	return err
}
