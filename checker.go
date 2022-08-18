package healthcheck

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alexliesenfeld/health"
)

type AndictlCheckerConfig struct {
	checkers []health.CheckerOption
}

func InitChecker() AndictlCheckerConfig {
	config := AndictlCheckerConfig{}
	config.checkers = make([]health.CheckerOption, 0, 10)
	// Set the time-to-live for our cache to 1 second (default).
	config.AddCheck(health.WithCacheDuration(1 * time.Second))
	// Configure a global timeout that will be applied to all checks.
	config.AddCheck(health.WithTimeout(10 * time.Second))
	// A check configuration to see if our database connection is up.
	// The check function will be executed for each HTTP request.

	config.AddCheck(health.WithCheck(health.Check{
		Name:    "goroutine-threshold", // A unique check name.
		Timeout: 2 * time.Second,       // A check specific timeout.
		Check:   GoroutineCountCheck(100),
	}))
	// Set a status listener that will be invoked when the health status changes.
	// More powerful hooks are also available (see docs).
	config.AddCheck(health.WithStatusListener(func(ctx context.Context, state health.CheckerState) {
		log.Println(fmt.Sprintf("health status changed to %s", state.Status))
	}))
	return config
}

func (c *AndictlCheckerConfig) AddCheck(check health.CheckerOption) {
	c.checkers = append(c.checkers, check)
}

func (c *AndictlCheckerConfig) AddDatabaseCheck(db *sql.DB) {
	check := health.WithCheck(health.Check{
		Name:    "database",      // A unique check name.
		Timeout: 2 * time.Second, // A check specific timeout.
		Check:   DatabasePingCheck(db, 1*time.Second),
	})
	fmt.Println("Check database health")
	c.AddCheck(check)
}

func (c AndictlCheckerConfig) GetCheckerHandler() http.HandlerFunc {
	return health.NewHandler(health.NewChecker(c.checkers...))
}
