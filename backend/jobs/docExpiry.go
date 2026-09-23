package jobs

import (
	"context"
	"log"

	"doc-manager/resources"
)

// SetDocExpiry updates the status of all documents older than 5 minutes from 'uploaded' to 'expired'
func SetDocExpiry(ctxOpt ...context.Context) {
	ctx := context.Background()
	if len(ctxOpt) > 0 && ctxOpt[0] != nil {
		ctx = ctxOpt[0]
	}

	if resources.DB == nil {
		log.Println("[Doc Expiry Job] Database connection not initialized")
		return
	}

	query := `
		UPDATE documents
		SET status = 'expired',
		    updated_at = NOW()
		WHERE status = 'uploaded'
		  AND created_at < NOW() - INTERVAL '5 minutes'
	`

	tag, err := resources.DB.Exec(ctx, query)
	if err != nil {
		log.Printf("[Doc Expiry Job] Failed to expire documents: %v", err)
		return
	}

	rowsAffected := tag.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("[Doc Expiry Job] Successfully updated %d document(s) to 'expired'", rowsAffected)
	}
}
