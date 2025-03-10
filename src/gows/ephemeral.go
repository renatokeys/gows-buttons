package gows

import (
	"errors"
	"github.com/devlikeapro/gows/storage"
	"github.com/golang/protobuf/proto"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (gows *GoWS) PopulateContextInfoDisappearingSettings(info *waE2E.ContextInfo, jid types.JID) (*waE2E.ContextInfo, error) {
	setting, err := gows.getEphemeralSettings(jid)
	if err != nil {
		return info, err
	}

	if !setting.IsEphemeral {
		return info, nil
	}

	if info == nil {
		info = &waE2E.ContextInfo{}
	}
	info.Expiration = proto.Uint32(setting.Setting.Expiration)
	info.EphemeralSettingTimestamp = setting.Setting.Timestamp
	info.DisappearingMode = &waE2E.DisappearingMode{
		Initiator:     setting.Setting.Initiator,
		Trigger:       setting.Setting.Trigger,
		InitiatedByMe: setting.Setting.InitiatedByMe,
	}
	return info, nil
}

func (gows *GoWS) getEphemeralSettings(jid types.JID) (*storage.StoredChatEphemeralSetting, error) {
	if jid.Server == types.GroupServer {
		group, err := gows.Storage.Groups.GetGroup(jid)
		if errors.Is(err, storage.ErrNotFound) {
			return storage.NotEphemeral(jid), nil
		}
		if err != nil {
			return nil, err
		}
		if !group.IsEphemeral {
			return storage.NotEphemeral(jid), nil
		}

		setting := &storage.StoredChatEphemeralSetting{
			ID:          jid,
			IsEphemeral: true,
			Setting: &storage.EphemeralSetting{
				Initiator:     waE2E.DisappearingMode_CHANGED_IN_CHAT.Enum(),
				Trigger:       waE2E.DisappearingMode_CHAT_SETTING.Enum(),
				InitiatedByMe: proto.Bool(false),
				Expiration:    group.DisappearingTimer,
			},
		}
		return setting, nil
	}

	if jid.Server == types.DefaultUserServer || jid.Server == types.HiddenUserServer {
		setting, err := gows.Storage.ChatEphemeralSetting.GetChatEphemeralSetting(jid)
		if errors.Is(err, storage.ErrNotFound) {
			return storage.NotEphemeral(jid), nil
		}
		if err != nil {
			return nil, err
		}
		if setting == nil {
			return storage.NotEphemeral(jid), nil
		}
		return setting, nil
	}
	return storage.NotEphemeral(jid), nil
}

// ExtractEphemeralSettingsFromMsg extracts ephemeral settings from a message event (from the initial message).
func (gows *GoWS) ExtractEphemeralSettingsFromMsg(event *events.Message) *storage.StoredChatEphemeralSetting {
	if event.Info.Chat.Server != types.DefaultUserServer && event.Info.Chat.Server != types.HiddenUserServer {
		return nil
	}
	contextInfo := ExtractContextInfo(event)
	if contextInfo == nil {
		return nil
	}
	if contextInfo.Expiration == nil {
		return nil
	}

	setting := storage.NotEphemeral(event.Info.Chat)
	setting.Setting = &storage.EphemeralSetting{
		Initiator:     contextInfo.DisappearingMode.Initiator,
		Trigger:       contextInfo.DisappearingMode.Trigger,
		InitiatedByMe: contextInfo.DisappearingMode.InitiatedByMe,
		Timestamp:     contextInfo.EphemeralSettingTimestamp,
		Expiration:    *contextInfo.Expiration,
	}
	setting.IsEphemeral = true
	return setting
}

// ExtractEphemeralSettingsChanged extracts ephemeral settings from a message event.
func (gows *GoWS) ExtractEphemeralSettingsChanged(event *events.Message) *storage.StoredChatEphemeralSetting {
	if event.Message == nil || event.Message.ProtocolMessage == nil {
		return nil
	}
	protocol := event.Message.ProtocolMessage
	type_ := *protocol.Type
	switch type_ {
	case waE2E.ProtocolMessage_EPHEMERAL_SETTING, waE2E.ProtocolMessage_EPHEMERAL_SYNC_RESPONSE:
		var setting *storage.StoredChatEphemeralSetting
		setting = storage.NotEphemeral(event.Info.Chat)
		isEphemeral := protocol.EphemeralExpiration != nil && *protocol.EphemeralExpiration > 0
		if isEphemeral {
			setting.IsEphemeral = true
			timestamp := event.Info.Timestamp.Unix()
			setting.Setting = &storage.EphemeralSetting{
				Initiator:     protocol.DisappearingMode.Initiator,
				Trigger:       protocol.DisappearingMode.Trigger,
				InitiatedByMe: protocol.DisappearingMode.InitiatedByMe,
				Timestamp:     &timestamp,
				Expiration:    *protocol.EphemeralExpiration,
			}
		}
		return setting
	default:
		return nil
	}
}
