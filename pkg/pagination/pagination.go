// Package pagination provides helpers to parse and respond with paginated results.
package pagination

import (
	"net/http"
	"strconv"
)

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 10000
)

// Params holds parsed pagination query parameters.
type Params struct {
	Page   int
	Limit  int
	Sort   string
	Order  string
	Offset int
}

// Parse extracts pagination params from a request's query string.
func Parse(r *http.Request) Params {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = defaultPage
	}

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > maxLimit {
		limit = defaultLimit
	}

	sort := q.Get("sort")
	order := q.Get("order")
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	return Params{
		Page:   page,
		Limit:  limit,
		Sort:   sort,
		Order:  order,
		Offset: (page - 1) * limit,
	}
}

// Meta is the pagination metadata included in list responses.
type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// NewMeta calculates total_pages from total count.
func NewMeta(p Params, total int) Meta {
	totalPages := 0
	if total > 0 {
		totalPages = (total + p.Limit - 1) / p.Limit
	}
	return Meta{
		Page:       p.Page,
		Limit:      p.Limit,
		Total:      total,
		TotalPages: totalPages,
	}
}
