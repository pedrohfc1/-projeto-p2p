package desktop

import (
	"context"
	"errors"
	"github.com/libp2p/go-libp2p/core/peer"
	"projetop2p/internal/chat"
	"projetop2p/internal/community"
	"projetop2p/internal/invite"
	"projetop2p/internal/localdb"
	"projetop2p/internal/network"
	"sort"
	"strings"
	"sync"
	"time"
)

type activityState struct {
	mu         sync.Mutex
	foreground bool
	out        chan chat.Event
	cursors    map[string]uint64
	typing     map[string]typingState
	lastTyping map[string]time.Time
	cursor     int
	reading    map[string]time.Time
	presence   map[string]voicePresence
}
type voicePresence struct {
	Group, Channel, Session string
	Muted, Deaf             bool
	Until                   time.Time
}

func newActivityState() activityState {
	return activityState{foreground: true, out: make(chan chat.Event, 16)}
}

type typingState struct {
	Room  chat.Room
	Actor string
	Until time.Time
}
type ActivityView struct {
	Overlays  map[string]chat.Overlay    `json:"overlays"`
	Reactions map[string][]chat.Reaction `json:"reactions"`
	Typing    []string                   `json:"typing"`
}

func (c *Core) selfID() string              { id, _ := peer.IDFromPrivateKey(c.key); return id.String() }
func (c *Core) activityNode() *network.Node { c.mu.Lock(); defer c.mu.Unlock(); return c.node }
func (c *Core) activityHandle(remote string, req community.Request) community.Response {
	fail := func(e error) community.Response {
		return community.Response{Error: e.Error(), Code: "activity_failure"}
	}
	if req.ActivityRoom == nil {
		return fail(errors.New("activity room required"))
	}
	r := *req.ActivityRoom
	self := c.selfID()
	if r.Group == "" {
		r.Peer = remote
	}
	if e := c.db.ActivityAccess(r, self, remote); e != nil {
		return fail(e)
	}
	if req.Method == "activity.post" {
		e := req.Activity
		if e == nil || e.Actor != remote || e.Room.Group != r.Group || (r.Group == "" && e.Room.Peer != self) {
			return fail(errors.New("activity authenticated actor/audience mismatch"))
		}
		if e.Kind == "voice" {
			if err := c.acceptVoicePresence(*e, remote); err != nil {
				return fail(err)
			}
		} else if e.Kind == "typing" {
			if err := c.acceptTyping(*e, remote); err != nil {
				return fail(err)
			}
		} else if err := c.db.SaveActivity(*e, self, remote); err != nil {
			return fail(err)
		}
		return community.Response{Ack: e.ID}
	}
	if req.Method != "activity.sync" {
		return fail(errors.New("unsupported activity method"))
	}
	journalContext := chat.Context(r, self, self)
	head := c.db.ActivityHead(journalContext)
	if req.ActivityAfter > head {
		return community.Response{ActivityCursor: head}
	}
	entries, e := c.db.Activities(journalContext, req.ActivityAfter)
	if e != nil {
		return fail(e)
	}
	if service, err := c.voiceService(); err == nil && service.View().Joined && len(entries) > 4 {
		entries = entries[:4]
	}
	res := community.Response{Activities: entries, ActivityCursor: req.ActivityAfter}
	if len(entries) > 0 {
		res.ActivityCursor = entries[len(entries)-1].Sequence
	}
	return res
}
func (c *Core) acceptTyping(e chat.Event, remote string) error {
	if err := chat.Verify(e, time.Now()); err != nil {
		return err
	}
	self := c.selfID()
	if err := c.db.ActivityAccess(e.Room, self, remote); err != nil {
		return err
	}
	if e.Room.Group != "" {
		r, err := c.db.Community(e.Room.Group)
		if err != nil {
			return err
		}
		ch, ok := r.State.Channels[e.Room.Channel]
		if r.State.Root != e.Root || r.State.InviteEpoch != e.Epoch || r.State.Role(e.Actor) == "" || r.State.Archived || !ok || ch.Deleted || ch.Kind != "text" {
			return errors.New("typing authorization denied")
		}
	}
	key := chat.Context(e.Room, self, e.Actor) + ":" + e.Room.Channel + ":" + e.Actor
	c.activity.mu.Lock()
	defer c.activity.mu.Unlock()
	if c.activity.typing == nil {
		c.activity.typing = map[string]typingState{}
	}
	for k, v := range c.activity.typing {
		if time.Now().After(v.Until) {
			delete(c.activity.typing, k)
		}
	}
	if len(c.activity.typing) >= 256 {
		return errors.New("typing limit")
	}
	if !e.Enabled {
		delete(c.activity.typing, key)
	} else {
		c.activity.typing[key] = typingState{e.Room, e.Actor, time.UnixMilli(e.At).Add(6 * time.Second)}
	}
	return nil
}
func (c *Core) ChatAction(room chat.Room, kind, target, text, emoji string, enabled bool) (string, error) {
	self := c.selfID()
	if e := c.db.ActivityAccess(room, self, self); e != nil {
		return "", e
	}
	if kind == "typing" {
		prefs, _ := c.db.NotificationPreferences()
		if !prefs.Typing {
			return "", nil
		}
	}
	targetAuthor := ""
	if room.Group == "" && kind != "typing" {
		var err error
		targetAuthor, target, err = c.db.DirectTargetAuthor(self, room.Peer, target)
		if err != nil {
			return "", err
		}
	}
	event := chat.Event{TargetAuthor: targetAuthor, Room: room, Kind: kind, Target: target, Text: text, Emoji: emoji, Enabled: enabled, Nonce: community.RandomID()}
	if room.Group != "" {
		r, e := c.db.Community(room.Group)
		if e != nil {
			return "", e
		}
		event.Root = r.State.Root
		event.Head = r.State.Head
		event.Epoch = r.State.InviteEpoch
	}
	event, e := chat.Sign(c.key, event)
	if e != nil {
		return "", e
	}
	if kind == "typing" {
		key := chat.Context(room, self, self) + ":" + room.Channel
		c.activity.mu.Lock()
		if c.activity.lastTyping == nil {
			c.activity.lastTyping = map[string]time.Time{}
		}
		if time.Since(c.activity.lastTyping[key]) < 3*time.Second {
			c.activity.mu.Unlock()
			return event.ID, nil
		}
		if len(c.activity.lastTyping) > 256 {
			c.activity.lastTyping = map[string]time.Time{}
		}
		c.activity.lastTyping[key] = time.Now()
		c.activity.mu.Unlock()
		if e = c.acceptTyping(event, self); e != nil {
			return "", e
		}
		c.sendTyping(event)
	} else {
		if e = c.db.SaveActivity(event, self, self); e != nil {
			return "", e
		}
		select {
		case c.activity.out <- event:
		default:
		}
	}
	return event.ID, nil
}
func (c *Core) activityTargets(room chat.Room) map[string][]string {
	out := map[string][]string{}
	self := c.selfID()
	if room.Group == "" {
		items, _ := c.db.Contacts()
		for _, p := range items {
			if p.ID == room.Peer && !p.Unsolicited {
				out[p.ID] = p.Addresses
			}
		}
		return out
	}
	r, e := c.db.Community(room.Group)
	if e != nil || r.Denied || r.State.Role(self) == "" {
		return out
	}
	routes, _ := c.db.CommunityRoutes(room.Group)
	for id, raw := range routes {
		ticket, e := invite.Parse(raw, time.Now())
		if e == nil && ticket.ID == id && id != self && r.State.Role(id) != "" {
			if n := c.activityNode(); n != nil && ticket.Route != "" {
				_ = n.RegisterInvitationRoute(id, ticket.Route)
			}
			out[id] = ticket.Addresses
		}
	}
	if r.Owner != self && r.State.Role(r.Owner) != "" {
		if _, ok := out[r.Owner]; !ok {
			out[r.Owner] = r.Addresses
		}
	}
	return out
}
func (c *Core) sendTyping(e chat.Event) {
	n := c.activityNode()
	if n == nil {
		return
	}
	ctx, cancel := context.WithTimeout(c.ctx, 1500*time.Millisecond)
	defer cancel()
	count := 0
	for id, addresses := range c.activityTargets(e.Room) {
		if e.Kind == "voice" {
			service, err := c.voiceService()
			if err != nil {
				continue
			}
			found := false
			for _, p := range service.View().Participants {
				if p.ID == id {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		if count >= 8 {
			break
		}
		count++
		_, _ = n.CommunityRequest(ctx, id, addresses, community.Request{Method: "activity.post", ActivityRoom: &e.Room, Activity: &e})
	}
}
func (c *Core) ActivityView(room chat.Room, ids []string, replies bool) (ActivityView, error) {
	if len(ids) > 300 {
		return ActivityView{}, errors.New("too many activity targets")
	}
	self := c.selfID()
	contextKey := chat.Context(room, self, self)
	if room.Group != "" {
		contextKey += ":" + room.Channel
	}
	c.activity.mu.Lock()
	if c.activity.reading == nil {
		c.activity.reading = map[string]time.Time{}
	}
	if len(c.activity.reading) > 512 {
		c.activity.reading = map[string]time.Time{}
	}
	visible := c.activity.foreground
	if replies && visible {
		c.activity.reading[contextKey] = time.Now().Add(5 * time.Second)
	} else {
		delete(c.activity.reading, contextKey)
	}
	c.activity.mu.Unlock()
	if replies && visible {
		_ = c.db.MarkRead(contextKey)
	}
	overlays, e := c.db.ActivityOverlays(chat.Context(room, self, self), ids, room.Channel, replies)
	if e != nil {
		return ActivityView{}, e
	}
	v := ActivityView{Overlays: overlays, Reactions: map[string][]chat.Reaction{}, Typing: []string{}}
	for id, p := range overlays {
		v.Reactions[id] = chat.Counts(p, self)
	}
	c.activity.mu.Lock()
	for _, t := range c.activity.typing {
		if t.Actor != self && time.Now().Before(t.Until) && chat.Context(t.Room, self, t.Actor) == chat.Context(room, self, self) && t.Room.Channel == room.Channel {
			v.Typing = append(v.Typing, t.Actor)
		}
	}
	c.activity.mu.Unlock()
	sort.Strings(v.Typing)
	return v, nil
}
func (c *Core) activityWatch() {
	defer c.wg.Done()
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case event := <-c.activity.out:
			c.sendTyping(event)
			continue
		case <-tick.C:
		}
		c.pollInbox()
		c.publishVoicePresence()
		n := c.activityNode()
		if n == nil {
			continue
		}
		self := c.selfID()
		rooms := []chat.Room{}
		contacts, _ := c.db.Contacts()
		for _, p := range contacts {
			if !p.Unsolicited {
				rooms = append(rooms, chat.Room{Peer: p.ID})
			}
		}
		groups, _ := c.db.Communities()
		for _, g := range groups {
			if !g.Denied && g.State.Role(self) != "" {
				rooms = append(rooms, chat.Room{Group: g.ID})
			}
		}
		if len(rooms) == 0 {
			continue
		}
		room := rooms[c.activity.cursor%len(rooms)]
		c.activity.cursor++
		count := 0
		for id, addresses := range c.activityTargets(room) {
			if count >= 2 {
				break
			}
			count++
			key := chat.Context(room, self, self) + ":" + id
			c.activity.mu.Lock()
			if c.activity.cursors == nil {
				c.activity.cursors = map[string]uint64{}
			}
			after, known := c.activity.cursors[key]
			if !known {
				after, _ = c.db.ActivityCursor(key)
				c.activity.cursors[key] = after
			}
			c.activity.mu.Unlock()
			ctx, cancel := context.WithTimeout(c.ctx, 2*time.Second)
			res, e := n.CommunityRequest(ctx, id, addresses, community.Request{Method: "activity.sync", ActivityRoom: &room, ActivityAfter: after})
			cancel()
			if e == nil && res.Error == "" && res.ActivityCursor < after {
				c.activity.mu.Lock()
				c.activity.cursors[key] = 0
				c.activity.mu.Unlock()
				_ = c.db.SaveActivityCursor(key, 0)
				continue
			}
			if e != nil || res.Error != "" || len(res.Activities) > 16 {
				continue
			}
			cursor := after
			for _, entry := range res.Activities {
				if entry.Sequence <= cursor || entry.Sequence > res.ActivityCursor {
					break
				}
				event := entry.Event
				if chat.Context(event.Room, self, event.Actor) != chat.Context(room, self, self) {
					break
				}
				if err := c.db.SaveActivity(event, self, id); err != nil {
					if strings.Contains(err.Error(), "target_missing") || strings.Contains(err.Error(), "sync_required") {
						break
					}
					if chat.Verify(event, time.UnixMilli(event.At)) != nil {
						break
					}
				}
				cursor = entry.Sequence
			}
			c.activity.mu.Lock()
			c.activity.cursors[key] = cursor
			c.activity.mu.Unlock()
			if cursor != after {
				_ = c.db.SaveActivityCursor(key, cursor)
			}
		}
	}
}

func (c *Core) pollInbox() {
	active := map[string]bool{}
	c.activity.mu.Lock()
	for key, until := range c.activity.reading {
		if time.Now().Before(until) {
			active[key] = true
		} else {
			delete(c.activity.reading, key)
		}
	}
	c.activity.mu.Unlock()
	_ = c.db.PollInbox(c.selfID(), active)
}
func (c *Core) NotificationPreferences() (localdb.NotificationPreferences, error) {
	return c.db.NotificationPreferences()
}
func (c *Core) SaveNotificationPreferences(p localdb.NotificationPreferences) error {
	return c.db.SaveNotificationPreferences(p)
}
func (c *Core) DrainNotices() ([]localdb.Notice, error) { return c.db.DrainNotices() }

func (c *Core) acceptVoicePresence(e chat.Event, remote string) error {
	if err := chat.Verify(e, time.Now()); err != nil {
		return err
	}
	if e.Actor != remote {
		return errors.New("voice presence actor mismatch")
	}
	if err := c.db.ActivityAccess(e.Room, c.selfID(), remote); err != nil {
		return err
	}
	service, err := c.voiceService()
	if err != nil || !service.HasSession(e.Room.Group, e.Room.Channel, e.Actor, e.Target) {
		return errors.New("voice presence requires current authorized session")
	}
	r, err := c.db.Community(e.Room.Group)
	if err != nil || r.State.Root != e.Root || r.State.InviteEpoch != e.Epoch || r.State.Role(e.Actor) == "" {
		return errors.New("voice presence membership changed")
	}
	c.activity.mu.Lock()
	defer c.activity.mu.Unlock()
	if c.activity.presence == nil {
		c.activity.presence = map[string]voicePresence{}
	}
	for id, p := range c.activity.presence {
		if time.Now().After(p.Until) {
			delete(c.activity.presence, id)
		}
	}
	if len(c.activity.presence) >= 6 {
		return errors.New("voice presence limit")
	}
	c.activity.presence[e.Actor] = voicePresence{e.Room.Group, e.Room.Channel, e.Target, e.Enabled, e.Deaf, time.Now().Add(6 * time.Second)}
	return nil
}
func (c *Core) publishVoicePresence() {
	service, err := c.voiceService()
	if err != nil {
		return
	}
	view := service.View()
	if !view.Joined {
		return
	}
	r, err := c.db.Community(view.Community)
	if err != nil {
		return
	}
	event, err := chat.Sign(c.key, chat.Event{Room: chat.Room{Group: view.Community, Channel: view.Channel}, Root: r.State.Root, Head: r.State.Head, Epoch: r.State.InviteEpoch, Nonce: community.RandomID(), Kind: "voice", Target: view.Session, Enabled: view.Muted || view.Deaf, Deaf: view.Deaf})
	if err != nil {
		return
	}
	c.sendTyping(event)
}

func (c *Core) SetChatForeground(visible bool) {
	c.activity.mu.Lock()
	c.activity.foreground = visible
	if !visible {
		c.activity.reading = map[string]time.Time{}
	}
	c.activity.mu.Unlock()
}
