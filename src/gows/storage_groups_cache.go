package gows

import (
	"github.com/devlikeapro/gows/storage"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"sync"
	"time"
)

const refreshInterval = 24 * time.Hour

type GroupCacheStorage struct {
	groups storage.GroupStorage
	gows   *GoWS

	lastTimeRefreshed time.Time
	refreshLock       sync.Mutex

	log waLog.Logger
}

func NewGroupCacheStorage(groups storage.GroupStorage, gows *GoWS) *GroupCacheStorage {
	return &GroupCacheStorage{
		groups: groups,
		gows:   gows,
		log:    gows.Log.Sub("GroupCacheStorage"),
	}
}

var _ storage.GroupStorage = (*GroupCacheStorage)(nil)

func (g *GroupCacheStorage) shouldRefresh() bool {
	return time.Since(g.lastTimeRefreshed) > refreshInterval
}

func (g *GroupCacheStorage) FetchGroups() error {
	g.refreshLock.Lock()
	defer g.refreshLock.Unlock()
	return g.fetchGroupsUnlocked()
}
func (g *GroupCacheStorage) fetchGroupsUnlocked() error {
	g.log.Debugf("Refreshing groups")
	groups, err := g.gows.GetJoinedGroups()
	if err != nil {
		return err
	}
	err = g.groups.DeleteGroups()
	if err != nil {
		return err
	}
	for _, group := range groups {
		err = g.groups.UpsertOneGroup(group)
		if err != nil {
			return err
		}
	}
	g.lastTimeRefreshed = time.Now()
	g.log.Debugf("Groups refreshed")
	return nil

}

func (g *GroupCacheStorage) fetchGroupsIfNeeded() error {
	g.refreshLock.Lock()
	defer g.refreshLock.Unlock()
	if g.shouldRefresh() {
		g.log.Debugf("Last time refreshed groups %s ago", time.Since(g.lastTimeRefreshed))
		return g.fetchGroupsUnlocked()
	}
	return nil
}

func (g *GroupCacheStorage) UpsertOneGroup(group *types.GroupInfo) error {
	return g.groups.UpsertOneGroup(group)
}

func (g *GroupCacheStorage) GetAllGroups(sort storage.Sort, pagination storage.Pagination) ([]*types.GroupInfo, error) {
	err := g.fetchGroupsIfNeeded()
	if err != nil {
		return nil, err
	}
	return g.groups.GetAllGroups(sort, pagination)
}

func (g *GroupCacheStorage) GetGroup(jid types.JID) (*types.GroupInfo, error) {
	err := g.fetchGroupsIfNeeded()
	if err != nil {
		return nil, err
	}
	return g.groups.GetGroup(jid)
}

func (g *GroupCacheStorage) DeleteGroup(jid types.JID) error {
	return g.groups.DeleteGroup(jid)
}

func (g *GroupCacheStorage) DeleteGroups() error {
	return g.groups.DeleteGroups()
}
