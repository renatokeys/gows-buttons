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

func randomId() string {
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
	buttonParams := map[string]interface{}{
		"display_text": button.Text,
		"id":           button.Id,
		"disabled":     false,
	}

	if button.Id == "" {
		buttonParams["id"] = randomId()
	}

	switch button.Type {
	case __.ButtonType_BUTTON_CALL:
		buttonParams["phone_number"] = button.PhoneNumber
	case __.ButtonType_BUTTON_COPY:
		buttonParams["copy_code"] = button.CopyCode
	case __.ButtonType_BUTTON_URL:
		buttonParams["url"] = button.Url
		buttonParams["merchant_url"] = button.Url
	}

	paramsJson, _ := json.Marshal(buttonParams)
	name := buttonTypeName(button.Type)
	paramsStr := string(paramsJson)

	return &waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{
		Name:             &name,
		ButtonParamsJSON: &paramsStr,
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
		"templateId": randomId(),
	})
	messageParamsStr := string(messageParamsJson)

	// Build native flow message
	nativeFlowMessage := &waE2E.InteractiveMessage_NativeFlowMessage{
		Buttons:           buttons,
		MessageParamsJSON: &messageParamsStr,
	}

	// Build interactive message
	interactiveMessage := &waE2E.InteractiveMessage{
		InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
			NativeFlowMessage: nativeFlowMessage,
		},
	}

	// Add header if present
	if req.Header != "" || len(req.HeaderImage) > 0 {
		hasMedia := len(req.HeaderImage) > 0
		interactiveMessage.Header = &waE2E.InteractiveMessage_Header{
			Title:              proto.String(req.Header),
			HasMediaAttachment: &hasMedia,
		}

		// Upload header image if present
		if len(req.HeaderImage) > 0 {
			mediaResponse, err := cli.UploadMedia(ctx, jid, req.HeaderImage, whatsmeow.MediaImage)
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
	if req.Body != "" {
		interactiveMessage.Body = &waE2E.InteractiveMessage_Body{
			Text: proto.String(req.Body),
		}
	}

	// Add footer if present
	if req.Footer != "" {
		interactiveMessage.Footer = &waE2E.InteractiveMessage_Footer{
			Text: proto.String(req.Footer),
		}
	}

	// Send InteractiveMessage directly
	message := &waE2E.Message{
		InteractiveMessage: interactiveMessage,
	}

	// gows-plus uses 4 args and different response structure
	res, err := cli.SendMessage(ctx, jid, message, whatsmeow.SendRequestExtra{})
	if err != nil {
		return nil, err
	}

	return &__.MessageResponse{Id: res.Info.ID, Timestamp: res.Info.Timestamp.Unix()}, nil
}
