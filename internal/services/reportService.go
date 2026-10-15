package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
)

var (
	ErrReportForbidden    = errors.New("only admins can view reports")
	ErrReportInvalidInput = errors.New("invalid report parameters")
)

const (
	reportDateLayout   = "2006-01-02"
	reportDefaultDays  = 30
	reportMaxRangeDays = 366
	reportCurrency     = "ETB"
)

type ReportService struct {
	repo *repositories.ReportRepository
}

func NewReportService(repo *repositories.ReportRepository) *ReportService {
	return &ReportService{repo: repo}
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func percent(part, whole int) float64 {
	if whole == 0 {
		return 0
	}
	return round2(float64(part) / float64(whole) * 100)
}

// dateRange validates ?from=&to= (YYYY-MM-DD, both inclusive, UTC).
// Defaults to the last 30 days. Returns [start, end) timestamps for SQL.
func dateRange(from, to string) (start, end time.Time, p models.ReportPeriod, err error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	toDay := today
	if to != "" {
		if toDay, err = time.Parse(reportDateLayout, to); err != nil {
			return start, end, p, fmt.Errorf("%w: 'to' must be YYYY-MM-DD", ErrReportInvalidInput)
		}
	}
	fromDay := toDay.AddDate(0, 0, -(reportDefaultDays - 1))
	if from != "" {
		if fromDay, err = time.Parse(reportDateLayout, from); err != nil {
			return start, end, p, fmt.Errorf("%w: 'from' must be YYYY-MM-DD", ErrReportInvalidInput)
		}
	}

	if fromDay.After(toDay) {
		return start, end, p, fmt.Errorf("%w: 'from' must not be after 'to'", ErrReportInvalidInput)
	}
	if toDay.Sub(fromDay) > time.Duration(reportMaxRangeDays)*24*time.Hour {
		return start, end, p, fmt.Errorf("%w: range cannot exceed %d days", ErrReportInvalidInput, reportMaxRangeDays)
	}

	p = models.ReportPeriod{From: fromDay.Format(reportDateLayout), To: toDay.Format(reportDateLayout)}
	return fromDay, toDay.AddDate(0, 0, 1), p, nil
}

func normalizeLimit(limit int) int {
	if limit < 1 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func requireAdmin(role string) error {
	if role != roleAdmin {
		return ErrReportForbidden
	}
	return nil
}

// --------------------
// BOOKINGS REPORT (by scheduled date)
// --------------------
func (s *ReportService) Bookings(ctx context.Context, role, from, to string) (*models.BookingsReport, error) {
	if err := requireAdmin(role); err != nil {
		return nil, err
	}
	start, end, period, err := dateRange(from, to)
	if err != nil {
		return nil, err
	}

	counts, err := s.repo.BookingsByStatus(ctx, start, end)
	if err != nil {
		return nil, err
	}
	daily, err := s.repo.BookingsDaily(ctx, start, end)
	if err != nil {
		return nil, err
	}

	byStatus := map[string]int{bookingPending: 0, bookingConfirmed: 0, bookingCancelled: 0, bookingCompleted: 0}
	total := 0
	for status, n := range counts {
		byStatus[status] = n
		total += n
	}

	return &models.BookingsReport{
		ReportPeriod:     period,
		Total:            total,
		ByStatus:         byStatus,
		CancellationRate: percent(byStatus[bookingCancelled], total),
		Daily:            daily,
	}, nil
}

// --------------------
// REVENUE REPORT (by payment date)
// --------------------
func (s *ReportService) Revenue(ctx context.Context, role, from, to, groupBy string) (*models.RevenueReport, error) {
	if err := requireAdmin(role); err != nil {
		return nil, err
	}
	start, end, period, err := dateRange(from, to)
	if err != nil {
		return nil, err
	}

	switch groupBy {
	case "":
		groupBy = "day"
	case "day", "week", "month":
	default:
		return nil, fmt.Errorf("%w: group_by must be day, week or month", ErrReportInvalidInput)
	}

	totals, err := s.repo.RevenueTotals(ctx, start, end)
	if err != nil {
		return nil, err
	}
	series, err := s.repo.RevenueSeries(ctx, start, end, groupBy)
	if err != nil {
		return nil, err
	}
	for i := range series {
		series[i].Net = round2(series[i].Gross - series[i].Refunded)
	}

	avg := 0.0
	if totals.Payments > 0 {
		avg = round2(totals.Gross / float64(totals.Payments))
	}

	return &models.RevenueReport{
		ReportPeriod:   period,
		Currency:       reportCurrency,
		GroupBy:        groupBy,
		GrossRevenue:   round2(totals.Gross),
		Refunded:       round2(totals.Refunded),
		NetRevenue:     round2(totals.Gross - totals.Refunded),
		PaymentsCount:  totals.Payments,
		RefundsCount:   totals.Refunds,
		AveragePayment: avg,
		Series:         series,
	}, nil
}

// --------------------
// SERVICES REPORT (by scheduled date)
// --------------------
func (s *ReportService) Services(ctx context.Context, role, from, to string, limit int) (*models.ServicesReport, error) {
	if err := requireAdmin(role); err != nil {
		return nil, err
	}
	start, end, period, err := dateRange(from, to)
	if err != nil {
		return nil, err
	}

	rows, err := s.repo.ServicePerformance(ctx, start, end, normalizeLimit(limit))
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].Revenue = round2(rows[i].Revenue)
	}
	return &models.ServicesReport{ReportPeriod: period, Services: rows}, nil
}

// --------------------
// STAFF PERFORMANCE REPORT (by scheduled date)
// --------------------
func (s *ReportService) StaffPerformance(ctx context.Context, role, from, to string, limit int) (*models.StaffPerformanceReport, error) {
	if err := requireAdmin(role); err != nil {
		return nil, err
	}
	start, end, period, err := dateRange(from, to)
	if err != nil {
		return nil, err
	}

	rows, err := s.repo.StaffPerformance(ctx, start, end, normalizeLimit(limit))
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].CompletionRate = percent(rows[i].Completed, rows[i].Bookings)
		rows[i].Revenue = round2(rows[i].Revenue)
	}
	return &models.StaffPerformanceReport{ReportPeriod: period, Staff: rows}, nil
}

// --------------------
// CUSTOMERS REPORT (by scheduled date)
// --------------------
func (s *ReportService) Customers(ctx context.Context, role, from, to string, limit int) (*models.CustomersReport, error) {
	if err := requireAdmin(role); err != nil {
		return nil, err
	}
	start, end, period, err := dateRange(from, to)
	if err != nil {
		return nil, err
	}

	total, repeat, err := s.repo.CustomerSummary(ctx, start, end)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.CustomerPerformance(ctx, start, end, normalizeLimit(limit))
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].TotalSpent = round2(rows[i].TotalSpent)
	}

	return &models.CustomersReport{
		ReportPeriod:    period,
		TotalCustomers:  total,
		RepeatCustomers: repeat,
		Customers:       rows,
	}, nil
}
