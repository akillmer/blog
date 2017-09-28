package blog

import (
	"bufio"
	"bytes"
	"errors"
	"image/jpeg"
	"os"

	pcolor "github.com/EdlinOrg/prominentcolor"
	"github.com/nfnt/resize"
	shortid "github.com/ventu-io/go-shortid"
)

// Image data associated with a blog entry
type Image struct {
	ID     string `json:"id"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Color  string `json:"color"`
	buffer bytes.Buffer
}

// NewImage analyzes the image for color and dimensions
func NewImage(file string) (*Image, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil {
		return nil, err
	}

	bimg := &Image{
		Width:  img.Bounds().Dx(),
		Height: img.Bounds().Dy(),
	}

	c := pcolor.KmeansWithArgs(pcolor.ArgumentNoCropping, img)
	if len(c) == 0 {
		return nil, errors.New("image has no color data")
	}
	bimg.Color = c[0].AsString()

	preview := resize.Resize(uint(opts.ImagePreviewWidth), 0, img, resize.NearestNeighbor)
	w := bufio.NewWriter(&bimg.buffer)
	if err = jpeg.Encode(w, preview, nil); err != nil {
		return nil, err
	}

	sid, err := shortid.Generate()
	if err != nil {
		return nil, err
	}

	bimg.ID = sid + ".jpg"

	return bimg, nil
}

// URL of the Image via Google Cloud Storage
func (i *Image) URL() string {
	return cdnURL + "/" + i.ID
}
