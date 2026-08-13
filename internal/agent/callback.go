package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rezexell/bashtt/internal/domain"
)

type CallbackSender struct {
	client *http.Client
	url    string
}

func NewCallbackSender(
	client *http.Client,
	url string,
) *CallbackSender {
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	return &CallbackSender{
		client: client,
		url:    url,
	}
}

type callbackRequest struct {
	User   string             `json:"user"`
	Script string             `json:"script"`
	Action domain.EventAction `json:"action"`
	Time   time.Time          `json:"time"`
}

func (s *CallbackSender) Send(
	ctx context.Context,
	event domain.Event,
) error {
	if !event.Action.IsValid() {
		return fmt.Errorf(
			"invalid event action: %q",
			event.Action,
		)
	}

	payload := callbackRequest{
		User:   event.User,
		Script: event.Script,
		Action: event.Action,
		Time:   event.CreatedAt,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf(
			"marshal callback: %w",
			err,
		)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.url,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf(
			"create callback request: %w",
			err,
		)
	}

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf(
			"send callback: %w",
			err,
		)
	}

	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf(
			"callback returned status %d",
			resp.StatusCode,
		)
	}

	return nil
}
