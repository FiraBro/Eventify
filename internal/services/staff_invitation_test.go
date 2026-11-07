package services

import (
	"strings"
	"testing"

	"github.com/FiraBro/local-go/internal/config"
)

func TestGenerateStaffInvitationToken(t *testing.T) {
	first, firstHash, err := GenerateStaffInvitationToken()
	if err != nil {
		t.Fatalf("generate invitation token: %v", err)
	}
	second, _, err := GenerateStaffInvitationToken()
	if err != nil {
		t.Fatalf("generate second invitation token: %v", err)
	}
	if first == "" || first == second {
		t.Fatal("expected non-empty, unpredictable distinct tokens")
	}
	if len(first) < 40 || strings.ContainsAny(first, "+/=\r\n") {
		t.Fatalf("token is not a sufficiently long URL-safe value: %q", first)
	}
	if firstHash != HashToken(first) {
		t.Fatal("returned token digest does not match raw token")
	}
	if firstHash == first {
		t.Fatal("token digest unexpectedly equals raw token")
	}
}

func TestInvitationLinkRequiresHTTPSInProduction(t *testing.T) {
	previous := config.AppConfig
	t.Cleanup(func() { config.AppConfig = previous })
	config.AppConfig = &config.Config{AppEnv: "production"}
	if _, err := invitationLink("http://app.example.test/accept", "token"); err == nil {
		t.Fatal("production invitation link accepted plain HTTP")
	}
	if link, err := invitationLink("https://app.example.test/accept", "token"); err != nil || !strings.Contains(link, "token=token") {
		t.Fatalf("production HTTPS invitation link: link=%q err=%v", link, err)
	}
}
