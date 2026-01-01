package api

import (
	"github.com/bher20/eratemanager/internal/rates"
)

// RefreshResponse is the response structure for refresh endpoints.
type RefreshResponse struct {
	Provider string               `json:"provider"`
	PDFURL   string               `json:"pdf_url"`
	Path     string               `json:"path"`
	Status   string               `json:"status"`
	Error    string               `json:"error,omitempty"`
	Rates    *rates.RatesResponse `json:"rates,omitempty"`
}
