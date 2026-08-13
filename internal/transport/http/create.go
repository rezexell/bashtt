package httptransport

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rezexell/bashtt/internal/domain"
	"github.com/rezexell/bashtt/internal/service"
)

type createRequest struct {
	Host     string `json:"host"`
	User     string `json:"user"`
	Password string `json:"password"`
	Template string `json:"template"`
}

type createResponse struct {
	ID        string `json:"id"`
	MachineID string `json:"machine_id"`
	Path      string `json:"path"`
	Template  string `json:"template"`
}

type CreateHandler struct {
	service *service.CreateService
}

func NewCreateHandler(
	service *service.CreateService,
) *CreateHandler {
	return &CreateHandler{
		service: service,
	}
}

func (h *CreateHandler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req createRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			"invalid JSON",
			http.StatusBadRequest,
		)
		return
	}

	result, err := h.service.Create(
		r.Context(),
		service.CreateRequest{
			Host:     req.Host,
			User:     req.User,
			Password: req.Password,
			Template: domain.Template(req.Template),
		},
	)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidTemplate) {
			http.Error(
				w,
				err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(
		createResponse{
			ID:        result.ID.String(),
			MachineID: result.MachineID.String(),
			Path:      result.Path,
			Template:  string(result.Template),
		},
	)
}
