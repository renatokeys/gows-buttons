package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	__ "github.com/devlikeapro/gows/proto"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func randomButtonId() string {
	return fmt.Sprintf("%016d", rand.Int63())
}

func buttonTypeName(t __.ButtonType) string {
	switch t {
	case __.ButtonType_BUTTON_REPLY:
		return "quick_reply"
	case __.ButtonType_BUTTON_URL:
		return "cta_url"
	case __.ButtonType_BUTTON_CALL:
		return "cta_call"
	case __.ButtonType_BUTTON_COPY:
		return "cta_copy"
	default:
		return "quick_reply"
	}
}

func buttonToNativeFlowButton(button *__.Button) *waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton {
	buttonId := button.GetId()
	if buttonId == "" {
		buttonId = randomButtonId()
	}

	buttonParams := map[string]interface{}{
		"display_text": button.GetText(),
		"id":           buttonId,
		"disabled":     false,
	}

	switch button.Type {
	case __.ButtonType_BUTTON_CALL:
		buttonParams["phone_number"] = button.GetPhoneNumber()
	case __.ButtonType_BUTTON_COPY:
		buttonParams["copy_code"] = button.GetCopyCode()
	case __.ButtonType_BUTTON_URL:
		buttonParams["url"] = button.GetUrl()
		buttonParams["merchant_url"] = button.GetUrl()
	}

	paramsJson, _ := json.Marshal(buttonParams)
	name := buttonTypeName(button.Type)

	return &waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{
		Name:             proto.String(name),
		ButtonParamsJSON: proto.String(string(paramsJson)),
	}
}

func (s *Server) SendButtons(ctx context.Context, req *__.SendButtonsRequest) (*__.MessageResponse, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}

	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}

	// Build buttons
	buttons := make([]*waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton, len(req.Buttons))
	for i, btn := range req.Buttons {
		buttons[i] = buttonToNativeFlowButton(btn)
	}

	messageParamsJson, _ := json.Marshal(map[string]interface{}{
		"from":       "api",
		"templateId": randomButtonId(),
	})

	// Build native flow message
	nativeFlowMessage := &waE2E.InteractiveMessage_NativeFlowMessage{
		Buttons:           buttons,
		MessageParamsJSON: proto.String(string(messageParamsJson)),
	}

	// Build interactive message
	interactiveMessage := &waE2E.InteractiveMessage{
		InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
			NativeFlowMessage: nativeFlowMessage,
		},
	}

	// Add header if present
	if req.GetHeader() != "" || len(req.GetHeaderImage()) > 0 {
		hasMedia := len(req.GetHeaderImage()) > 0
		interactiveMessage.Header = &waE2E.InteractiveMessage_Header{
			Title:              proto.String(req.GetHeader()),
			HasMediaAttachment: proto.Bool(hasMedia),
		}

		// Upload header image if present
		if len(req.GetHeaderImage()) > 0 {
			mediaResponse, err := cli.UploadMedia(ctx, jid, req.GetHeaderImage(), whatsmeow.MediaImage)
			if err != nil {
				return nil, err
			}
			interactiveMessage.Header.Media = &waE2E.InteractiveMessage_Header_ImageMessage{
				ImageMessage: &waE2E.ImageMessage{
					URL:           &mediaResponse.URL,
					DirectPath:    &mediaResponse.DirectPath,
					MediaKey:      mediaResponse.MediaKey,
					FileEncSHA256: mediaResponse.FileEncSHA256,
					FileSHA256:    mediaResponse.FileSHA256,
					FileLength:    &mediaResponse.FileLength,
				},
			}
		}
	}

	// Add body if present
	if req.GetBody() != "" {
		interactiveMessage.Body = &waE2E.InteractiveMessage_Body{
			Text: proto.String(req.GetBody()),
		}
	}

	// Add footer if present
	if req.GetFooter() != "" {
		interactiveMessage.Footer = &waE2E.InteractiveMessage_Footer{
			Text: proto.String(req.GetFooter()),
		}
	}

	// Wrap in ViewOnceMessage (required for buttons to display correctly)
	deviceListMetadataVersion := int32(2)
	message := &waE2E.Message{
		ViewOnceMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{
				MessageContextInfo: &waE2E.MessageContextInfo{
					DeviceListMetadata:        &waE2E.DeviceListMetadata{},
					DeviceListMetadataVersion: &deviceListMetadataVersion,
				},
				InteractiveMessage: interactiveMessage,
			},
		},
	}

	res, err := cli.SendMessage(ctx, jid, message, whatsmeow.SendRequestExtra{})
	if err != nil {
		return nil, err
	}

	return &__.MessageResponse{
		Id:        res.Info.ID,
		Timestamp: res.Info.Timestamp.Unix(),
	}, nil
}
