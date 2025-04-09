package media

import (
	"github.com/h2non/bimg"
)

type Size struct {
	Width  uint32
	Height uint32
}

var ThumbnailSize = Size{
	Width:  72,
	Height: 72,
}

var ProfilePictureSize = Size{
	Width:  640,
	Height: 640,
}

var PreviewLinkSize = Size{
	Width:  1024,
	Height: 512,
}

var PreviewLinkBuiltInSize = Size{
	Width:  192,
	Height: 192,
}

// ImageThumbnail generates a thumbnail image from an image.
func ImageThumbnail(image []byte) ([]byte, error) {
	return Resize(image, ThumbnailSize)
}

func ProfilePicture(image []byte) ([]byte, error) {
	return Resize(image, ProfilePictureSize)
}

// Resize generates a thumbnail image from an image.
func Resize(image []byte, size Size) ([]byte, error) {
	img := bimg.NewImage(image)
	options := bimg.Options{
		Width:  int(size.Width),
		Height: int(size.Height),
		Crop:   true,
		Type:   bimg.JPEG,
	}
	resized, err := img.Process(options)
	if err != nil {
		return nil, err
	}
	return resized, nil
}

func CurrentSize(buffer []byte) (Size, error) {
	image := bimg.NewImage(buffer)

	// Get the size
	size, err := image.Size()
	if err != nil {
		return Size{}, err
	}
	s := Size{
		Width:  uint32(size.Width),
		Height: uint32(size.Height),
	}
	return s, nil
}
