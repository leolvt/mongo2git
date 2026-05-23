// Package hostname resolves the machine's fully qualified domain name.
package hostname

import (
	"net"
	"os"
	"strings"
)

// ResolveFQDN determines the fully qualified domain name by resolving the
// short hostname to an IP and performing a reverse lookup. Falls back to
// os.Hostname() on any failure, or "unknown" if even that fails.
func ResolveFQDN() string {
	short, err := os.Hostname()
	if err != nil {
		return "unknown"
	}

	addrs, err := net.LookupHost(short)
	if err != nil || len(addrs) == 0 {
		return short
	}

	names, err := net.LookupAddr(addrs[0])
	if err != nil || len(names) == 0 {
		return short
	}

	// Reverse lookup returns a trailing dot; strip it.
	fqdn := strings.TrimSuffix(names[0], ".")
	if fqdn == "" {
		return short
	}
	return fqdn
}
