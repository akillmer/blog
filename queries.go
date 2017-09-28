package blog

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/boltdb/bolt"
)

func txGetPage(id []byte, tx *bolt.Tx) (*Page, error) {
	page := &Page{}
	pageBucket := tx.Bucket([]byte("pages"))
	buf := pageBucket.Get([]byte(id))
	if buf == nil {
		return nil, ErrPageNotFound
	}
	if err := json.Unmarshal(buf, page); err != nil {
		return nil, err
	}
	return page, nil
}

// GetPage by it's ID (URL slug)
func GetPage(id string) (*Page, error) {
	var page *Page
	if err := db.View(func(tx *bolt.Tx) error {
		p, err := txGetPage([]byte(id), tx)
		if err != nil {
			return err
		}
		page = p
		return nil
	}); err != nil {
		return nil, err
	}
	return page, nil
}

// DeletePage and remove its assets from the CDN
func DeletePage(id string) error {
	p, err := GetPage(id)
	if err != nil {
		return err
	}

	if err = db.Update(func(tx *bolt.Tx) error {
		return p.txDelete(tx)
	}); err != nil {
		return err
	}

	for _, v := range p.Images {
		ctx := context.Background()
		if err := storageBucket.Object(v.ID).Delete(ctx); err != nil {
			return err
		}
		if err := storageBucket.Object("preview_" + v.ID).Delete(ctx); err != nil {
			return err
		}
	}

	return nil
}

// SearchPages returns a slice of Pages that match keywords within the Title and Desc
func SearchPages(keywords ...string) ([]*Page, error) {
	matchedIDs := make(map[string]struct{})

	if err := db.View(func(tx *bolt.Tx) error {
		searchBucket := tx.Bucket([]byte("search"))
		for _, query := range keywords {
			q := strings.ToLower(query)
			searchBucket.ForEach(func(k, v []byte) error {
				if bytes.Contains(k, []byte(q)) {
					if _, exists := matchedIDs[string(k)]; !exists {
						matchedIDs[string(v)] = struct{}{}
					}
				}
				return nil
			})
		}
		return nil
	}); err != nil {
		return nil, err
	}

	pages := make([]*Page, len(matchedIDs))
	i := 0

	if err := db.View(func(tx *bolt.Tx) error {
		for k := range matchedIDs {
			p, err := txGetPage([]byte(k), tx)
			if err != nil {
				return err
			}
			pages[i] = p
			i++
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return pages, nil
}

// AllTags returns all tags with the number of occurences for each
func AllTags() (map[string]int, error) {
	tags := make(map[string]int)

	if err := db.View(func(tx *bolt.Tx) error {
		tagBucket := tx.Bucket([]byte("tags"))
		return tagBucket.ForEach(func(k, v []byte) error {
			t := string(k)
			if _, exists := tags[t]; exists {
				tags[t]++
			} else {
				tags[t] = 1
			}
			return nil
		})
	}); err != nil {
		return nil, err
	}

	return tags, nil
}

// GetPagesByTag finds all Pages with a given tag
func GetPagesByTag(tag string) ([]*Page, error) {
	pages := []*Page{}
	tag = strings.ToLower(tag)

	if err := db.View(func(tx *bolt.Tx) error {
		tagBucket := tx.Bucket([]byte("tags")).Bucket([]byte(tag))
		if tagBucket == nil {
			// not having a tag bucket isnt' really an error, just means there's no pages
			return nil
		}
		return tagBucket.ForEach(func(k, v []byte) error {
			p, err := txGetPage(k, tx)
			if err != nil {
				return err
			}
			pages = append(pages, p)
			return nil
		})
	}); err != nil {
		return nil, err
	}

	return pages, nil
}
