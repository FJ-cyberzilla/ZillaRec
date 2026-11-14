package network

import (
    "context"
    "crypto/tls"
    "fmt"
    "net"
    "net/http"
    "time"
)

// NewQuad9Transport creates an HTTP transport that exclusively uses Quad9 DNS.
// Designed for Android/Termux scenarios: Termux's resolv.conf is ignored,
// VPN DNS bypass is also avoided (unless explicitly required).
func NewQuad9Transport() *http.Transport {
    dialer := &net.Dialer{
        Timeout:   10 * time.Second,
        KeepAlive: 30 * time.Second,
    }

    // Use DoT (DNS over TLS) instead of UDP/53:
    // - Port 53 is often blocked or hijacked on mobile networks
    // - DoT is truly "secure" DNS and validates the resolver's identity
    resolver := &net.Resolver{
        PreferGo: true,
        Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
            var lastErr error
            // Quad9 primary + secondary DoT
            for _, server := range []string{"9.9.9.9:853", "149.112.112.112:853"} {
                d := &tls.Dialer{
                    NetDialer: dialer,
                    Config: &tls.Config{
                        ServerName: "dns.quad9.net",
                        MinVersion: tls.VersionTLS12,
                    },
                }
                cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
                conn, err := d.DialContext(cctx, "tcp", server)
                cancel()
                if err == nil {
                    return conn, nil
                }
                lastErr = err
            }
            return nil, fmt.Errorf("quad9 DoT unreachable: %w", lastErr)
        },
    }

    return &http.Transport{
        ForceAttemptHTTP2:     true,
        MaxIdleConns:          100,
        MaxIdleConnsPerHost:   10,
        IdleConnTimeout:       30 * time.Second,
        TLSHandshakeTimeout:   10 * time.Second,
        ResponseHeaderTimeout: 20 * time.Second,

        DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
            host, port, err := net.SplitHostPort(addr)
            if err != nil {
                return nil, err
            }

            // Dial IP literals directly without going through DNS
            if ip := net.ParseIP(host); ip != nil {
                return dialer.DialContext(ctx, network, addr)
            }

            ips, err := resolver.LookupIPAddr(ctx, host)
            if err != nil {
                return nil, fmt.Errorf("resolve %s via quad9: %w", host, err)
            }
            if len(ips) == 0 {
                return nil, fmt.Errorf("no addresses for %s via quad9", host)
            }

            // Try each IP with a short per-attempt timeout to avoid
            // unreachable IPv6 addresses causing long hangs on mobile networks
            var lastErr error
            for _, ip := range ips {
                cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
                conn, err := dialer.DialContext(cctx, network, net.JoinHostPort(ip.IP.String(), port))
                cancel()
                if err == nil {
                    return conn, nil
                }
                lastErr = err
            }
            return nil, fmt.Errorf("dial %s: %w", host, lastErr)
        },
    }
}
