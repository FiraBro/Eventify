package services

import (
	"strings"
	"time"

	"github.com/FiraBro/local-go/internal/repositories"
)

// ASSUMPTION about how staff_schedule stores data (confirm with SELECT DISTINCT on the table):
//   day_of_week: an English weekday name, full or three letters ("Monday", "mon"), any case.
//   start_time / end_time: a clock time such as "09:00", "9:00", "09:00:00" or "9:30 PM".
// Rows in any other format are ignored, so a staff member whose rows cannot be read is never
// assigned automatically. That fails safe: the booking is refused, never put on the wrong person.

var weekdayNames = map[string]time.Weekday{
	"sunday": time.Sunday, "sun": time.Sunday,
	"monday": time.Monday, "mon": time.Monday,
	"tuesday": time.Tuesday, "tue": time.Tuesday, "tues": time.Tuesday,
	"wednesday": time.Wednesday, "wed": time.Wednesday,
	"thursday": time.Thursday, "thu": time.Thursday, "thur": time.Thursday, "thurs": time.Thursday,
	"friday": time.Friday, "fri": time.Friday,
	"saturday": time.Saturday, "sat": time.Saturday,
}

func matchesWeekday(day string, wd time.Weekday) bool {
	got, ok := weekdayNames[strings.ToLower(strings.TrimSpace(day))]
	return ok && got == wd
}

var clockLayouts = []string{"15:04", "15:04:05", "3:04PM", "3:04 PM", "3PM", "3 PM"}

// parseClock returns the number of seconds after midnight for a clock time.
func parseClock(s string) (seconds int, ok bool) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "24:00" || s == "24:00:00" {
		return 24 * 3600, true
	}
	for _, layout := range clockLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Hour()*3600 + t.Minute()*60 + t.Second(), true
		}
	}
	return 0, false
}

// fitsShift reports whether [localStart, localEnd) lies entirely inside one working window of that
// weekday. localStart and localEnd must already be expressed in the business timezone.
func fitsShift(localStart, localEnd time.Time, shifts []repositories.ShiftRow) bool {
	startSec := localStart.Hour()*3600 + localStart.Minute()*60 + localStart.Second()
	endSec := startSec + int(localEnd.Sub(localStart).Seconds())
	if endSec > 24*3600 {
		return false // must finish on the same local day
	}

	for _, sh := range shifts {
		if !matchesWeekday(sh.Day, localStart.Weekday()) {
			continue
		}
		open, okOpen := parseClock(sh.Start)
		closeAt, okClose := parseClock(sh.End)
		if !okOpen || !okClose || closeAt <= open {
			continue
		}
		if startSec >= open && endSec <= closeAt {
			return true
		}
	}
	return false
}

type eligibility struct {
	IDs          []string // in the order of the candidates (least busy first)
	OnHoliday    int
	OutsideShift int
}

// filterEligible keeps the candidates who are not on holiday and are working during the slot.
func filterEligible(
	candidates []repositories.StaffCandidate,
	onHoliday map[string]bool,
	shifts map[string][]repositories.ShiftRow,
	localStart, localEnd time.Time,
) eligibility {
	var out eligibility
	for _, c := range candidates {
		switch {
		case onHoliday[c.ID]:
			out.OnHoliday++
		case !fitsShift(localStart, localEnd, shifts[c.ID]):
			out.OutsideShift++
		default:
			out.IDs = append(out.IDs, c.ID)
		}
	}
	return out
}
