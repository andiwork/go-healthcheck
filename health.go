package healthcheck

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"time"
)

// TCPDialCheck returns a Check that checks TCP connectivity to the provided
// endpoint.
func TCPDialCheck(addr string, timeout time.Duration) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err != nil {
			return err
		}
		return conn.Close()
	}
}

// HTTPGetCheck returns a Check that performs an HTTP GET request against the
// specified URL. The check fails if the response times out or returns a non-200
// status code.
func HTTPGetCheck(url string, timeout time.Duration) func(ctx context.Context) error {
	client := http.Client{
		Timeout: timeout,
		// never follow redirects
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return func(ctx context.Context) error {
		resp, err := client.Get(url)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("returned status %d", resp.StatusCode)
		}
		return nil
	}
}

// DatabasePingCheck returns a Check that validates connectivity to a
// database/sql.DB using Ping().
func DatabasePingCheck(database *sql.DB, timeout time.Duration) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if database == nil {
			return fmt.Errorf("database is nil")
		}
		return database.PingContext(ctx)
	}
}

// DNSResolveCheck returns a Check that makes sure the provided host can resolve
// to at least one IP address within the specified timeout.
func DNSResolveCheck(host string, timeout time.Duration) func(ctx context.Context) error {
	resolver := net.Resolver{}
	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		addrs, err := resolver.LookupHost(ctx, host)
		if err != nil {
			return err
		}
		if len(addrs) < 1 {
			return fmt.Errorf("could not resolve host")
		}
		return nil
	}
}

// GoroutineCountCheck returns a Check that fails if too many goroutines are
// running (which could indicate a resource leak).
func GoroutineCountCheck(threshold int) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		count := runtime.NumGoroutine()
		if count > threshold {
			return fmt.Errorf("too many goroutines (%d > %d)", count, threshold)
		}
		return nil
	}
}

// GCMaxPauseCheck returns a Check that fails if any recent Go garbage
// collection pause exceeds the provided threshold.
func GCMaxPauseCheck(threshold time.Duration) func(ctx context.Context) error {
	thresholdNanoseconds := uint64(threshold.Nanoseconds())
	return func(ctx context.Context) error {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)
		for _, pause := range stats.PauseNs {
			if pause > thresholdNanoseconds {
				return fmt.Errorf("recent GC cycle took %s > %s", time.Duration(pause), threshold)
			}
		}
		return nil
	}
}
