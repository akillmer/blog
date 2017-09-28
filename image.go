package blog

import (
	"bytes"
	"image"
	"image/jpeg"
	"os"

	"github.com/nfnt/resize"
)

// Image data associated with a blog entry
type Image struct {
	Width, Height int
	Preview       *bytes.Buffer
}

// NewImage analyzes the image for color, dimensions, and saves a preview image to a buffer
func NewImage(file string) (*Image, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	bimg := &Image{
		Preview: new(bytes.Buffer),
		Width:   img.Bounds().Dx(),
		Height:  img.Bounds().Dy(),
	}

	preview := resize.Resize(uint(opts.ImagePreviewWidth), 0, img, resize.NearestNeighbor)
	if err = jpeg.Encode(bimg.Preview, preview, nil); err != nil {
		return nil, err
	}

	return bimg, nil
}
