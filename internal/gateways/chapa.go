package gateways

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/FiraBro/local-go/internal/models"
)

const chapaBaseURL = "https://api.chapa.co/v1"

type ChapaGateway struct {
	secretKey     string
	webhookSecret string
	baseURL       string
	client        *http.Client
}

func NewChapaGateway(secretKey, webhookSecret string) *ChapaGateway {
	return &ChapaGateway{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		baseURL:       chapaBaseURL,
		client:        &http.Client{Timeout: 20 * time.Second},
	}
}

type chapaEnvelope struct {
	Status  string          `json:"status"`
	Message json.RawMessage `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (g *ChapaGateway) do(ctx context.Context, method, path, contentType string, body io.Reader) (*chapaEnvelope, error) {
	req, err := http.NewRequestWithContext(ctx, method, g.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+g.secretKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var env chapaEnvelope
	_ = json.Unmarshal(raw, &env)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || (env.Status != "" && env.Status != "success") {
		msg := strings.Trim(string(env.Message), `"`)
		if msg == "" {
			msg = string(raw)
		}
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return nil, fmt.Errorf("chapa: %s (http %d)", msg, resp.StatusCode)
	}
	return &env, nil
}

// Initialize creates a hosted checkout session.
func (g *ChapaGateway) Initialize(ctx context.Context, r *models.GatewayInitRequest) (*models.GatewayInitResponse, error) {
	payload := map[string]string{
		"amount":   strconv.FormatFloat(r.Amount, 'f', 2, 64),
		"currency": r.Currency,
		"email":    r.Email,
		"tx_ref":   r.TxRef,
	}
	optional := map[string]string{
		"first_name":   r.FirstName,
		"last_name":    r.LastName,
		"phone_number": r.PhoneNumber,
		"callback_url": r.CallbackURL,
		"return_url":   r.ReturnURL,
	}
	for k, v := range optional {
		if v != "" {
			payload[k] = v
		}
	}

	b, _ := json.Marshal(payload)
	env, err := g.do(ctx, http.MethodPost, "/transaction/initialize", "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	var data struct {
		CheckoutURL string `json:"checkout_url"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil || data.CheckoutURL == "" {
		return nil, errors.New("chapa: checkout_url missing in response")
	}
	return &models.GatewayInitResponse{CheckoutURL: data.CheckoutURL}, nil
}

// Verify asks Chapa for the real state of a transaction.
func (g *ChapaGateway) Verify(ctx context.Context, txRef string) (*models.GatewayVerifyResponse, error) {
	env, err := g.do(ctx, http.MethodGet, "/transaction/verify/"+url.PathEscape(txRef), "", nil)
	if err != nil {
		return nil, err
	}

	var data struct {
		Status    string      `json:"status"`
		Reference string      `json:"reference"`
		Amount    json.Number `json:"amount"` // accepts 100 or "100"
		Currency  string      `json:"currency"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, fmt.Errorf("chapa: unexpected verify response: %w", err)
	}
	amount, _ := data.Amount.Float64()

	return &models.GatewayVerifyResponse{
		Status:    data.Status,
		Reference: data.Reference,
		Amount:    amount,
		Currency:  data.Currency,
	}, nil
}

// Refund refunds the full transaction. Chapa expects ITS reference, not our tx_ref.
func (g *ChapaGateway) Refund(ctx context.Context, chapaRef, reason, refundRef string) error {
	form := url.Values{}
	if reason != "" {
		form.Set("reason", reason)
	}
	form.Set("reference", refundRef) // unique id for this refund request

	_, err := g.do(ctx, http.MethodPost, "/refund/"+url.PathEscape(chapaRef),
		"application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	return err
}

// ValidateWebhookSignature checks the Chapa-Signature / x-chapa-signature headers.
// Per Chapa docs the signature is an HMAC-SHA256 hex digest; both variants are accepted here.
// IMPORTANT: the service ALSO re-verifies every transaction through the API, so a
// forged webhook can never mark a payment as paid.
func (g *ChapaGateway) ValidateWebhookSignature(body []byte, signatures ...string) bool {
	if g.secretKey == "" {
		return false
	}
	keys := []string{g.secretKey}
	if g.webhookSecret != "" {
		keys = append(keys, g.webhookSecret)
	}

	var expected []string
	for _, k := range keys {
		expected = append(expected, sign(k, []byte(k)), sign(k, body))
	}

	for _, sig := range signatures {
		sig = strings.ToLower(strings.TrimSpace(sig))
		if sig == "" {
			continue
		}
		for _, e := range expected {
			if hmac.Equal([]byte(sig), []byte(e)) {
				return true
			}
		}
	}
	return false
}

func sign(key string, msg []byte) string {
	m := hmac.New(sha256.New, []byte(key))
	m.Write(msg)
	return hex.EncodeToString(m.Sum(nil))
}