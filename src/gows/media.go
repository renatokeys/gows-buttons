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

func (gows *GoWS) AddLinkPreviewIfFound(jid types.JID, message *waE2E.ExtendedTextMessage) error {
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
		return fmt.Errorf("failed to fetch preview: %w", err)
	}

	var resp *whatsmeow.UploadResponse
	var thumbnail *[]byte
	if preview.ImageUrl != "" {
		resp, thumbnail, err = gows.fetchAndUploadMedia(jid, preview.ImageUrl)
		if err != nil {
			return fmt.Errorf("failed get image preview: %w", err)
		}
	}

	if resp == nil && preview.IconUrl != "" {
		resp, thumbnail, err = gows.fetchAndUploadMedia(jid, preview.ImageUrl)
		if err != nil {
			return fmt.Errorf("failed get image preview from icon: %w", err)
		}
	}

	type_ := waE2E.ExtendedTextMessage_NONE
	message.PreviewType = &type_
	message.MatchedText = &matched
	message.Title = &preview.Title
	message.Description = &preview.Description
	if resp != nil {
		message.JPEGThumbnail = *thumbnail
		message.ThumbnailDirectPath = &resp.DirectPath
		message.ThumbnailSHA256 = resp.FileSHA256
		message.ThumbnailEncSHA256 = resp.FileEncSHA256
		message.MediaKey = resp.MediaKey
		now := time.Now().Unix()
		message.MediaKeyTimestamp = &now
	}
	return nil
}

func (gows *GoWS) fetchAndUploadMedia(jid types.JID, imageUrl string) (*whatsmeow.UploadResponse, *[]byte, error) {
	image, err := gows.int.DownloadMedia(imageUrl)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download image: %w", err)
	}
	resp, err := gows.Upload(gows.Context, image, whatsmeow.MediaImage)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to upload image: %w", err)
	}
	thumbnail, err := media.ImageThumbnail(image)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate thumbnail: %w", err)
	}
	return &resp, &thumbnail, nil
}
