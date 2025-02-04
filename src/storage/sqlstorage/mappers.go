package sqlstorage

import (
	"encoding/json"
	"github.com/devlikeapro/gows/storage"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/encoding/protojson"
)

type JsonMapper struct {
}

func (f *JsonMapper) Marshal(entity *storage.StoredMessage) ([]byte, error) {
	return json.Marshal(entity)
}

func (f *JsonMapper) Unmarshal(data []byte, entity *storage.StoredMessage) error {
	return json.Unmarshal(data, entity)
}

func isNullJson(data []byte) bool {
	if len(data) != 4 {
		return false
	}
	// null
	return data[0] == 'n' && data[1] == 'u' && data[2] == 'l' && data[3] == 'l'
}

type MessageMapper struct {
}

func (f *MessageMapper) ToFields(entity *storage.StoredMessage) map[string]interface{} {
	return map[string]interface{}{
		"id":        entity.Info.ID,
		"jid":       entity.Info.Chat,
		"from_me":   entity.Info.IsFromMe,
		"timestamp": entity.Info.Timestamp,
	}
}
func (f *MessageMapper) Marshal(msg *storage.StoredMessage) ([]byte, error) {
	// Temporary structure to hold JSON data
	var temp struct {
		Info                  types.MessageInfo             `json:"Info"`
		Message               json.RawMessage               `json:"Message"`
		IsEphemeral           bool                          `json:"IsEphemeral"`
		IsViewOnce            bool                          `json:"IsViewOnce"`
		IsViewOnceV2          bool                          `json:"IsViewOnceV2"`
		IsViewOnceV2Extension bool                          `json:"IsViewOnceV2Extension"`
		IsDocumentWithCaption bool                          `json:"IsDocumentWithCaption"`
		IsLottieSticker       bool                          `json:"IsLottieSticker"`
		IsEdit                bool                          `json:"IsEdit"`
		SourceWebMsg          json.RawMessage               `json:"SourceWebMsg"`
		UnavailableRequestID  string                        `json:"UnavailableRequestID"`
		RetryCount            int                           `json:"RetryCount"`
		NewsletterMeta        *events.NewsletterMessageMeta `json:"NewsletterMeta"`
		RawMessage            json.RawMessage               `json:"RawMessage"`
		Status                storage.Status                `json:"Status"`
	}

	if msg.Message.Message != nil {
		var err error
		temp.Message, err = protojson.Marshal(msg.Message.Message)
		if err != nil {
			return nil, err
		}
	}

	if msg.RawMessage != nil {
		var err error
		temp.RawMessage, err = protojson.Marshal(msg.RawMessage)
		if err != nil {
			return nil, err
		}
	}

	if msg.SourceWebMsg != nil {
		var err error
		temp.SourceWebMsg, err = protojson.Marshal(msg.SourceWebMsg)
		if err != nil {
			return nil, err
		}
	}

	temp.Info = msg.Info
	temp.IsEphemeral = msg.IsEphemeral
	temp.IsViewOnce = msg.IsViewOnce
	temp.IsViewOnceV2 = msg.IsViewOnceV2
	temp.IsViewOnceV2Extension = msg.IsViewOnceV2Extension
	temp.IsDocumentWithCaption = msg.IsDocumentWithCaption
	temp.IsLottieSticker = msg.IsLottieSticker
	temp.IsEdit = msg.IsEdit
	temp.UnavailableRequestID = msg.UnavailableRequestID
	temp.RetryCount = msg.RetryCount
	temp.NewsletterMeta = msg.NewsletterMeta
	temp.Status = msg.Status

	return json.Marshal(temp)
}

func (f *MessageMapper) Unmarshal(data []byte, msg *storage.StoredMessage) error {
	// Temporary structure to hold JSON data
	var temp struct {
		Info                  types.MessageInfo             `json:"Info"`
		Message               json.RawMessage               `json:"Message"`
		IsEphemeral           bool                          `json:"IsEphemeral"`
		IsViewOnce            bool                          `json:"IsViewOnce"`
		IsViewOnceV2          bool                          `json:"IsViewOnceV2"`
		IsViewOnceV2Extension bool                          `json:"IsViewOnceV2Extension"`
		IsDocumentWithCaption bool                          `json:"IsDocumentWithCaption"`
		IsLottieSticker       bool                          `json:"IsLottieSticker"`
		IsEdit                bool                          `json:"IsEdit"`
		SourceWebMsg          json.RawMessage               `json:"SourceWebMsg"`
		UnavailableRequestID  string                        `json:"UnavailableRequestID"`
		RetryCount            int                           `json:"RetryCount"`
		NewsletterMeta        *events.NewsletterMessageMeta `json:"NewsletterMeta"`
		RawMessage            json.RawMessage               `json:"RawMessage"`
		Status                storage.Status                `json:"Status"`
	}

	// Unmarshal into the temporary structure
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	msg.Message = &events.Message{}
	// Assign values to msg
	msg.Info = temp.Info
	msg.IsEphemeral = temp.IsEphemeral
	msg.IsViewOnce = temp.IsViewOnce
	msg.IsViewOnceV2 = temp.IsViewOnceV2
	msg.IsViewOnceV2Extension = temp.IsViewOnceV2Extension
	msg.IsDocumentWithCaption = temp.IsDocumentWithCaption
	msg.IsLottieSticker = temp.IsLottieSticker
	msg.IsEdit = temp.IsEdit
	msg.UnavailableRequestID = temp.UnavailableRequestID
	msg.RetryCount = temp.RetryCount
	msg.NewsletterMeta = temp.NewsletterMeta
	msg.Status = temp.Status

	// Unmarshal Message if present
	if !isNullJson(temp.Message) {
		msg.Message.Message = &waProto.Message{}
		if err := protojson.Unmarshal(temp.Message, msg.Message.Message); err != nil {
			return err
		}
	}

	// Unmarshal RawMessage if present
	if !isNullJson(temp.RawMessage) {
		msg.RawMessage = &waProto.Message{}
		if err := protojson.Unmarshal(temp.RawMessage, msg.RawMessage); err != nil {
			return err
		}
	}

	// Unmarshal SourceWebMsg if present
	if !isNullJson(temp.SourceWebMsg) {
		msg.SourceWebMsg = &waProto.WebMessageInfo{}
		if err := protojson.Unmarshal(temp.SourceWebMsg, msg.SourceWebMsg); err != nil {
			return err
		}
	}

	return nil
}
