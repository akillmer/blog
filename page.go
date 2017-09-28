package blog

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/boltdb/bolt"
)

// Page contains content and metadata
type Page struct {
	ID        string            `json:"id"`
	Published string            `json:"published"`
	Modified  string            `json:"modified"`
	Title     string            `json:"title"`
	Desc      string            `json:"desc"`
	Tags      []string          `json:"tags"`
	Images    map[string]*Image `json:"images"`
	buffer    []byte
}

var (
	rePageTitle         = regexp.MustCompile(`(?m)^# (.+)`)
	rePageDesc          = regexp.MustCompile(`(?m)^## (.+)`)
	rePageImages        = regexp.MustCompile(`!\[.*]\((.+)\)`)
	rePageTags          = regexp.MustCompile(`\[tags]: (.+)`)
	ErrPageMissingTitle = errors.New("markdown missing top level header (title)")
	ErrPageMissingDesc  = errors.New("markdown missing second level header (description)")
	ErrPageMissingTags  = errors.New("markdown missing tags")
	ErrPageNotFound     = errors.New("post not found")
)

// NewPage parses a page's folder for its content and metadata
func NewPage(pathToDir string) (*Page, error) {
	allFiles, err := ioutil.ReadDir(pathToDir)
	if err != nil {
		return nil, err
	}

	var mdFile string
	for _, f := range allFiles {
		if path.Ext(f.Name()) == ".md" {
			mdFile = path.Join(pathToDir, f.Name())
		}
	}

	buf, err := ioutil.ReadFile(mdFile)
	if err != nil {
		return nil, err
	}
	page := &Page{ID: path.Base(pathToDir)}

	title := rePageTitle.FindSubmatch(buf)
	if title == nil {
		return nil, ErrPageMissingTitle
	}
	page.Title = string(title[1])
	// remove the blog's title from the markdown
	buf = bytes.Replace(buf, append(title[0], []byte("\n\n")...), nil, 1)

	desc := rePageDesc.FindSubmatch(buf)
	if desc == nil {
		return nil, ErrPageMissingDesc
	}
	page.Desc = string(desc[1])
	// remove the blog's description from the markdown
	buf = bytes.Replace(buf, append(desc[0], []byte("\n\n")...), nil, 1)

	images := rePageImages.FindAllSubmatch(buf, -1)
	if images != nil {
		page.Images = make(map[string]*Image, len(images))

		for i := range images {
			origName := string(images[i][1])
			// skip any "hot linked" images
			if strings.HasPrefix(origName, "http") {
				continue
			}

			if _, exists := page.Images[origName]; exists == false {
				img, err := NewImage(path.Join(pathToDir, origName))

				if err != nil {
					return nil, err
				}
				page.Images[origName] = img
				buf = bytes.Replace(buf, []byte(origName), []byte(img.URL()), -1)
			}
		}
	}

	tags := rePageTags.FindAllSubmatch(buf, -1)
	if tags == nil {
		return nil, ErrPageMissingTags
	}

	for i := range tags {
		t := bytes.Split(tags[i][1], []byte(","))
		for _, tag := range t {
			page.Tags = append(page.Tags, strings.ToLower(string(tag)))
		}
	}

	page.buffer = buf

	return page, nil
}

func (p *Page) txPut(tx *bolt.Tx) error {
	buf, err := json.Marshal(p)
	if err != nil {
		return err
	}

	pageBucket := tx.Bucket([]byte("pages"))
	if err := pageBucket.Put([]byte(p.ID), buf); err != nil {
		return err
	}

	htmlBucket := tx.Bucket([]byte("html"))
	if err := htmlBucket.Put([]byte(p.ID), p.buffer); err != nil {
		return err
	}

	tagBucket := tx.Bucket([]byte("tags"))
	for _, tag := range p.Tags {
		b, err := tagBucket.CreateBucketIfNotExists([]byte(tag))
		if err != nil {
			return err
		}
		if err := b.Put([]byte(p.ID), nil); err != nil {
			return err
		}
	}

	searchBucket := tx.Bucket([]byte("search"))
	if err := searchBucket.Put([]byte(strings.ToLower(p.Title)), []byte(p.ID)); err != nil {
		return err
	}
	if err := searchBucket.Put([]byte(strings.ToLower(p.Desc)), []byte(p.ID)); err != nil {
		return err
	}

	dateBucket := tx.Bucket([]byte("by-date"))
	if err := dateBucket.Put([]byte(p.Published), []byte(p.ID)); err != nil {
		return err
	}

	return nil
}

func (p *Page) txDelete(tx *bolt.Tx) error {
	pageBucket := tx.Bucket([]byte("pages"))
	// deleting an object from a bucket returns no error (unless tx is read-only)
	// so the key needs to be checked explicitly to confirm
	if k := pageBucket.Get([]byte(p.ID)); k == nil {
		return ErrPageNotFound
	}

	if err := pageBucket.Delete([]byte(p.ID)); err != nil {
		return err
	}

	htmlBucket := tx.Bucket([]byte("html"))
	if err := htmlBucket.Delete([]byte(p.ID)); err != nil {
		return err
	}

	tagBucket := tx.Bucket([]byte("tags"))
	for _, tag := range p.Tags {
		b := tagBucket.Bucket([]byte(tag))
		if err := b.Delete([]byte(p.ID)); err != nil {
			return err
		}
		// determine if the bucket is empty, if so delete
		prev, _ := b.Cursor().Prev()
		next, _ := b.Cursor().Next()
		if prev == nil && next == nil {
			if err := tagBucket.DeleteBucket([]byte(tag)); err != nil {
				return err
			}
		}
	}

	searchBucket := tx.Bucket([]byte("search"))
	if err := searchBucket.Delete([]byte(strings.ToLower(p.Title))); err != nil {
		return err
	}
	if err := searchBucket.Delete([]byte(strings.ToLower(p.Desc))); err != nil {
		return err
	}

	dateBucket := tx.Bucket([]byte("by-date"))
	if err := dateBucket.Delete([]byte(p.Published)); err != nil {
		return err
	}

	return nil
}

func (p *Page) requestHTML() error {
	reader := bytes.NewReader(p.buffer)
	req, err := http.NewRequest("POST", "https://api.github.com/markdown/raw", reader)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "text/x-markdown")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("%d: %s", resp.StatusCode, resp.Status)
	}

	p.buffer, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// Save a Page and push its assets to the CDN
func (p *Page) Save() error {
	err := p.requestHTML()
	if err != nil {
		return err
	}

	if err = db.Update(func(tx *bolt.Tx) error {
		timestamp := time.Now().Format(time.RFC3339)
		if err := p.txDelete(tx); err == nil {
			// this is a modified page
			p.Modified = timestamp
		} else if err == ErrPageNotFound {
			// this is a new page
			p.Published = timestamp
		} else {
			// something bad happened
			return err
		}
		return p.txPut(tx)
	}); err != nil {
		return err
	}

	for k, v := range p.Images {
		f, err := os.Open(path.Join(opts.BlogDir, p.ID, k))
		if err != nil {
			return err
		}

		// upload the high res image
		ctx := context.Background()
		w := storageBucket.Object(v.ID).NewWriter(ctx)
		w.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}
		w.ContentType = "image/jpeg"
		w.CacheControl = fmt.Sprintf("public, max-age=%d", storageMaxAge)

		if _, err = io.Copy(w, f); err != nil {
			return err
		}

		f.Close()
		w.Close()

		// upload the preview image
		r := bufio.NewReader(&v.buffer)
		w = storageBucket.Object("preview_" + v.ID).NewWriter(ctx)
		w.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}
		w.ContentType = "image/jpeg"
		w.CacheControl = fmt.Sprintf("public, max-age=%d", storageMaxAge)
		if _, err = io.Copy(w, r); err != nil {
			return err
		}

		w.Close()
	}

	return nil
}
