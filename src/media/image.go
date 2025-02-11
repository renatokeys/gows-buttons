package media

import (
	"github.com/h2non/bimg"
)

// ImageThumbnail generates a thumbnail image from an image.
func ImageThumbnail(image []byte) ([]byte, error) {
	return Resize(image, 72, 72)
}

func ProfilePicture(image []byte) ([]byte, error) {
	return Resize(image, 640, 640)
}

// Resize generates a thumbnail image from an image.
func Resize(image []byte, width int, height int) ([]byte, error) {
	img := bimg.NewImage(image)
	options := bimg.Options{
		Width:  width,
		Height: height,
		Crop:   true,
		Type:   bimg.JPEG,
	}
	resized, err := img.Process(options)
	if err != nil {
		return nil, err
	}
	return resized, nil
}
