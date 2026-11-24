package command

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go-api/internal/infrastructure/config"
	s3store "go-api/internal/infrastructure/storage/s3"

	"github.com/spf13/cobra"
)

// Derived object key fragments under the media bucket (sources are kept).
var derivedKeyParts = []string{
	"/proxy/",
	"/peaks/",
	"/audio/",
	"/analysis/",
	"/reframe/",
	"/transcript/",
}

func NewPurgeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "purge",
		Short: "Retention / cleanup commands",
	}

	var days int
	var dryRun bool
	var prefix string
	retention := &cobra.Command{
		Use:   "retention",
		Short: "Delete derived media objects older than N days (default 7)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if days < 1 {
				return fmt.Errorf("days must be >= 1")
			}
			cfg := config.Load()
			store, err := s3store.New(s3store.Config{
				Endpoint:  cfg.S3Endpoint,
				Region:    cfg.S3Region,
				AccessKey: cfg.S3AccessKey,
				SecretKey: cfg.S3SecretKey,
			})
			if err != nil {
				return err
			}
			bucket := cfg.S3BucketMedia
			if bucket == "" {
				bucket = envOr("S3_BUCKET_MEDIA", "media")
			}
			cutoff := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
			ctx := context.Background()
			objects, err := store.ListPrefix(ctx, bucket, prefix)
			if err != nil {
				return err
			}
			var deleted, skipped int
			for _, obj := range objects {
				if !isDerivedKey(obj.Key) {
					skipped++
					continue
				}
				if obj.LastModified.IsZero() || obj.LastModified.After(cutoff) {
					skipped++
					continue
				}
				if dryRun {
					fmt.Printf("dry-run delete s3://%s/%s (modified %s)\n", bucket, obj.Key, obj.LastModified.Format(time.RFC3339))
					deleted++
					continue
				}
				if err := store.DeleteObject(ctx, bucket, obj.Key); err != nil {
					return err
				}
				fmt.Printf("deleted s3://%s/%s\n", bucket, obj.Key)
				deleted++
			}
			fmt.Printf("retention done days=%d dryRun=%v deleted=%d skipped=%d\n", days, dryRun, deleted, skipped)
			return nil
		},
	}
	retention.Flags().IntVar(&days, "days", envInt("RETENTION_DAYS", 7), "Delete derived objects older than this many days")
	retention.Flags().BoolVar(&dryRun, "dry-run", false, "List candidates without deleting")
	retention.Flags().StringVar(&prefix, "prefix", "media/", "S3 key prefix to scan")
	cmd.AddCommand(retention)
	return cmd
}

func isDerivedKey(key string) bool {
	for _, part := range derivedKeyParts {
		if strings.Contains(key, part) {
			return true
		}
	}
	return false
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
