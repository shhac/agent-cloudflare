package api

import (
	"encoding/json"
	"fmt"

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
		return agenterrors.New("Authentication failed: "+msg, agenterrors.FixableByHuman).
			WithHints(append(hints,
				"Run 'agent-cloudflare profiles check' to verify the active profile",
				"Re-add the token with 'agent-cloudflare profiles update <profile> --form' if it was revoked or copied incorrectly")...)
	case status == 403:
		return agenterrors.New("Permission denied: "+msg, agenterrors.FixableByHuman).
			WithHints(append(hints,
				"The token may need additional Cloudflare permission groups for this account or zone",
				"Confirm --account-id/--zone-id target the resource granted to the token")...)
	case status == 404:
		return agenterrors.New("Not found: "+msg, agenterrors.FixableByAgent).
			WithHints(append(hints,
				"Check the account ID, zone ID, zone name, or resource ID",
				"Use list commands such as 'agent-cloudflare zones list' or 'agent-cloudflare dns list <zone>' to rediscover valid IDs")...)
	case status == 429:
		return agenterrors.New("Rate limited: "+msg, agenterrors.FixableByRetry).
			WithHints(append(hints,
				"Wait and retry",
				"Reduce the command scope to one account/zone or a smaller time window")...)
	case status >= 500:
		return agenterrors.New("Cloudflare API error: "+msg, agenterrors.FixableByRetry).
			WithHints(append(hints,
				"Cloudflare returned a server error; retry later",
				"Use --debug to capture redacted request metadata if the issue persists")...)
	default:
		return agenterrors.New(msg, agenterrors.FixableByAgent).
			WithHints(append(hints, "Check the command arguments and Cloudflare resource identifiers")...)
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

func classifyGraphQLError(body []byte) *agenterrors.APIError {
	var parsed struct {
		Errors []struct {
			Message    string `json:"message"`
			Path       []any  `json:"path"`
			Extensions struct {
				Code string `json:"code"`
			} `json:"extensions"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Errors) == 0 {
		return nil
	}
	first := parsed.Errors[0]
	msg := first.Message
	if msg == "" {
		msg = "Cloudflare GraphQL query failed"
	}
	hints := []string{
		"Check that the token has Analytics Read permissions for the zone/account",
		"GraphQL analytics availability can vary by plan, dataset, and time window",
	}
	if first.Extensions.Code != "" {
		hints = append([]string{"GraphQL code: " + first.Extensions.Code}, hints...)
	}
	return agenterrors.New("Cloudflare GraphQL error: "+msg, agenterrors.FixableByHuman).WithHints(hints...)
}
