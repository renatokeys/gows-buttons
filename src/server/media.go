package server

import (
	"context"
	"encoding/json"
	"github.com/devlikeapro/gows/proto"
	"go.mau.fi/whatsmeow/proto/waE2E"
)

func (s *Server) DownloadMedia(ctx context.Context, req *__.DownloadMediaRequest) (*__.DownloadMediaResponse, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	// Parse Message from JSON provided
	msg, buildMessageError := BuildMessage(req.GetMessage())
	if buildMessageError != nil {
		cli.Log.Warnf("Failed to build message from JSON: %v", buildMessageError)
	}

	// If parsing JSON failed - fetch it from storage
	if msg == nil && req.MessageId != "" {
		cli.Log.Debugf("Fetching message from storage '%s'", req.MessageId)
		storedMessage, err := cli.Storage.Messages.GetMessage(req.GetMessageId())
		if err != nil {
			cli.Log.Warnf("Failed to fetch message '%s' from storage: %v", req.MessageId, err)
		}
		if storedMessage != nil {
			cli.Log.Infof("Found message '%s' in storage, using it to fetch media", req.MessageId)
			msg = storedMessage.Message.Message
		}
	}

	if msg == nil {
		cli.Log.Warnf("Failed to build message '%s' from JSON or fetch storage", req.MessageId)
		return nil, buildMessageError
	}

	resp, err := cli.DownloadAny(ctx, msg)
	if err != nil {
		cli.Log.Errorf("Failed to download media for '%s' message: %v", req.MessageId, err)
		return nil, nil
	}
	return &__.DownloadMediaResponse{Content: resp}, nil
}

// BuildMessage builds a message from the given JSON data
func BuildMessage(data string) (*waE2E.Message, error) {
	var message waE2E.Message
	err := json.Unmarshal([]byte(data), &message)
	if err != nil {
		return nil, err
	}
	return &message, nil
}
