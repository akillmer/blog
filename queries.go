package blog

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/boltdb/bolt"
)

// shareable transaction that returns bytes, not a struct
func txGetPageHeader(id []byte, tx *bolt.Tx) ([]byte, error) {
	var buf []byte
	pageBucket := tx.Bucket([]byte("pages"))
	p := pageBucket.Get([]byte(id))
	if p == nil {
		return nil, ErrPageNotFound
	}
	buf = make([]byte, len(p))
	copy(buf, p)
	return buf, nil
}

// GetPageHeader by it's ID (URL slug)
func GetPageHeader(id string) ([]byte, error) {
	var buf []byte

	if err := db.View(func(tx *bolt.Tx) error {
		var err error
		buf, err = txGetPageHeader([]byte(id), tx)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return buf, nil
}

// DeletePage and remove its assets from the CDN
func DeletePage(id string) error {
	buf, err := GetPageHeader(id)
	if err != nil {
		return err
	}

	p := &Page{}
	if err = json.Unmarshal(buf, p); err != nil {
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

// SearchPages returns a slice that matches keywords within the Title and Desc
func SearchPages(keywords ...string) ([][]byte, error) {
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

	pages := make([][]byte, len(matchedIDs))
	i := 0

	if err := db.View(func(tx *bolt.Tx) error {
		for k := range matchedIDs {
			buf, err := txGetPageHeader([]byte(k), tx)
			if err != nil {
				return err
			}
			pages[i] = buf
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
func GetPagesByTag(tag string) ([][]byte, error) {
	pages := [][]byte{}
	tag = strings.ToLower(tag)

	if err := db.View(func(tx *bolt.Tx) error {
		tagBucket := tx.Bucket([]byte("tags")).Bucket([]byte(tag))
		if tagBucket == nil {
			// not having a tag bucket isnt' really an error, just means there's no pages
			return nil
		}
		return tagBucket.ForEach(func(k, v []byte) error {
			buf, err := txGetPageHeader(k, tx)
			if err != nil {
				return err
			}
			pages = append(pages, buf)
			return nil
		})
	}); err != nil {
		return nil, err
	}

	return pages, nil
}
