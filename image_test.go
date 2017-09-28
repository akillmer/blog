package blog

import (
	"image/jpeg"
	"testing"
)

func TestImage(t *testing.T) {
	img, err := NewImage("./blog-test/sample_a.jpg")
	if err != nil {
		t.Fatal(err)
	}

	if img.Width != 600 {
		t.Errorf("image width should be 600, got %d", img.Width)
	}

	if img.Height != 408 {
		t.Errorf("image height should be 408, got %d", img.Height)
	}

	preview, err := jpeg.Decode(img.Preview)
	if err != nil {
		t.Fatal(err)
	}

	if preview.Bounds().Dx() != opts.ImagePreviewWidth {
		t.Errorf("preview image width should be %d, got %d", opts.ImagePreviewWidth, preview.Bounds().Dx())
	}
}
