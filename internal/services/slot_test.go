package services

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/FiraBro/local-go/internal/repositories"
)

func TestEndAtFor(t *testing.T) {
	start := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		minutes int
		want    time.Time
	}{
		{"one hour", 60, start.Add(time.Hour)},
		{"minimum slot", 5, start.Add(5 * time.Minute)},
		{"crosses midnight", 14 * 60, time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC)},
		{"full day", 24 * 60, start.Add(24 * time.Hour)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := endAtFor(start, tt.minutes); !got.Equal(tt.want) {
				t.Errorf("endAtFor(%d) = %v, want %v", tt.minutes, got, tt.want)
			}
		})
	}
}

func TestReschedule(t *testing.T) {
	oldStart := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	oldEnd := oldStart.Add(90 * time.Minute)
	plus3 := time.FixedZone("UTC+3", 3*60*60)

	tests := []struct {
		name      string
		newStart  time.Time
		wantStart time.Time
		wantEnd   time.Time
	}{
		{"later day, same duration", time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC),
			time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC), time.Date(2026, 12, 1, 10, 30, 0, 0, time.UTC)},
		{"other zone is the same instant", time.Date(2026, 12, 1, 12, 0, 0, 0, plus3),
			time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC), time.Date(2026, 12, 1, 10, 30, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd := reschedule(oldStart, oldEnd, tt.newStart)
			if !gotStart.Equal(tt.wantStart) || !gotEnd.Equal(tt.wantEnd) {
				t.Errorf("reschedule() = %v, %v; want %v, %v", gotStart, gotEnd, tt.wantStart, tt.wantEnd)
			}
			if gotEnd.Sub(gotStart) != 90*time.Minute {
				t.Errorf("duration changed to %v", gotEnd.Sub(gotStart))
			}
		})
	}
}

func TestTranslateBookingWriteError(t *testing.T) {
	other := errors.New("connection reset")

	tests := []struct {
		name string
		in   error
		want error
	}{
		{"nil stays nil", nil, nil},
		{"conflict", repositories.ErrSlotConflict, ErrBookingConflict},
		{"wrapped conflict", fmt.Errorf("insert: %w", repositories.ErrSlotConflict), ErrBookingConflict},
		{"other error is unchanged", other, other},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translateBookingWriteError(tt.in)
			if tt.want == nil {
				if got != nil {
					t.Errorf("got %v, want nil", got)
				}
				return
			}
			if !errors.Is(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckDuration(t *testing.T) {
	tests := []struct {
		name    string
		minutes int
		wantErr bool
	}{
		{"negative", -1, true},
		{"zero", 0, true},
		{"just below minimum", 4, true},
		{"minimum", 5, false},
		{"typical", 60, false},
		{"maximum", 1440, false},
		{"just above maximum", 1441, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkDuration(tt.minutes)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkDuration(%d) error = %v, wantErr %v", tt.minutes, err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, ErrInvalidInput) {
				t.Errorf("error %v does not wrap ErrInvalidInput", err)
			}
		})
	}
}
