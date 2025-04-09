package gows

import (
	"context"
	"fmt"
	"github.com/devlikeapro/gows/media"
	"github.com/gogo/protobuf/proto"
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

// AddLinkPreviewSafe adds a link preview to the message if a link is found in the text.
// logs an error if the preview cannot be fetched.
func (gows *GoWS) AddLinkPreviewSafe(jid types.JID, message *waE2E.ExtendedTextMessage, highQuality bool, preview *media.LinkPreview) {
	linkPreviewCtx, cancel := context.WithTimeout(gows.Context, FetchPreviewTimeout)
	defer cancel()
	err := gows.AddLinkPreviewWithContext(linkPreviewCtx, jid, message, highQuality, preview)
	if err != nil {
		gows.Log.Errorf("Failed to add link preview: %v", err)
	}
}

// AddLinkPreviewWithContext adds a link preview to the message if a link is found in the text.
// returns an error if the preview cannot be fetched.
func (gows *GoWS) AddLinkPreviewWithContext(
	ctx context.Context,
	jid types.JID,
	message *waE2E.ExtendedTextMessage,
	highQuality bool,
	preview *media.LinkPreview,
) (err error) {
	var matched string

	if preview == nil {
		// If the preview is nil, we need to extract the URL from the text
		text := message.Text
		matched = media.ExtractUrlFromText(*text)
		if matched == "" {
			return nil
		}
		// "matched" must be exact as it was in the text
		// but scraped URL should be normalized (because it'd also find www.whatsapp.com)
		url := media.MakeSureURL(matched)
		preview, err = media.GoscraperFetchPreview(ctx, url)
		if err != nil || preview == nil {
			return fmt.Errorf("failed to fetch preview info for (%s): %w", url, err)
		}
	} else {
		// If the preview provided, we need to extract the URL from it
		matched = preview.Url
	}

	type_ := waE2E.ExtendedTextMessage_NONE
	message.PreviewType = &type_
	message.MatchedText = &matched
	message.Title = &preview.Title
	message.Description = &preview.Description

	imageUrl := preview.ImageUrl
	if imageUrl == "" {
		// If the image URL is empty, we need to use the icon URL
		imageUrl = preview.IconUrl
		// HQ thumbnail is not supported for icon URL
		highQuality = false
		gows.Log.Infof("Using icon URL (%s) for link preview", imageUrl)
	}

	if imageUrl == "" {
		// valid case if no image URL is provided or found
		return nil
	}

	image, err := media.FetchBodyByUrl(ctx, imageUrl)
	if err != nil {
		return fmt.Errorf("failed to download image (%s) for link preview: %w", preview.ImageUrl, err)
	}

	if !highQuality {
		thumbnail, err := media.Resize(image, media.PreviewLinkBuiltInSize)
		if err != nil {
			return fmt.Errorf("failed to generate thumbnail: %w", err)
		}
		message.JPEGThumbnail = thumbnail
	} else {
		thumbnail, err := media.ImageAutoThumbnail(image)
		if err != nil {
			return fmt.Errorf("failed to generate thumbnail: %w", err)
		}
		resp, err := gows.UploadMedia(gows.Context, jid, image, whatsmeow.MediaLinkThumbnail)
		if err != nil {
			return fmt.Errorf("failed to upload image (%s): %w", preview.ImageUrl, err)
		}
		size, err := media.CurrentSize(image)
		if err != nil {
			size = media.PreviewLinkSize
		}
		message.JPEGThumbnail = thumbnail
		message.ThumbnailDirectPath = &resp.DirectPath
		message.ThumbnailSHA256 = resp.FileSHA256
		message.ThumbnailEncSHA256 = resp.FileEncSHA256
		message.ThumbnailHeight = proto.Uint32(size.Height)
		message.ThumbnailWidth = proto.Uint32(size.Width)
		message.MediaKey = resp.MediaKey
		now := time.Now().Unix()
		message.MediaKeyTimestamp = &now
	}
	return nil
}
