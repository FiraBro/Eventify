package models

import "time"

// ReportPeriod echoes the (inclusive) date range a report was built for.
type ReportPeriod struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ---------- GET /reports/bookings ----------

type DailyCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type BookingsReport struct {
	ReportPeriod
	Total            int            `json:"total"`
	ByStatus         map[string]int `json:"by_status"`
	CancellationRate float64        `json:"cancellation_rate"` // percent
	Daily            []DailyCount   `json:"daily"`
}

// ---------- GET /reports/revenue ----------

type RevenueTotals struct {
	Gross    float64
	Payments int
	Refunded float64
	Refunds  int
}

type RevenuePoint struct {
	Period   string  `json:"period"` // first day of the day/week/month bucket
	Gross    float64 `json:"gross"`
	Refunded float64 `json:"refunded"`
	Net      float64 `json:"net"`
}

type RevenueReport struct {
	ReportPeriod
	Currency       string         `json:"currency"`
	GroupBy        string         `json:"group_by"`
	GrossRevenue   float64        `json:"gross_revenue"`
	Refunded       float64        `json:"refunded"`
	NetRevenue     float64        `json:"net_revenue"`
	PaymentsCount  int            `json:"payments_count"`
	RefundsCount   int            `json:"refunds_count"`
	AveragePayment float64        `json:"average_payment"`
	Series         []RevenuePoint `json:"series"`
}

// ---------- GET /reports/services ----------

type ServiceReportRow struct {
	ServiceID string  `json:"service_id"`
	Name      string  `json:"name"`
	Category  string  `json:"category"`
	Price     float64 `json:"price"`
	Bookings  int     `json:"bookings"`
	Completed int     `json:"completed"`
	Cancelled int     `json:"cancelled"`
	Revenue   float64 `json:"revenue"` // net of refunds
}

type ServicesReport struct {
	ReportPeriod
	Services []ServiceReportRow `json:"services"`
}

// ---------- GET /reports/staff-performance ----------

type StaffReportRow struct {
	StaffID        string  `json:"staff_id"`
	Name           string  `json:"name"`
	Email          string  `json:"email"`
	Bookings       int     `json:"bookings"`
	Completed      int     `json:"completed"`
	Cancelled      int     `json:"cancelled"`
	CompletionRate float64 `json:"completion_rate"` // percent of this staff's bookings
	Revenue        float64 `json:"revenue"`         // net of refunds
}

type StaffPerformanceReport struct {
	ReportPeriod
	Staff []StaffReportRow `json:"staff"`
}

// ---------- GET /reports/customers ----------

type CustomerReportRow struct {
	UserID        string    `json:"user_id"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	Bookings      int       `json:"bookings"`
	Completed     int       `json:"completed"`
	Cancelled     int       `json:"cancelled"`
	TotalSpent    float64   `json:"total_spent"` // net of refunds
	LastBookingAt time.Time `json:"last_booking_at"`
}

type CustomersReport struct {
	ReportPeriod
	TotalCustomers  int                 `json:"total_customers"`
	RepeatCustomers int                 `json:"repeat_customers"` // 2+ bookings in the period
	Customers       []CustomerReportRow `json:"customers"`
}
