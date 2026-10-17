package services

import (
	"reflect"
	"testing"
	"time"

	"github.com/FiraBro/local-go/internal/repositories"
)

func TestParseClock(t *testing.T) {
	tests := []struct {
		in   string
		want int
		ok   bool
	}{
		{"09:00", 9 * 3600, true},
		{"9:00", 9 * 3600, true},
		{" 08:15 ", 8*3600 + 15*60, true},
		{"17:30", 17*3600 + 30*60, true},
		{"09:00:30", 9*3600 + 30, true},
		{"9:30 PM", 21*3600 + 30*60, true},
		{"9:30pm", 21*3600 + 30*60, true},
		{"12:00 AM", 0, true},
		{"12:00 PM", 12 * 3600, true},
		{"3PM", 15 * 3600, true},
		{"24:00", 24 * 3600, true},
		{"", 0, false},
		{"25:00", 0, false},
		{"09:60", 0, false},
		{"noon", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := parseClock(tt.in)
			if ok != tt.ok || (ok && got != tt.want) {
				t.Errorf("parseClock(%q) = %d, %v; want %d, %v", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestMatchesWeekday(t *testing.T) {
	tests := []struct {
		day  string
		wd   time.Weekday
		want bool
	}{
		{"Monday", time.Monday, true},
		{"monday", time.Monday, true},
		{"MON", time.Monday, true},
		{" Tue ", time.Tuesday, true},
		{"Tuesday", time.Monday, false},
		{"1", time.Monday, false}, // numeric weekdays are not supported
		{"", time.Monday, false},
		{"Funday", time.Monday, false},
	}
	for _, tt := range tests {
		t.Run(tt.day, func(t *testing.T) {
			if got := matchesWeekday(tt.day, tt.wd); got != tt.want {
				t.Errorf("matchesWeekday(%q, %v) = %v, want %v", tt.day, tt.wd, got, tt.want)
			}
		})
	}
}

func at(t *testing.T, loc *time.Location, day, hour, minute int) time.Time {
	t.Helper()
	d := time.Date(2026, 11, day, hour, minute, 0, 0, loc)
	return d
}

func TestFitsShift(t *testing.T) {
	// 2026-11-23 is a Monday, 2026-11-24 a Tuesday.
	if at(t, time.UTC, 23, 0, 0).Weekday() != time.Monday {
		t.Fatal("test calendar is wrong: 23 Nov 2026 should be a Monday")
	}
	utc := time.UTC
	office := []repositories.ShiftRow{{Day: "Monday", Start: "09:00", End: "17:00"}}
	lunch := []repositories.ShiftRow{
		{Day: "Monday", Start: "09:00", End: "12:00"},
		{Day: "Monday", Start: "13:00", End: "17:00"},
	}

	tests := []struct {
		name   string
		shifts []repositories.ShiftRow
		start  time.Time
		end    time.Time
		want   bool
	}{
		{"inside", office, at(t, utc, 23, 9, 0), at(t, utc, 23, 10, 0), true},
		{"ends exactly at closing time", office, at(t, utc, 23, 16, 0), at(t, utc, 23, 17, 0), true},
		{"runs past closing time", office, at(t, utc, 23, 16, 30), at(t, utc, 23, 17, 30), false},
		{"starts before opening", office, at(t, utc, 23, 8, 30), at(t, utc, 23, 9, 30), false},
		{"other weekday", office, at(t, utc, 24, 10, 0), at(t, utc, 24, 11, 0), false},
		{"no shifts at all", nil, at(t, utc, 23, 10, 0), at(t, utc, 23, 11, 0), false},
		{"spans the lunch break", lunch, at(t, utc, 23, 11, 30), at(t, utc, 23, 12, 30), false},
		{"inside the afternoon window", lunch, at(t, utc, 23, 13, 0), at(t, utc, 23, 14, 0), true},
		{"exactly the break", lunch, at(t, utc, 23, 12, 0), at(t, utc, 23, 13, 0), false},
		{"crosses midnight", []repositories.ShiftRow{{Day: "Monday", Start: "00:00", End: "24:00"}},
			at(t, utc, 23, 23, 30), at(t, utc, 24, 0, 30), false},
		{"unreadable shift is ignored", []repositories.ShiftRow{{Day: "Monday", Start: "soon", End: "later"}},
			at(t, utc, 23, 10, 0), at(t, utc, 23, 11, 0), false},
		{"closing before opening is ignored", []repositories.ShiftRow{{Day: "Monday", Start: "17:00", End: "09:00"}},
			at(t, utc, 23, 10, 0), at(t, utc, 23, 11, 0), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fitsShift(tt.start, tt.end, tt.shifts); got != tt.want {
				t.Errorf("fitsShift() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFitsShiftUsesBusinessTimezone(t *testing.T) {
	plus3 := time.FixedZone("UTC+3", 3*3600)
	office := []repositories.ShiftRow{{Day: "Monday", Start: "09:00", End: "17:00"}}

	// The same instant: 07:00-08:00 UTC is 10:00-11:00 in UTC+3.
	startUTC := time.Date(2026, 11, 23, 7, 0, 0, 0, time.UTC)
	endUTC := startUTC.Add(time.Hour)

	if fitsShift(startUTC, endUTC, office) {
		t.Error("07:00 UTC must be outside a 09:00-17:00 shift read in UTC")
	}
	if !fitsShift(startUTC.In(plus3), endUTC.In(plus3), office) {
		t.Error("07:00 UTC is 10:00 local in UTC+3 and must be inside the shift")
	}
}

func TestFilterEligible(t *testing.T) {
	start := time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC) // Monday
	end := start.Add(time.Hour)
	mondayAllDay := []repositories.ShiftRow{{Day: "Monday", Start: "09:00", End: "17:00"}}
	mondayAfternoon := []repositories.ShiftRow{{Day: "Monday", Start: "13:00", End: "17:00"}}

	candidates := []repositories.StaffCandidate{
		{ID: "c1"}, {ID: "c2"}, {ID: "c3"}, {ID: "c4"}, {ID: "c5"},
	}
	holiday := map[string]bool{"c2": true}
	shifts := map[string][]repositories.ShiftRow{
		"c1": mondayAllDay,
		"c2": mondayAllDay,
		"c3": mondayAfternoon,
		// c4 has no schedule rows
		"c5": mondayAllDay,
	}

	got := filterEligible(candidates, holiday, shifts, start, end)

	if want := []string{"c1", "c5"}; !reflect.DeepEqual(got.IDs, want) {
		t.Errorf("IDs = %v, want %v (order must follow the candidates)", got.IDs, want)
	}
	if got.OnHoliday != 1 {
		t.Errorf("OnHoliday = %d, want 1", got.OnHoliday)
	}
	if got.OutsideShift != 2 {
		t.Errorf("OutsideShift = %d, want 2 (c3 works only in the afternoon, c4 has no schedule)", got.OutsideShift)
	}

	if none := filterEligible(nil, nil, nil, start, end); len(none.IDs) != 0 {
		t.Errorf("no candidates must give no eligible staff, got %v", none.IDs)
	}
}
