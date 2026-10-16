package repositories

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lib/pq"
)

func TestMapBookingWriteError(t *testing.T) {
	plain := errors.New("connection reset")
	overlap := &pq.Error{Code: "23P01", Constraint: "bookings_no_staff_overlap"}

	tests := []struct {
		name         string
		in           error
		wantConflict bool
	}{
		{"overlap violation", overlap, true},
		{"wrapped overlap violation", fmt.Errorf("exec: %w", overlap), true},
		{"exclusion violation of another constraint", &pq.Error{Code: "23P01", Constraint: "some_other_constraint"}, false},
		{"unique violation", &pq.Error{Code: "23505", Constraint: "bookings_no_staff_overlap"}, false},
		{"plain error", plain, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapBookingWriteError(tt.in)
			if tt.wantConflict {
				if !errors.Is(got, ErrSlotConflict) {
					t.Errorf("got %v, want ErrSlotConflict", got)
				}
				return
			}
			if errors.Is(got, ErrSlotConflict) {
				t.Errorf("got ErrSlotConflict for %v", tt.in)
			}
			if got != tt.in {
				t.Errorf("error changed: got %v, want %v", got, tt.in)
			}
		})
	}

	if got := mapBookingWriteError(nil); got != nil {
		t.Errorf("nil input produced %v", got)
	}
}
