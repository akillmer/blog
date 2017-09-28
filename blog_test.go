package blog

import (
	"os"
	"testing"
)

func TestBlogInit(t *testing.T) {
	if err := os.Remove("./test.db"); err != nil {
		t.Errorf("failed to remove previous `test.db`: %s", err)
	}

	options := &Options{
		DB:                "./test.db",
		RepoDir:           "./repo-test",
		StorageBucket:     "blog-media",
		ImagePreviewWidth: 80,
	}

	storageMaxAge = 0 // don't cache any stored assets for testing

	if err := Init(options); err != nil {
		t.Fatal(err)
	}
}
