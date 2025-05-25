package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/devlikeapro/gows/gows"
	"strconv"
	"time"

	"github.com/devlikeapro/gows/media"
	__ "github.com/devlikeapro/gows/proto"
	"github.com/golang/protobuf/proto"
	"go.mau.fi/util/random"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

func (s *Server) GenerateNewMessageID(ctx context.Context, req *__.Session) (*__.NewMessageIDResponse, error) {
	cli, err := s.Sm.Get(req.GetId())
	if err != nil {
		return nil, err
	}
	id := cli.GenerateMessageID()
	return &__.NewMessageIDResponse{Id: id}, nil
}

func parseParticipantJIDs(participants []string) ([]types.JID, error) {
	jids := make([]types.JID, 0, len(participants))
	for _, p := range participants {
		jid, err := types.ParseJID(p)
		if err != nil {
			return nil, fmt.Errorf("invalid participant jid (%s): %w", p, err)
		}
		jids = append(jids, jid)
	}
	return jids, nil
}

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

	if len(req.Participants) > 0 {
		participants, err := parseParticipantJIDs(req.Participants)
		if err != nil {
			return nil, err
		}
		extra.Participants = participants
	}

	if req.Id != "" {
		extra.ID = req.Id
	}

	if req.GetPoll() != nil {
		poll := req.Poll
		selectableOptionCount := 0
		switch poll.MultipleAnswers {
		case false:
			selectableOptionCount = 1
		case true:
			selectableOptionCount = 0
		}
		message = cli.BuildPollCreationV3(poll.Name, poll.Options, selectableOptionCount)
		if message.PollCreationMessageV3 != nil {
			message.PollCreationMessageV3.ContextInfo = contextInfo
		}
		if message.PollCreationMessage != nil {
			message.PollCreationMessage.ContextInfo = contextInfo
		}
	} else if req.Event != nil {
		var location *gows.EventLocation
		if req.Event.Location != nil {
			location = &gows.EventLocation{
				Name:             req.Event.Location.Name,
				DegreesLongitude: req.Event.Location.DegreesLongitude,
				DegreesLatitude:  req.Event.Location.DegreesLatitude,
			}
		}
		event := &gows.EventMessage{
			Name:               req.Event.Name,
			Description:        req.Event.Description,
			StartTime:          req.Event.StartTime,
			EndTime:            req.Event.EndTime,
			ExtraGuestsAllowed: req.Event.ExtraGuestsAllowed,
			Location:           location,
		}
		message = gows.BuildEventCreation(event)
		message.EventMessage.ContextInfo = contextInfo
	} else if len(req.Contacts) > 0 {
		// Share contacts messages
		contacts := make([]gows.Contact, 0, len(req.Contacts))
		for _, contact := range req.Contacts {
			contacts = append(contacts, gows.Contact{
				DisplayName: contact.DisplayName,
				Vcard:       contact.Vcard,
			})
		}
		message = gows.BuildContactsMessage(contacts, contextInfo)
	} else if req.Media == nil {
		// Text Message
		message = cli.BuildTextMessage(req.Text)
		// Link Preview
		if req.LinkPreview {
			var preview *media.LinkPreview
			if req.Preview != nil {
				// Custom preview provided
				preview = &media.LinkPreview{
					Url:         req.Preview.Url,
					Title:       req.Preview.Title,
					Description: req.Preview.Description,
					Image:       req.Preview.Image,
				}
			}
			cli.AddLinkPreviewSafe(jid, message.ExtendedTextMessage, req.LinkPreviewHighQuality, preview)
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
	if err != nil {
		return nil, err
	}
	if jid.Server == types.NewsletterServer {
		serverID, err := strconv.Atoi(req.MessageId)
		if err != nil {
			cli.Log.Debugf("failed to convert message id (%s) when sending reaction to newsletter: %v", req.MessageId, err)
		}

		if serverID == 0 || err != nil {
			// It's not int - try to get it from storage
			storedMsg, err := cli.Storage.Messages.GetMessage(req.MessageId)
			if err != nil {
				cli.Log.Debugf("failed to get message (%s) when sending reaction to newsletter: %v", req.MessageId, err)
			}
			if storedMsg != nil {
				serverID = storedMsg.Message.Info.ServerID
			}
		}
		if serverID == 0 {
			return nil, fmt.Errorf("failed to get server id for newsletter message (%s)", req.MessageId)
		}

		messageID := cli.GenerateMessageID()
		err = cli.NewsletterSendReaction(jid, serverID, req.Reaction, messageID)
		if err != nil {
			return nil, err
		}
		msg := __.MessageResponse{
			Id:        messageID,
			Timestamp: time.Now().Unix(),
			Message:   nil,
		}
		return &msg, nil
	} else {
		sender, err := types.ParseJID(req.Sender)
		if err != nil {
			return nil, err
		}
		message := cli.BuildReaction(jid, sender, req.MessageId, req.Reaction)
		res, err := cli.SendMessage(ctx, jid, message, whatsmeow.SendRequestExtra{})
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

	ids := req.MessageIds
	if req.MessageId != "" {
		ids = append(ids, req.MessageId)
	}

	if len(ids) == 0 {
		return nil, errors.New("no message ids provided")
	}

	err = cli.MarkRead(ids, jid, sender, receiptType)
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

	extra := whatsmeow.SendRequestExtra{}
	if len(req.Participants) > 0 {
		participants, err := parseParticipantJIDs(req.Participants)
		if err != nil {
			return nil, err
		}
		extra.Participants = participants
	}

	message := cli.BuildRevoke(jid, participantJid, req.MessageId)
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
		cli.AddLinkPreviewSafe(jid, message.ExtendedTextMessage, req.LinkPreviewHighQuality, nil)
	}

	editMessage := cli.BuildEdit(jid, req.MessageId, message)
	res, err := cli.SendMessage(ctx, jid, editMessage, whatsmeow.SendRequestExtra{})
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

func (s *Server) SendButtonReply(ctx context.Context, req *__.ButtonReplyRequest) (*__.MessageResponse, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}

	contextInfo, err := cli.PopulateContextInfoWithReply(nil, req.ReplyTo)
	if err != nil {
		cli.Log.Warnf("Failed to get message for reply: %v", err)
	}

	message := &waE2E.Message{
		ButtonsResponseMessage: &waE2E.ButtonsResponseMessage{
			Type: waE2E.ButtonsResponseMessage_DISPLAY_TEXT.Enum(),
			Response: &waE2E.ButtonsResponseMessage_SelectedDisplayText{
				SelectedDisplayText: req.SelectedDisplayText,
			},
			SelectedButtonID: proto.String(req.SelectedButtonID),
			ContextInfo:      contextInfo,
		},
	}

	message.MessageContextInfo = &waE2E.MessageContextInfo{
		MessageSecret: random.Bytes(32),
	}

	res, err := cli.SendMessage(ctx, jid, message, whatsmeow.SendRequestExtra{})
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

func (s *Server) CancelEventMessage(ctx context.Context, req *__.CancelEventMessageRequest) (*__.MessageResponse, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.Jid)
	if err != nil {
		return nil, err
	}

	eventMessage, err := cli.Storage.Messages.GetMessage(req.MessageId)
	if err != nil {
		return nil, err
	}
	if eventMessage == nil {
		return nil, fmt.Errorf("event message not found: %s", req.MessageId)
	}
	update := eventMessage.RawMessage.EventMessage
	update.IsCanceled = proto.Bool(true)
	update.ContextInfo = nil
	message, err := cli.BuildEventUpdate(ctx, &eventMessage.Info, update)

	res, err := cli.SendMessage(ctx, jid, message, whatsmeow.SendRequestExtra{})
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
