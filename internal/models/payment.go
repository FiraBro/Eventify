package models

import "time"

type Payment struct {
	ID              string     `json:"id"`
	BookingID       string     `json:"booking_id"`
	UserID          string     `json:"user_id"`
	TxRef           string     `json:"tx_ref"`
	ChapaReference  string     `json:"chapa_reference,omitempty"`
	Amount          float64    `json:"amount"`
	Currency        string     `json:"currency"`
	Status          string     `json:"status"`
	CheckoutURL     string     `json:"checkout_url,omitempty"`
	RefundReference string     `json:"refund_reference,omitempty"`
	RefundReason    string     `json:"refund_reason,omitempty"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	RefundedAt      *time.Time `json:"refunded_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// POST /payments/checkout
type CheckoutRequest struct {
	BookingID   string `json:"booking_id"`
	Email       string `json:"email"`
	FirstName   string `json:"first_name,omitempty"`
	LastName    string `json:"last_name,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
}

// POST /payments/refund/:bookingId (body is optional)
type RefundRequest struct {
	Reason string `json:"reason,omitempty"`
}

// GET /payments/:bookingId/status
type PaymentStatusResponse struct {
	BookingID  string     `json:"booking_id"`
	Status     string     `json:"status"` // unpaid | pending | paid | failed | refund_pending | refunded
	Amount     float64    `json:"amount"`
	Currency   string     `json:"currency,omitempty"`
	PaidAt     *time.Time `json:"paid_at,omitempty"`
	RefundedAt *time.Time `json:"refunded_at,omitempty"`
}

// ---- gateway DTOs (provider-agnostic) ----

type GatewayInitRequest struct {
	Amount      float64
	Currency    string
	Email       string
	FirstName   string
	LastName    string
	PhoneNumber string
	TxRef       string
	CallbackURL string
	ReturnURL   string
}

type GatewayInitResponse struct {
	CheckoutURL string
}

type GatewayVerifyResponse struct {
	Status    string // success | failed | pending ...
	Reference string // provider's own reference (needed for refunds)
	Amount    float64
	Currency  string
}
