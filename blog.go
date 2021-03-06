package blog

import (
	"context"
	"time"

	"cloud.google.com/go/storage"
	"github.com/boltdb/bolt"
)

type Options struct {
	DB                string
	RepoDir           string
	StorageBucket     string
	ImagePreviewWidth int
}

var (
	db            *bolt.DB
	opts          *Options
	storageBucket *storage.BucketHandle
	cdnURL        string
	storageMaxAge = 86400 // gets set to 0 when testing (via TestBlogInit)
)

// Init the Blog package with passed Options
func Init(options *Options) error {
	opts = options
	cdnURL = "https://storage.googleapis.com/" + opts.StorageBucket

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	storageBucket = client.Bucket(opts.StorageBucket)

	db, err = bolt.Open(opts.DB, 0644, nil)
	if err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte("pages")); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte("html")); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte("tags")); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte("search")); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte("by-date")); err != nil {
			return err
		}
		return nil
	})
}

// DateFormat returns the time in RFC3339 format
func DateFormat(t time.Time) string {
	return t.Format(time.RFC3339)
}
