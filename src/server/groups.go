package server

import (
	"context"
	"fmt"
	"github.com/devlikeapro/gows/media"
	__ "github.com/devlikeapro/gows/proto"
	"github.com/devlikeapro/gows/storage"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

func (s *Server) GetGroups(ctx context.Context, req *__.Session) (*__.JsonList, error) {
	cli, err := s.Sm.Get(req.GetId())
	if err != nil {
		return nil, err
	}
	sort := storage.Sort{Field: "jid", Order: storage.SortAsc}
	groups, err := cli.Storage.Groups.GetAllGroups(sort, storage.Pagination{})
	if err != nil {
		return nil, err
	}
	return toJsonList(groups)
}

func (s *Server) FetchGroups(ctx context.Context, req *__.Session) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.GetId())
	if err != nil {
		return nil, err
	}
	err = cli.Storage.Groups.FetchGroups()
	if err != nil {
		return nil, err
	}
	return &__.Empty{}, nil
}

func (s *Server) GetGroupInfo(ctx context.Context, req *__.JidRequest) (*__.Json, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}
	info, err := cli.Storage.Groups.GetGroup(jid)
	if err != nil {
		return nil, err
	}
	return toJson(info)
}

func (s *Server) CreateGroup(ctx context.Context, req *__.CreateGroupRequest) (*__.Json, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jids := make([]types.JID, 0, len(req.Participants))
	for ind, j := range req.Participants {
		jid, err := types.ParseJID(j)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JID at index %d (%d): %w", ind, j, err)
		}
		jids = append(jids, jid)
	}

	request := whatsmeow.ReqCreateGroup{
		Name:         req.Name,
		Participants: jids,
	}
	group, err := cli.CreateGroup(request)
	if err != nil {
		return nil, err
	}
	return toJson(group)
}

func (s *Server) LeaveGroup(ctx context.Context, req *__.JidRequest) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}
	err = cli.LeaveGroup(jid)
	if err != nil {
		return nil, err
	}
	return &__.Empty{}, nil
}

func (s *Server) GetGroupInviteLink(ctx context.Context, req *__.JidRequest) (*__.OptionalString, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}
	link, err := cli.GetGroupInviteLink(jid, false)
	if err != nil {
		return nil, err
	}
	return &__.OptionalString{Value: link}, nil
}

func (s *Server) RevokeGroupInviteLink(ctx context.Context, req *__.JidRequest) (*__.OptionalString, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}
	link, err := cli.GetGroupInviteLink(jid, false)
	if err != nil {
		return nil, err
	}
	return &__.OptionalString{Value: link}, nil
}

func (s *Server) GetGroupInfoFromLink(ctx context.Context, req *__.GroupCodeRequest) (*__.Json, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	info, err := cli.GetGroupInfoFromLink(req.GetCode())
	if err != nil {
		return nil, err
	}
	return toJson(info)
}

func (s *Server) JoinGroupWithLink(ctx context.Context, req *__.GroupCodeRequest) (*__.Json, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := cli.JoinGroupWithLink(req.GetCode())
	if err != nil {
		return nil, err
	}
	resp := make(map[string]interface{})
	resp["jid"] = jid
	return toJson(resp)
}

func (s *Server) SetGroupName(ctx context.Context, req *__.JidStringRequest) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}
	err = cli.SetGroupName(jid, req.GetValue())
	if err != nil {
		return nil, err
	}
	return &__.Empty{}, nil
}

func (s *Server) SetGroupDescription(ctx context.Context, req *__.JidStringRequest) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}
	err = cli.SetGroupDescription(jid, req.GetValue())
	if err != nil {
		return nil, err
	}
	return &__.Empty{}, nil
}

func (s *Server) SetGroupPicture(ctx context.Context, req *__.SetPictureRequest) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}
	content := req.Picture
	var picture []byte
	if len(content) != 0 {
		picture, err = media.ProfilePicture(req.Picture)
		if err != nil {
			return nil, err
		}
	}
	_, err = cli.SetGroupPhoto(jid, picture)
	if err != nil {
		return nil, err
	}
	return &__.Empty{}, nil
}

func (s *Server) SetGroupLocked(ctx context.Context, req *__.JidBoolRequest) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}
	err = cli.SetGroupLocked(jid, req.GetValue())
	if err != nil {
		return nil, err
	}
	return &__.Empty{}, nil
}

func (s *Server) SetGroupAnnounce(ctx context.Context, req *__.JidBoolRequest) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}
	err = cli.SetGroupAnnounce(jid, req.GetValue())
	if err != nil {
		return nil, err
	}
	return &__.Empty{}, nil
}

func (s *Server) UpdateGroupParticipants(ctx context.Context, req *__.UpdateParticipantsRequest) (*__.JsonList, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}
	participants := make([]types.JID, 0, len(req.GetParticipants()))
	for ind, p := range req.GetParticipants() {
		jid, err := types.ParseJID(p)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JID at index %d (%s): %w", ind, p, err)
		}
		participants = append(participants, jid)
	}

	var action whatsmeow.ParticipantChange
	switch req.Action {
	case __.ParticipantAction_ADD:
		action = whatsmeow.ParticipantChangeAdd
	case __.ParticipantAction_REMOVE:
		action = whatsmeow.ParticipantChangeRemove
	case __.ParticipantAction_PROMOTE:
		action = whatsmeow.ParticipantChangePromote
	case __.ParticipantAction_DEMOTE:
		action = whatsmeow.ParticipantChangeDemote
	default:
		return nil, fmt.Errorf("unknown action: %v", req.Action)
	}
	result, err := cli.UpdateGroupParticipants(jid, participants, action)
	if err != nil {
		return nil, err
	}
	return toJsonList(result)
}
