package api

import (
	"encoding/json"

	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
)

type Envelope struct {
	Success    bool            `json:"success"`
	Result     json.RawMessage `json:"result"`
	Errors     []Message       `json:"errors"`
	Messages   []Message       `json:"messages"`
	ResultInfo *ResultInfo     `json:"result_info,omitempty"`
}

type Message struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type ResultInfo struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	Count      int `json:"count,omitempty"`
	TotalCount int `json:"total_count,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

func DecodeEnvelope(body []byte) (json.RawMessage, *ResultInfo, error) {
	var env Envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, nil, agenterrors.Wrap(err, agenterrors.FixableByAgent).WithHint("Cloudflare returned a response the CLI could not decode")
	}
	if !env.Success {
		msg := "Cloudflare API request failed"
		if len(env.Errors) > 0 && env.Errors[0].Message != "" {
			msg = env.Errors[0].Message
		}
		return nil, env.ResultInfo, agenterrors.New(msg, agenterrors.FixableByAgent)
	}
	return env.Result, env.ResultInfo, nil
}

func DecodeResultList(raw json.RawMessage) ([]json.RawMessage, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent).WithHint("Expected Cloudflare result to be a list")
	}
	return items, nil
}

func DecodeBucketList(raw json.RawMessage) ([]json.RawMessage, error) {
	var direct []json.RawMessage
	if err := json.Unmarshal(raw, &direct); err == nil {
		return direct, nil
	}
	var wrapped struct {
		Buckets []json.RawMessage `json:"buckets"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent).WithHint("Expected Cloudflare R2 result to include a buckets list")
	}
	return wrapped.Buckets, nil
}
