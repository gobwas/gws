package async

import (
	"github.com/gobwas/gws/client/ev"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Handle(loop *ev.Loop, data interface{}, cb ev.Callback) error {
	cb(nil, data)
	return nil
}

func (h *Handler) IsActive() bool {
	return false
}
