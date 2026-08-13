package httptransport

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rezexell/bashtt/internal/domain"
	"github.com/rezexell/bashtt/internal/service"
)

type CallbackHandler struct {
	service *service.CallbackService
}

func NewCallbackHandler(
	service *service.CallbackService,
) *CallbackHandler {
	return &CallbackHandler{
		service: service,
	}
}

type callbackRequest struct {
	User   string `json:"user"`
	Script string `json:"script"`
	Action string `json:"action"`
	Time   string `json:"time"`
}

func (h *CallbackHandler) Callback(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req callbackRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)

		return
	}

	eventTime, err := time.Parse(
		time.RFC3339,
		req.Time,
	)
	if err != nil {
		http.Error(
			w,
			"invalid time",
			http.StatusBadRequest,
		)

		return
	}

	err = h.service.Handle(
		r.Context(),
		service.CallbackRequest{
			User:   req.User,
			Script: req.Script,
			Action: domain.EventAction(req.Action),
			Time:   eventTime,
		},
	)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)

		return
	}

	w.WriteHeader(http.StatusOK)
}
