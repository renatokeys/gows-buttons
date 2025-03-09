package gows

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"time"
)

var (
	FetchPreviewTimeout = 6 * time.Second
)

func (gows *GoWS) BuildConversationMessage(text string) *waE2E.Message {
	message := waE2E.Message{}
	message.Conversation = proto.String(text)
	return &message
}

// BuildTextMessage builds a text message and adds a link preview if requested.
func (gows *GoWS) BuildTextMessage(text string) *waE2E.Message {
	message := waE2E.Message{}
	message.ExtendedTextMessage = &waE2E.ExtendedTextMessage{
		Text: proto.String(text),
	}
	return &message
}

// BuildEdit builds a message edit message using the given variables.
// The built message can be sent normally using Client.SendMessage.
//
// Adjusted from the original meow BuildEdit - it counts for participants (groups)
//
//	resp, err := cli.SendMessage(context.Background(), chat, cli.BuildEdit(chat, originalMessageID, &waE2E.Message{
//		Conversation: proto.String("edited message"),
//	})
func (gows *GoWS) BuildEdit(chat types.JID, id types.MessageID, newContent *waE2E.Message) *waE2E.Message {
	key := &waCommon.MessageKey{
		FromMe:    proto.Bool(true),
		ID:        proto.String(id),
		RemoteJID: proto.String(chat.String()),
	}
	// If the chat is a group, set the participant
	if chat.Server == types.GroupServer {
		key.Participant = proto.String(gows.int.GetOwnID().ToNonAD().String())
	}
	return &waE2E.Message{
		EditedMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{
				ProtocolMessage: &waE2E.ProtocolMessage{
					Key:           key,
					Type:          waE2E.ProtocolMessage_MESSAGE_EDIT.Enum(),
					EditedMessage: newContent,
					TimestampMS:   proto.Int64(time.Now().UnixMilli()),
				},
			},
		},
	}
}

func (gows *GoWS) PopulateContextInfoWithReply(info *waE2E.ContextInfo, replyToId types.MessageID) (*waE2E.ContextInfo, error) {
	msg, err := gows.Storage.Messages.GetMessage(replyToId)
	if err != nil {
		return info, err
	}

	if info == nil {
		info = &waE2E.ContextInfo{}
	}

	quoted := msg.Message.Message
	quoted.MessageContextInfo = nil
	info.StanzaID = proto.String(msg.Info.ID)
	info.Participant = proto.String(msg.Info.Sender.ToNonAD().String())
	info.QuotedMessage = quoted
	return info, nil
}

func (gows *GoWS) PopulateContextInfoDisappearingSettings(info *waE2E.ContextInfo, jid types.JID) (*waE2E.ContextInfo, error) {
	// Groups
	if jid.Server == types.GroupServer {
		group, err := gows.Storage.Groups.GetGroup(jid)
		if err != nil {
			return info, err
		}
		if group == nil {
			return info, fmt.Errorf("group not found: %s", jid)
		}
		if !group.IsEphemeral {
			return info, nil
		}

		if info == nil {
			info = &waE2E.ContextInfo{}
		}
		info.Expiration = proto.Uint32(group.DisappearingTimer)
		info.DisappearingMode = &waE2E.DisappearingMode{
			Initiator:     waE2E.DisappearingMode_CHANGED_IN_CHAT.Enum(),
			Trigger:       waE2E.DisappearingMode_CHAT_SETTING.Enum(),
			InitiatedByMe: proto.Bool(false),
		}
	}
	return info, nil
}
