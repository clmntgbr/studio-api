package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/persistence/purge"
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
	var force bool
	cmd := &cobra.Command{
		Use:   "purge",
		Short: "Empty every table except " + strings.Join(purge.KeptTables, ", "),
		Long: "Truncate project/media/pipeline tables for local reset. " +
			"Keeps " + strings.Join(purge.KeptTables, ", ") + ". " +
			"Refused when GO_ENV is production. Pass --force to apply.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			env := config.Load()
			if config.IsProduction(env.Environment) {
				return errors.New("purge is refused when GO_ENV is production")
			}
			db := config.ConnectDatabase(env)
			if !force {
				tables, err := purge.Tables(db)
				if err != nil {
					return err
				}
				fmt.Printf("Would empty %d tables: %s\n", len(tables), strings.Join(tables, ", "))
				return errors.New("refusing to purge without --force")
			}
			tables, err := purge.Run(db)
			if err != nil {
				return err
			}
			fmt.Printf("Emptied %d tables: %s\n", len(tables), strings.Join(tables, ", "))
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "actually truncate the tables")

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

	var s3Force bool
	var s3DryRun bool
	var s3Prefix string
	var s3Buckets []string
	s3Cmd := &cobra.Command{
		Use:   "s3",
		Short: "Delete all objects in media/renders buckets (or --bucket)",
		Long: "Empty MinIO/S3 object content for local reset. " +
			"Defaults to S3_BUCKET_MEDIA + S3_BUCKET_RENDERS. " +
			"Refused when GO_ENV is production. Pass --force to apply.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			if config.IsProduction(cfg.Environment) {
				return errors.New("purge s3 is refused when GO_ENV is production")
			}
			store, err := s3store.New(s3store.Config{
				Endpoint:  cfg.S3Endpoint,
				Region:    cfg.S3Region,
				AccessKey: cfg.S3AccessKey,
				SecretKey: cfg.S3SecretKey,
			})
			if err != nil {
				return err
			}
			buckets := s3Buckets
			if len(buckets) == 0 {
				media := cfg.S3BucketMedia
				if media == "" {
					media = "media"
				}
				renders := cfg.S3BucketRenders
				if renders == "" {
					renders = "renders"
				}
				buckets = []string{media, renders}
			}
			ctx := context.Background()
			var total int
			apply := s3Force && !s3DryRun
			for _, bucket := range buckets {
				objects, err := store.ListPrefix(ctx, bucket, s3Prefix)
				if err != nil {
					return err
				}
				keys := make([]string, 0, len(objects))
				for _, obj := range objects {
					keys = append(keys, obj.Key)
				}
				if !apply {
					fmt.Printf("Would delete %d objects from s3://%s/%s\n", len(keys), bucket, s3Prefix)
					total += len(keys)
					continue
				}
				if err := store.DeleteObjects(ctx, bucket, keys); err != nil {
					return err
				}
				fmt.Printf("Deleted %d objects from s3://%s/%s\n", len(keys), bucket, s3Prefix)
				total += len(keys)
			}
			if !apply {
				if s3DryRun {
					fmt.Printf("purge s3 dry-run candidates=%d buckets=%s\n", total, strings.Join(buckets, ","))
					return nil
				}
				return errors.New("refusing to purge s3 without --force")
			}
			fmt.Printf("purge s3 done deleted=%d buckets=%s\n", total, strings.Join(buckets, ","))
			return nil
		},
	}
	s3Cmd.Flags().BoolVar(&s3Force, "force", false, "actually delete objects")
	s3Cmd.Flags().BoolVar(&s3DryRun, "dry-run", false, "list counts without deleting")
	s3Cmd.Flags().StringVar(&s3Prefix, "prefix", "", "optional key prefix to scope deletes")
	s3Cmd.Flags().StringSliceVar(&s3Buckets, "bucket", nil, "bucket(s) to empty (default: media + renders)")
	cmd.AddCommand(s3Cmd)
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
