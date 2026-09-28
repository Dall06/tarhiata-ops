package usecases

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

func generateTestCertBase64(t *testing.T, domainName string, notBefore, notAfter time.Time) string {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey error: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: domainName,
		},
		DNSNames:  []string{domainName},
		NotBefore: notBefore,
		NotAfter:  notAfter,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("x509.CreateCertificate error: %v", err)
	}

	return base64.StdEncoding.EncodeToString(certDER)
}

func TestInspectSSLAcmeUseCase_TableDriven(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

	activeCertB64 := generateTestCertBase64(t, "app.example.com", now.Add(-30*24*time.Hour), now.Add(60*24*time.Hour))
	expiringCertB64 := generateTestCertBase64(t, "api.example.com", now.Add(-80*24*time.Hour), now.Add(10*24*time.Hour))
	expiredCertB64 := generateTestCertBase64(t, "old.example.com", now.Add(-100*24*time.Hour), now.Add(-5*24*time.Hour))

	validStorage := map[string]interface{}{
		"letsencrypt": map[string]interface{}{
			"Certificates": []map[string]interface{}{
				{
					"domain": map[string]interface{}{
						"main": "app.example.com",
						"sans": []string{"www.app.example.com"},
					},
					"certificate": activeCertB64,
				},
				{
					"domain": map[string]interface{}{
						"main": "api.example.com",
					},
					"certificate": expiringCertB64,
				},
				{
					"domain": map[string]interface{}{
						"main": "old.example.com",
					},
					"certificate": expiredCertB64,
				},
			},
		},
	}
	validJSON, _ := json.Marshal(validStorage)

	tests := []struct {
		name       string
		rawJSON    string
		wantLen    int
		checkFirst func(t *testing.T, certs []ACMECertificateSummary)
	}{
		{
			name:    "Valid acme.json with active, expiring and expired certs",
			rawJSON: string(validJSON),
			wantLen: 3,
			checkFirst: func(t *testing.T, certs []ACMECertificateSummary) {
				if certs[0].Status != "active" {
					t.Errorf("expected cert 0 status active, got %s", certs[0].Status)
				}
				if certs[1].Status != "expiring_soon" {
					t.Errorf("expected cert 1 status expiring_soon, got %s", certs[1].Status)
				}
				if certs[2].Status != "expired" {
					t.Errorf("expected cert 2 status expired, got %s", certs[2].Status)
				}
			},
		},
		{
			name:    "Empty JSON storage",
			rawJSON: "{}",
			wantLen: 0,
			checkFirst: func(t *testing.T, certs []ACMECertificateSummary) {
				// No certs expected
			},
		},
		{
			name:    "Invalid JSON",
			rawJSON: "not_a_json",
			wantLen: 0,
			checkFirst: func(t *testing.T, certs []ACMECertificateSummary) {
				// No certs expected
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exec := &mockSecuritySSHExecutor{responses: map[string]string{
				"cat /opt/tarhiata": tc.rawJSON,
			}}
			uc := NewInspectSSLAcmeUseCase(exec)

			certs, err := uc.ParseACMEJSON(tc.rawJSON, now)
			if err != nil {
				t.Fatalf("ParseACMEJSON() error = %v", err)
			}
			if len(certs) != tc.wantLen {
				t.Fatalf("expected %d certs, got %d", tc.wantLen, len(certs))
			}
			if tc.checkFirst != nil {
				tc.checkFirst(t, certs)
			}
		})
	}
}

func TestInspectSSLAcmeUseCase_Execute(t *testing.T) {
	exec := &mockSecuritySSHExecutor{responses: map[string]string{
		"cat /opt/tarhiata": "{}",
	}}
	uc := NewInspectSSLAcmeUseCase(exec)
	certs, err := uc.Execute(domain.ServerConfig{Host: "10.0.0.1"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(certs) != 0 {
		t.Fatalf("expected 0 certs from empty storage, got %d", len(certs))
	}
}
