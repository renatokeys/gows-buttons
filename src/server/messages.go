package server

import (
	"context"
	"errors"
	"github.com/devlikeapro/gows/media"
	"github.com/devlikeapro/gows/proto"
	"github.com/golang/protobuf/proto"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"time"
)

func (s *Server) SendMessage(ctx context.Context, req *__.MessageRequest) (*__.MessageResponse, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}

	var contextInfo *waE2E.ContextInfo

	contextInfo, err = cli.PopulateContextInfoDisappearingSettings(contextInfo, jid)
	if err != nil {
		cli.Log.Warnf("Failed to get disappearing settings: %v", err)
	}

	if req.ReplyTo != "" {
		contextInfo, err = cli.PopulateContextInfoWithReply(contextInfo, req.ReplyTo)
		if err != nil {
			cli.Log.Warnf("Failed to get message for reply: %v", err)
		}
	}

	var message *waE2E.Message
	extra := whatsmeow.SendRequestExtra{}
	if req.Media == nil {
		// Text Message
		message = cli.BuildTextMessage(req.Text)
		// Link Preview
		if req.LinkPreview {
			cli.AddLinkPreviewSafe(jid, message.ExtendedTextMessage, req.LinkPreviewHighQuality)
		}

		message.ExtendedTextMessage.ContextInfo = contextInfo

		// Status Text Message
		var backgroundArgb *uint32
		if req.BackgroundColor != nil {
			backgroundArgb, err = media.ParseColor(req.BackgroundColor.Value)
			if err != nil {
				return nil, err
			}
			message.ExtendedTextMessage.BackgroundArgb = backgroundArgb
		}
		var font *waE2E.ExtendedTextMessage_FontType
		if req.Font != nil {
			font = media.ParseFont(req.Font.Value)
			message.ExtendedTextMessage.Font = font
		}
	} else {
		var mediaResponse whatsmeow.UploadResponse
		message = &waE2E.Message{}
		var mediaType whatsmeow.MediaType
		switch req.Media.Type {
		case __.MediaType_IMAGE:
			// Upload
			mediaType = whatsmeow.MediaImage
			mediaResponse, err = cli.UploadMedia(ctx, jid, req.Media.Content, mediaType)
			if err != nil {
				return nil, err
			}

			// Generate Thumbnail
			thumbnail, err := media.ImageThumbnail(req.Media.Content)
			if err != nil {
				cli.Log.Errorf("Failed to generate thumbnail: %v", err)
			}
			// Attach
			message.ImageMessage = &waE2E.ImageMessage{
				Caption:       proto.String(req.Text),
				Mimetype:      proto.String(req.Media.Mimetype),
				JPEGThumbnail: thumbnail,
				URL:           &mediaResponse.URL,
				DirectPath:    &mediaResponse.DirectPath,
				FileSHA256:    mediaResponse.FileSHA256,
				FileLength:    &mediaResponse.FileLength,
				MediaKey:      mediaResponse.MediaKey,
				FileEncSHA256: mediaResponse.FileEncSHA256,
			}
			message.ImageMessage.ContextInfo = contextInfo
		case __.MediaType_AUDIO:
			mediaType = whatsmeow.MediaAudio
			var waveform []byte
			var duration float32
			// Get waveform and duration if available
			if req.Media.Audio != nil {
				waveform = req.Media.Audio.Waveform
				duration = req.Media.Audio.Duration
			}

			if waveform == nil || len(waveform) == 0 {
				// Generate waveform
				waveform, err = media.Waveform(req.Media.Content)
				if err != nil {
					cli.Log.Errorf("Failed to generate waveform: %v", err)
				}
			}
			if duration == 0 {
				// Get duration
				duration, err = media.Duration(req.Media.Content)
				if err != nil {
					cli.Log.Errorf("Failed to get duration of audio: %v", err)
				}
			}
			durationSeconds := uint32(duration)

			// Upload
			mediaResponse, err = cli.UploadMedia(ctx, jid, req.Media.Content, mediaType)
			if err != nil {
				return nil, err
			}

			// Attach
			ptt := true
			message.AudioMessage = &waE2E.AudioMessage{
				Mimetype:      proto.String(req.Media.Mimetype),
				URL:           &mediaResponse.URL,
				DirectPath:    &mediaResponse.DirectPath,
				MediaKey:      mediaResponse.MediaKey,
				FileEncSHA256: mediaResponse.FileEncSHA256,
				FileSHA256:    mediaResponse.FileSHA256,
				FileLength:    &mediaResponse.FileLength,
				Seconds:       &durationSeconds,
				Waveform:      waveform,
				PTT:           &ptt,
			}
			message.AudioMessage.ContextInfo = contextInfo
		case __.MediaType_VIDEO:
			mediaType = whatsmeow.MediaVideo
			// Upload
			mediaResponse, err = cli.UploadMedia(ctx, jid, req.Media.Content, mediaType)
			if err != nil {
				return nil, err
			}

			// Generate Thumbnail
			thumbnail, err := media.VideoThumbnail(
				req.Media.Content,
				0,
				struct{ Width int }{Width: 72},
			)

			if err != nil {
				cli.Log.Infof("Failed to generate video thumbnail: %v", err)
			}

			message.VideoMessage = &waE2E.VideoMessage{
				Caption:       proto.String(req.Text),
				Mimetype:      proto.String(req.Media.Mimetype),
				URL:           &mediaResponse.URL,
				DirectPath:    &mediaResponse.DirectPath,
				MediaKey:      mediaResponse.MediaKey,
				FileEncSHA256: mediaResponse.FileEncSHA256,
				FileSHA256:    mediaResponse.FileSHA256,
				FileLength:    &mediaResponse.FileLength,
				JPEGThumbnail: thumbnail,
			}
			message.VideoMessage.ContextInfo = contextInfo

		case __.MediaType_DOCUMENT:
			mediaType = whatsmeow.MediaDocument
			// Upload
			mediaResponse, err = cli.UploadMedia(ctx, jid, req.Media.Content, mediaType)
			if err != nil {
				return nil, err
			}

			// Generate Thumbnail if possible
			thumbnail, err := media.ImageThumbnail(req.Media.Content)
			if err != nil {
				cli.Log.Infof("Failed to generate thumbnail: %v", err)
			}

			// Attach
			fileName := req.Media.Filename
			if fileName == "" {
				fileName = "Untitled"
			}
			documentMessage := &waE2E.DocumentMessage{
				Caption:       proto.String(req.Text),
				Title:         proto.String(fileName),
				Mimetype:      proto.String(req.Media.Mimetype),
				URL:           proto.String(mediaResponse.URL),
				DirectPath:    proto.String(mediaResponse.DirectPath),
				MediaKey:      mediaResponse.MediaKey,
				FileEncSHA256: mediaResponse.FileEncSHA256,
				FileSHA256:    mediaResponse.FileSHA256,
				FileLength:    proto.Uint64(mediaResponse.FileLength),
				FileName:      proto.String(fileName),
				JPEGThumbnail: thumbnail,
			}

			documentMessage.ContextInfo = contextInfo
			message.DocumentWithCaptionMessage = &waE2E.FutureProofMessage{
				Message: &waE2E.Message{
					DocumentMessage: documentMessage,
				},
			}
		}

		// Newsletters
		if mediaResponse.Handle != "" {
			extra.MediaHandle = mediaResponse.Handle
		}
	}

	res, err := cli.SendMessage(ctx, jid, message, extra)
	if err != nil {
		return nil, err
	}
	data, err := toJson(res)
	if err != nil {
		cli.Log.Errorf("Error marshaling message for response %v: %v", res.Info.ID, err)
	}
	msg := __.MessageResponse{
		Id:        res.Info.ID,
		Timestamp: res.Info.Timestamp.Unix(),
		Message:   data,
	}
	return &msg, nil
}

func (s *Server) SendReaction(ctx context.Context, req *__.MessageReaction) (*__.MessageResponse, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.Jid)
	sender, err := types.ParseJID(req.Sender)

	message := cli.BuildReaction(jid, sender, req.MessageId, req.Reaction)
	res, err := cli.SendMessage(ctx, jid, message)
	if err != nil {
		return nil, err
	}
	data, err := toJson(res)
	if err != nil {
		cli.Log.Errorf("Error marshaling message for response %v: %v", res.Info.ID, err)
	}
	msg := __.MessageResponse{
		Id:        res.Info.ID,
		Timestamp: res.Info.Timestamp.Unix(),
		Message:   data,
	}
	return &msg, nil
}

func (s *Server) MarkRead(ctx context.Context, req *__.MarkReadRequest) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.Jid)
	if err != nil {
		return nil, err
	}

	sender, err := types.ParseJID(req.Sender)
	if err != nil {
		return nil, err
	}

	var receiptType types.ReceiptType
	switch req.Type {
	case __.ReceiptType_READ:
		receiptType = types.ReceiptTypeRead
	case __.ReceiptType_PLAYED:
		receiptType = types.ReceiptTypePlayed
	default:
		return nil, errors.New("invalid receipt type: " + req.Type.String())
	}

	// id to ids array
	ids := []types.MessageID{req.MessageId}
	now := time.Now()
	err = cli.MarkRead(ids, now, jid, sender, receiptType)
	if err != nil {
		return nil, err
	}
	return &__.Empty{}, nil
}

func (s *Server) RevokeMessage(ctx context.Context, req *__.RevokeMessageRequest) (*__.MessageResponse, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.Jid)
	if err != nil {
		return nil, err
	}

	var participantJid types.JID
	if req.Sender != "" {
		participantJid, err = types.ParseJID(req.Sender)
		if err != nil {
			return nil, err
		}
	} else {
		participantJid = *cli.Store.ID
	}

	message := cli.BuildRevoke(jid, participantJid, req.MessageId)
	res, err := cli.SendMessage(ctx, jid, message)
	if err != nil {
		return nil, err
	}

	data, err := toJson(res)
	if err != nil {
		cli.Log.Errorf("Error marshaling message for response %v: %v", res.Info.ID, err)
	}
	msg := __.MessageResponse{
		Id:        res.Info.ID,
		Timestamp: res.Info.Timestamp.Unix(),
		Message:   data,
	}
	return &msg, nil
}

func (s *Server) EditMessage(ctx context.Context, req *__.EditMessageRequest) (*__.MessageResponse, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.Jid)
	if err != nil {
		return nil, err
	}

	message := cli.BuildConversationMessage(req.Text)
	if req.LinkPreview && media.ExtractUrlFromText(req.Text) != "" {
		// Switch to text message if it has URL and link preview is requested
		message = cli.BuildTextMessage(req.Text)
		cli.AddLinkPreviewSafe(jid, message.ExtendedTextMessage, req.LinkPreviewHighQuality)
	}

	editMessage := cli.BuildEdit(jid, req.MessageId, message)
	res, err := cli.SendMessage(ctx, jid, editMessage)
	if err != nil {
		return nil, err
	}

	data, err := toJson(res)
	if err != nil {
		cli.Log.Errorf("Error marshaling message for response %v: %v", res.Info.ID, err)
	}
	msg := __.MessageResponse{
		Id:        res.Info.ID,
		Timestamp: res.Info.Timestamp.Unix(),
		Message:   data,
	}
	return &msg, nil
}
