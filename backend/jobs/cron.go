package jobs

import (
	"context"
	"log"
	"time"
)

// StartDocExpiryCron runs the SetDocExpiry job periodically at the specified interval
func StartDocExpiryCron(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 1 * time.Hour
	}

	log.Printf("[Doc Expiry Cron] Starting expiry cron worker (interval: %v)...", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial run
	SetDocExpiry(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[Doc Expiry Cron] Context canceled, stopping expiry cron worker...")
			return
		case <-ticker.C:
			SetDocExpiry(ctx)
		}
	}
}
