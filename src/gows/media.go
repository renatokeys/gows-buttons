package gows

import (
	"context"
	"github.com/devlikeapro/gows/media"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
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

func (gows *GoWS) AddLinkPreviewIfFound(message *waE2E.ExtendedTextMessage) error {
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
		return err
	}

	type_ := waE2E.ExtendedTextMessage_NONE
	message.PreviewType = &type_
	message.MatchedText = &matched
	message.Title = &preview.Title
	message.Description = &preview.Description
	return nil
}
