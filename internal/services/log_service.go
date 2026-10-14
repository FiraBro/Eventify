package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/google/uuid"
)

var (
	ErrLogForbidden    = errors.New("only admins can view logs")
	ErrLogInvalidInput = errors.New("invalid log filter")
)

const maxLogDetailsBytes = 4096

var loggedMethods = map[string]bool{"POST": true, "PUT": true, "PATCH": true, "DELETE": true}

type LogService struct {
	repo *repositories.LogRepository
}

func NewLogService(repo *repositories.LogRepository) *LogService {
	return &LogService{repo: repo}
}

// Record stores one entry. Called by the ActivityLogger middleware.
func (s *LogService) Record(ctx context.Context, entry *models.ActivityLog) error {
	if len(entry.Details) > 0 {
		if len(entry.Details) > maxLogDetailsBytes || !json.Valid(entry.Details) {
			entry.Details = json.RawMessage(`{"truncated":true}`)
		}
	}
	return s.repo.Create(ctx, entry)
}

func logInvalid(msg string) error {
	return fmt.Errorf("%w: %s", ErrLogInvalidInput, msg)
}

func escapeLike(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(q)
}

// List returns a page of logs, newest first. Admin only.
func (s *LogService) List(ctx context.Context, role string, p *models.LogListParams) (*models.LogPage, error) {
	if role != roleAdmin {
		return nil, ErrLogForbidden
	}

	page, limit := p.Page, p.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	f := &models.LogFilter{Limit: limit, Offset: (page - 1) * limit}

	if p.UserID != "" {
		if _, err := uuid.Parse(p.UserID); err != nil {
			return nil, logInvalid("user_id must be a valid id")
		}
		f.UserID = p.UserID
	}
	if p.Method != "" {
		m := strings.ToUpper(p.Method)
		if !loggedMethods[m] {
			return nil, logInvalid("method must be POST, PUT, PATCH or DELETE")
		}
		f.Method = m
	}
	if p.Status != "" {
		code, err := strconv.Atoi(p.Status)
		if err != nil || code < 100 || code > 599 {
			return nil, logInvalid("status must be an HTTP status code such as 200 or 404")
		}
		f.Status = code
	}
	if q := strings.TrimSpace(p.Q); q != "" {
		f.Q = escapeLike(q)
	}

	// optional dates (YYYY-MM-DD, both inclusive, UTC)
	if p.From != "" {
		d, err := time.Parse(reportDateLayout, p.From)
		if err != nil {
			return nil, logInvalid("'from' must be YYYY-MM-DD")
		}
		f.From = &d
	}
	if p.To != "" {
		d, err := time.Parse(reportDateLayout, p.To)
		if err != nil {
			return nil, logInvalid("'to' must be YYYY-MM-DD")
		}
		end := d.AddDate(0, 0, 1)
		f.To = &end
	}
	if f.From != nil && f.To != nil && !f.From.Before(*f.To) {
		return nil, logInvalid("'from' must not be after 'to'")
	}

	items, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, err
	}
	return &models.LogPage{Items: items, Total: total, Page: page, Limit: limit}, nil
}