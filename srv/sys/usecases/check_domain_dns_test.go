package usecases

import (
	"errors"
	"testing"
)

type fakeDNSResolver struct {
	ips map[string][]string
	err error
}

func (f *fakeDNSResolver) LookupHost(host string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.ips[host], nil
}

func TestCheckDomainDNSUseCase_Execute(t *testing.T) {
	t.Run("dominio que resuelve a la IP del servidor da match", func(t *testing.T) {
		resolver := &fakeDNSResolver{ips: map[string][]string{"example.com": {"1.2.3.4"}}}
		uc := NewCheckDomainDNSUseCase(resolver)

		result, err := uc.Execute("https://example.com/path", "1.2.3.4")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "match" || !result.Matches {
			t.Errorf("se esperaba match, got %+v", result)
		}
		if result.Domain != "example.com" {
			t.Errorf("se esperaba dominio normalizado 'example.com', got %q", result.Domain)
		}
	})

	t.Run("dominio que resuelve a otra IP da mismatch", func(t *testing.T) {
		resolver := &fakeDNSResolver{ips: map[string][]string{"example.com": {"9.9.9.9"}}}
		uc := NewCheckDomainDNSUseCase(resolver)

		result, err := uc.Execute("example.com", "1.2.3.4")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "mismatch" || result.Matches {
			t.Errorf("se esperaba mismatch, got %+v", result)
		}
	})

	t.Run("dominio que no resuelve da not_found sin error", func(t *testing.T) {
		resolver := &fakeDNSResolver{err: errors.New("no such host")}
		uc := NewCheckDomainDNSUseCase(resolver)

		result, err := uc.Execute("no-existe.com", "1.2.3.4")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "not_found" {
			t.Errorf("se esperaba not_found, got %+v", result)
		}
	})

	t.Run("formato de dominio inválido se rechaza sin llamar al resolver", func(t *testing.T) {
		resolver := &fakeDNSResolver{}
		uc := NewCheckDomainDNSUseCase(resolver)

		if _, err := uc.Execute("; rm -rf /", "1.2.3.4"); err == nil {
			t.Fatal("se esperaba error de formato inválido")
		}
	})

	t.Run("puerto y path se descartan del dominio", func(t *testing.T) {
		resolver := &fakeDNSResolver{ips: map[string][]string{"example.com": {"1.2.3.4"}}}
		uc := NewCheckDomainDNSUseCase(resolver)

		result, err := uc.Execute("http://example.com:8080/some/path?x=1", "1.2.3.4")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Domain != "example.com" {
			t.Errorf("se esperaba 'example.com' sin puerto ni path, got %q", result.Domain)
		}
	})
}
