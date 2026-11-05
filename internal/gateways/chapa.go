package gateways

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync"
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

var (
	dohCacheMu sync.RWMutex
	dohCache   = make(map[string]dohEntry)
)

type dohEntry struct {
	ips       []string
	expiresAt time.Time
}

// resolveViaDoH queries public DNS over HTTPS (port 443) to bypass local network/ISP
// environments that drop or block standard port 53 UDP/TCP DNS packets.
func resolveViaDoH(ctx context.Context, hostname string) ([]string, error) {
	dohCacheMu.RLock()
	if entry, ok := dohCache[hostname]; ok && time.Now().Before(entry.expiresAt) {
		dohCacheMu.RUnlock()
		return entry.ips, nil
	}
	dohCacheMu.RUnlock()

	// Direct IP endpoints over HTTPS (no DNS needed to reach the resolver)
	dohEndpoints := []struct {
		url    string
		header string
	}{
		{
			url:    fmt.Sprintf("https://1.1.1.1/dns-query?name=%s&type=A", url.QueryEscape(hostname)),
			header: "application/dns-json",
		},
		{
			url:    fmt.Sprintf("https://8.8.8.8/resolve?name=%s&type=A", url.QueryEscape(hostname)),
			header: "application/json",
		},
		{
			url:    fmt.Sprintf("https://1.0.0.1/dns-query?name=%s&type=A", url.QueryEscape(hostname)),
			header: "application/dns-json",
		},
	}

	dohClient := &http.Client{
		Timeout: 4 * time.Second,
	}

	var lastErr error
	for _, ep := range dohEndpoints {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep.url, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Accept", ep.header)

		resp, err := dohClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		var result struct {
			Status int `json:"Status"`
			Answer []struct {
				Type int    `json:"type"`
				Data string `json:"data"`
				TTL  int    `json:"TTL"`
			} `json:"Answer"`
		}

		decodeErr := json.NewDecoder(resp.Body).Decode(&result)
		_ = resp.Body.Close()
		if decodeErr != nil {
			lastErr = decodeErr
			continue
		}

		var ips []string
		minTTL := 300
		for _, ans := range result.Answer {
			if ans.Type == 1 && ans.Data != "" { // Type 1 = A record (IPv4)
				ips = append(ips, ans.Data)
				if ans.TTL > 0 && ans.TTL < minTTL {
					minTTL = ans.TTL
				}
			}
		}

		if len(ips) > 0 {
			dohCacheMu.Lock()
			dohCache[hostname] = dohEntry{
				ips:       ips,
				expiresAt: time.Now().Add(time.Duration(minTTL) * time.Second),
			}
			dohCacheMu.Unlock()
			return ips, nil
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("DoH resolution for %s failed: %w", hostname, lastErr)
	}
	return nil, fmt.Errorf("DoH resolution for %s returned no A records", hostname)
}

func NewChapaGateway(secretKey, webhookSecret string) *ChapaGateway {
	baseURL := chapaBaseURL

	serverName := "api.chapa.co"
	if parsed, err := url.Parse(baseURL); err == nil && parsed.Hostname() != "" {
		serverName = parsed.Hostname()
	}

	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, splitErr := net.SplitHostPort(addr)
		if splitErr != nil {
			return dialer.DialContext(ctx, network, addr)
		}

		// If connecting to an IP (e.g. 127.0.0.1 in tests), dial directly without DNS
		if net.ParseIP(host) != nil {
			return dialer.DialContext(ctx, network, addr)
		}

		// 1. Resolve host via DNS-over-HTTPS (port 443) to avoid blocked UDP 53
		ips, dohErr := resolveViaDoH(ctx, host)
		if dohErr == nil && len(ips) > 0 {
			log.Printf("chapa: DoH resolved %s -> %v", host, ips)
			for _, ip := range ips {
				target := net.JoinHostPort(ip, port)
				conn, err := dialer.DialContext(ctx, network, target)
				if err == nil {
					log.Printf("chapa: connected to %s via DoH IP %s", host, ip)
					return conn, nil
				}
				log.Printf("chapa: connect to %s (%s) failed: %v", host, ip, err)
			}
		} else {
			log.Printf("chapa: DoH resolution failed (%v); falling back to system dial...", dohErr)
		}

		// 2. Fallback to standard dial
		return dialer.DialContext(ctx, network, addr)
	}

	// Clone http.DefaultTransport to inherit standard connection pooling and proxy settings.
	// Explicitly configure ServerName for SNI negotiation with Cloudflare, require TLS 1.2+,
	// and preserve default modern cipher suites.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = dialContext
	transport.TLSClientConfig = &tls.Config{
		ServerName: serverName,
		MinVersion: tls.VersionTLS12,
	}

	return &ChapaGateway{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		baseURL:       baseURL,
		client: &http.Client{
			Timeout:   20 * time.Second,
			Transport: transport,
		},
	}
}

// SetClient overrides the internal HTTP client (primarily for testing with mock servers).
func (g *ChapaGateway) SetClient(client *http.Client) {
	if client != nil {
		g.client = client
	}
}

// SetBaseURL overrides the gateway base URL (primarily for testing with mock servers).
func (g *ChapaGateway) SetBaseURL(baseURL string) {
	g.baseURL = baseURL
}

type chapaEnvelope struct {
	Status  string          `json:"status"`
	Message json.RawMessage `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (g *ChapaGateway) do(ctx context.Context, method, path, contentType string, body io.Reader) (*chapaEnvelope, error) {
	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			log.Printf("chapa: DNS lookup start for %s", info.Host)
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			if info.Err != nil {
				log.Printf("chapa: DNS lookup failed: %v", info.Err)
			} else {
				log.Printf("chapa: DNS resolved %v", info.Addrs)
			}
		},
		ConnectStart: func(network, addr string) {
			log.Printf("chapa: TCP connect start to %s", addr)
		},
		ConnectDone: func(network, addr string, err error) {
			if err != nil {
				log.Printf("chapa: TCP connect to %s failed: %v", addr, err)
			} else {
				log.Printf("chapa: TCP connected to %s", addr)
			}
		},
		TLSHandshakeStart: func() {
			log.Printf("chapa: TLS handshake starting...")
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			if err != nil {
				log.Printf("chapa: TLS handshake failed: %v", err)
			} else {
				log.Printf("chapa: TLS handshake completed successfully (TLS %x, cipher %x, SNI %s)", state.Version, state.CipherSuite, state.ServerName)
			}
		},
	}

	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), method, g.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+g.secretKey)
	keyType := "unrecognized"
	switch {
	case strings.HasPrefix(g.secretKey, "CHASECK"):
		keyType = "secret-key format"
	case strings.HasPrefix(g.secretKey, "CHAPUBK"):
		keyType = "public-key format"
	}
	log.Printf("DEBUG: Chapa API key format -> %s (length=%d); Authorization scheme='Bearer '", keyType, len(g.secretKey))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		// Client.Do may return a nil response on transport failures (including
		// DNS, TLS, and connection errors), so do not inspect resp here.
		log.Printf("chapa: %s %s request failed: %v", method, path, err)
		return nil, fmt.Errorf("chapa: %s %s request: %w", method, path, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var env chapaEnvelope
	_ = json.Unmarshal(raw, &env)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || (env.Status != "" && env.Status != "success") {
		if resp.StatusCode == http.StatusUnauthorized {
			log.Println("WARNING: Chapa rejected the API credentials; checkout was not created.")
		}
		msg := strings.Trim(string(env.Message), `"`)
		if msg == "" {
			msg = string(raw)
		}
		return nil, fmt.Errorf("chapa: %s (http %d; response: %s)", msg, resp.StatusCode, string(raw))
	}
	return &env, nil
}

// Initialize creates a hosted checkout session.
func (g *ChapaGateway) Initialize(ctx context.Context, r *models.GatewayInitRequest) (*models.GatewayInitResponse, error) {
	if r == nil || r.Amount <= 0 || math.IsNaN(r.Amount) || math.IsInf(r.Amount, 0) {
		return nil, errors.New("chapa: amount must be a finite positive number")
	}
	currency := "ETB"
	email := strings.TrimSpace(r.Email)
	if email == "" {
		email = "test-customer@example.com"
	}
	firstName := strings.TrimSpace(r.FirstName)
	if firstName == "" {
		firstName = "TestCustomer"
	}
	lastName := strings.TrimSpace(r.LastName)
	if lastName == "" {
		lastName = "User"
	}
	payload := map[string]any{
		"amount":       r.Amount,
		"currency":     currency,
		"email":        email,
		"first_name":   firstName,
		"last_name":    lastName,
		"tx_ref":       r.TxRef,
		"return_url":   "http://localhost:3100/customer/bookings",
		"callback_url": "http://127.0.0",
	}
	optional := map[string]string{
		"phone_number": r.PhoneNumber,
	}
	for k, v := range optional {
		if v != "" {
			payload[k] = v
		}
	}
	debugPayload, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("chapa: encode checkout payload: %w", err)
	}
	log.Printf("CHAPA OUTBOUND PAYLOAD: %s", debugPayload)
	b := debugPayload
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
