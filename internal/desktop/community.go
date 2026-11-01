package desktop

import (
	"context"
	"projetop2p/internal/communities"
	"projetop2p/internal/community"
	"time"
)

func (c *Core) CreateCommunity(name, description, icon string) (string, error) {
	return c.groups.Create(name, description, icon)
}
func (c *Core) CommunityState(id, channel string, before uint64) (communities.View, error) {
	return c.groups.View(id, channel, before)
}
func (c *Core) CommunityAction(id, kind string, body community.Body) (communities.Receipt, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.communityTimeout())
	defer cancel()
	return c.groups.Action(ctx, id, kind, body)
}
func (c *Core) CommunityInvite(id string) (string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.communityTimeout())
	defer cancel()
	return c.groups.Invite(ctx, id)
}
func (c *Core) JoinCommunity(raw string) (communities.Receipt, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.communityTimeout())
	defer cancel()
	return c.groups.Join(ctx, raw)
}
func (c *Core) SyncCommunity(id string) error {
	ctx, cancel := context.WithTimeout(c.ctx, c.communityTimeout())
	defer cancel()
	return c.groups.Sync(ctx, id)
}
func (c *Core) DiscardCommunityPending(id, op string) error { return c.groups.Discard(id, op) }

func (c *Core) communityTimeout() time.Duration {
	s, e := c.db.Settings()
	if e == nil && mode(s) == "integrated" {
		return 45 * time.Second
	}
	return 10 * time.Second
}

func (c *Core) CommunityInvitePolicy(id string, options community.InviteOptions) (string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.communityTimeout())
	defer cancel()
	return c.groups.InvitePolicy(ctx, id, &options)
}

func (c *Core) CommunityInviteStatus(id, raw string) (community.InvitationStatus, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.communityTimeout())
	defer cancel()
	return c.groups.InviteStatus(ctx, id, raw)
}
