package hostname

import (
	"strings"
	"testing"
)

func TestResolveFQDN_ReturnsNonEmpty(t *testing.T) {
	fqdn := ResolveFQDN()
	if fqdn == "" {
		t.Fatal("expected non-empty FQDN")
	}
}

func TestResolveFQDN_NoTrailingDot(t *testing.T) {
	fqdn := ResolveFQDN()
	if strings.HasSuffix(fqdn, ".") {
		t.Errorf("expected no trailing dot, got %q", fqdn)
	}
}

func TestResolveFQDN_NotUnknownOnHealthySystem(t *testing.T) {
	fqdn := ResolveFQDN()
	// "unknown" is only returned when os.Hostname() itself fails, which
	// should never happen on a healthy system.
	if fqdn == "unknown" {
		t.Errorf("expected a real hostname, got %q", fqdn)
	}
}
