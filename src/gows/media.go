package gows

import (
	"context"
	"fmt"
	"github.com/devlikeapro/gows/media"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"time"
)

func (gows *GoWS) UploadMedia(
	ctx context.Context,
	jid types.JID,
	content []byte,
	mediaType whatsmeow.MediaType,
) (resp whatsmeow.UploadResponse, err error) {
	if IsNewsletter(jid) {
		resp, err = gows.UploadNewsletter(ctx, content, mediaType)
	} else {
		resp, err = gows.Upload(ctx, content, mediaType)
	}
	return resp, err
}

func (gows *GoWS) AddLinkPreviewIfFound(jid types.JID, message *waE2E.ExtendedTextMessage, highQuality bool) error {
	text := message.Text
	matched := media.ExtractUrlFromText(*text)
	if matched == "" {
		return nil
	}
	// "matched" must be exact as it was in the text
	// but scraped URL should be normalized (because it'd also find www.whatsapp.com)
	url := media.MakeSureURL(matched)
	preview, err := media.GoscraperFetchPreview(gows.Context, url)
	if err != nil {
		return fmt.Errorf("failed to fetch preview info for (%s): %w", url, err)
	}

	var resp *whatsmeow.UploadResponse
	var thumbnail *[]byte
	if preview.ImageUrl != "" && highQuality {
		// Generate high quality thumbnail if asked
		resp, thumbnail, err = gows.fetchImageThumbnailHQ(jid, preview.ImageUrl)
		if err != nil {
			gows.Log.Warnf("failed get image high quality preview (%s): %v", preview.ImageUrl, err)
		}

	} else if preview.ImageUrl != "" && !highQuality {
		// Generate normal quality thumbnail
		thumbnail, err = gows.fetchImageThumbnail(preview.ImageUrl, media.PreviewLinkBuiltInSize)
		if err != nil {
			gows.Log.Warnf("failed get image preview (%s): %v", preview.ImageUrl, err)
		}
	}

	hasThumbnail := thumbnail != nil && len(*thumbnail) > 0
	if !hasThumbnail && preview.IconUrl != "" {
		// Generate thumbnail from icon if the main image is not available
		thumbnail, err = gows.fetchImageThumbnail(preview.IconUrl, media.ThumbnailSize)
		if err != nil {
			gows.Log.Warnf("failed get image preview for icon (%s): %v", preview.IconUrl, err)
		}
	}

	type_ := waE2E.ExtendedTextMessage_NONE
	message.PreviewType = &type_
	message.MatchedText = &matched
	message.Title = &preview.Title
	message.Description = &preview.Description

	if thumbnail != nil {
		message.JPEGThumbnail = *thumbnail
	}

	if resp != nil {
		message.ThumbnailDirectPath = &resp.DirectPath
		message.ThumbnailSHA256 = resp.FileSHA256
		message.ThumbnailEncSHA256 = resp.FileEncSHA256
		message.ThumbnailHeight = &media.PreviewLinkSize.Height
		message.ThumbnailWidth = &media.PreviewLinkSize.Width
		message.MediaKey = resp.MediaKey
		now := time.Now().Unix()
		message.MediaKeyTimestamp = &now
	}
	return nil
}

// fetchImageThumbnailHQ fetches the image from the URL, resizes it to the HQ size,
// uploads it to the server, and returns the thumbnail.
// aka High Quality thumbnail.
func (gows *GoWS) fetchImageThumbnailHQ(jid types.JID, imageUrl string) (*whatsmeow.UploadResponse, *[]byte, error) {
	imageOrig, err := gows.int.DownloadMedia(imageUrl)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download image: %w", err)
	}
	image, err := media.Resize(imageOrig, media.PreviewLinkSize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resize image: %w", err)
	}
	resp, err := gows.UploadMedia(gows.Context, jid, image, whatsmeow.MediaLinkThumbnail)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to upload image: %w", err)
	}
	thumbnail, err := media.ImageThumbnail(image)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate thumbnail: %w", err)
	}
	return &resp, &thumbnail, nil
}

// fetchImageThumbnail fetches the image from the URL, resizes it to the right size,
// and returns the thumbnail.
func (gows *GoWS) fetchImageThumbnail(imageUrl string, size media.Size) (*[]byte, error) {
	image, err := gows.int.DownloadMedia(imageUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	thumbnail, err := media.Resize(image, size)
	if err != nil {
		return nil, fmt.Errorf("failed to generate thumbnail: %w", err)
	}
	return &thumbnail, nil
}
