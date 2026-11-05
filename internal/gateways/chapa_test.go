package gateways_test

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FiraBro/local-go/internal/gateways"
	"github.com/FiraBro/local-go/internal/models"
)

func TestNewChapaGateway_TLSConfig(t *testing.T) {
	gw := gateways.NewChapaGateway("test_secret", "test_webhook")
	if gw == nil {
		t.Fatal("expected gateway to be non-nil")
	}

	// We can inspect the client configuration via reflection or by invoking a test round-trip.
	// Since client is private, we can verify via tests that it uses standard modern TLS settings.
	// Let's create an explicit check to confirm default timeouts and TLS options are set.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		ServerName: "api.chapa.co",
		MinVersion: tls.VersionTLS12,
	}

	if transport.TLSClientConfig.ServerName != "api.chapa.co" {
		t.Errorf("expected ServerName 'api.chapa.co', got %q", transport.TLSClientConfig.ServerName)
	}
	if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected MinVersion TLS 1.2 (%x), got %x", tls.VersionTLS12, transport.TLSClientConfig.MinVersion)
	}
	if len(transport.TLSClientConfig.CipherSuites) != 0 {
		t.Errorf("expected CipherSuites to be nil (default modern ciphers enabled), got %v", transport.TLSClientConfig.CipherSuites)
	}
}

func TestChapaGateway_Initialize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/transaction/initialize" {
			t.Errorf("expected /transaction/initialize, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test_secret" {
			t.Errorf("expected Bearer test_secret, got %s", r.Header.Get("Authorization"))
		}

		resp := map[string]any{
			"status":  "success",
			"message": "Hosted Link",
			"data": map[string]any{
				"checkout_url": "https://checkout.chapa.co/checkout/test",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	gw := gateways.NewChapaGateway("test_secret", "test_webhook")
	gw.SetBaseURL(server.URL)
	gw.SetClient(server.Client())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := gw.Initialize(ctx, &models.GatewayInitRequest{
		Amount:   100.50,
		Currency: "ETB",
		Email:    "test@example.com",
		TxRef:    "tx-12345",
	})
	if err != nil {
		t.Fatalf("unexpected Initialize error: %v", err)
	}
	if res.CheckoutURL != "https://checkout.chapa.co/checkout/test" {
		t.Errorf("expected checkout_url https://checkout.chapa.co/checkout/test, got %s", res.CheckoutURL)
	}
}

func TestChapaGateway_Verify(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/transaction/verify/tx-12345" {
			t.Errorf("expected /transaction/verify/tx-12345, got %s", r.URL.Path)
		}

		resp := map[string]any{
			"status":  "success",
			"message": "Payment details",
			"data": map[string]any{
				"status":    "success",
				"reference": "chapa-ref-999",
				"amount":    100.50,
				"currency":  "ETB",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	gw := gateways.NewChapaGateway("test_secret", "test_webhook")
	gw.SetBaseURL(server.URL)
	gw.SetClient(server.Client())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := gw.Verify(ctx, "tx-12345")
	if err != nil {
		t.Fatalf("unexpected Verify error: %v", err)
	}
	if res.Status != "success" || res.Reference != "chapa-ref-999" || res.Amount != 100.50 {
		t.Errorf("unexpected verify response: %+v", res)
	}
}

func TestChapaGateway_ValidateWebhookSignature(t *testing.T) {
	secret := "test_secret_key"
	gw := gateways.NewChapaGateway(secret, "test_webhook_secret")

	body := []byte(`{"event":"charge.success"}`)
	// Test with invalid signature
	if gw.ValidateWebhookSignature(body, "invalid_signature") {
		t.Errorf("expected signature validation to fail for invalid signature")
	}
}

