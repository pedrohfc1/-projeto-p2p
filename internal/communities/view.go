package communities

import (
	"projetop2p/internal/community"
	"projetop2p/internal/localdb"
	"sort"
	"strings"
)

type Summary struct {
	IconHash       string              `json:"iconHash,omitempty"`
	AuthorityState string              `json:"authorityState"`
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	Icon           string              `json:"icon"`
	Owner          string              `json:"owner"`
	Role           string              `json:"role"`
	Scope          string              `json:"scope"`
	Sequence       uint64              `json:"sequence"`
	Head           string              `json:"head"`
	Archived       bool                `json:"archived"`
	Denied         bool                `json:"denied"`
	LastSync       int64               `json:"lastSync"`
	Error          string              `json:"error"`
	Members        []community.Member  `json:"members"`
	Channels       []community.Channel `json:"channels"`
}
type MessageView struct {
	ID       string `json:"id"`
	Sequence uint64 `json:"sequence"`
	Author   string `json:"author"`
	Name     string `json:"name"`
	Text     string `json:"text"`
	Time     int64  `json:"time"`
	Status   string `json:"status"`
	Error    string `json:"error"`
}
type View struct {
	Community Summary       `json:"community"`
	Messages  []MessageView `json:"messages"`
	HasMore   bool          `json:"hasMore"`
}

func (s *Service) summary(r localdb.CommunityRecord) Summary {
	scope := r.Scope
	if r.Owner == s.id {
		_, scope = s.environment()
	}
	v := Summary{ID: r.ID, Name: r.Name, Description: r.State.Description, Icon: r.State.Icon, Owner: r.Owner, Role: r.State.Role(s.id), Scope: scope, Sequence: r.State.Sequence, Head: r.State.Head, Archived: r.State.Archived, Denied: r.Denied, LastSync: r.LastSync, Error: r.LastError, Members: []community.Member{}, Channels: []community.Channel{}}
	v.AuthorityState = "synchronized"
	if r.LastSync == 0 {
		v.AuthorityState = "sync_pending"
	}
	if r.LastError != "" {
		v.AuthorityState = "sync_pending"
		for _, code := range []string{"invalid_address", "transport_unavailable", "timeout", "connectivity_failure", "owner_unavailable"} {
			if strings.Contains(r.LastError, code+":") {
				v.AuthorityState = code
				break
			}
		}
	}
	if r.Denied {
		v.AuthorityState = "access_revoked"
	}
	if r.Owner == s.id {
		v.AuthorityState = "local"
	}
	for _, m := range r.State.Members {
		v.Members = append(v.Members, m)
	}
	artwork, err := s.db.Artwork(r.ID)
	if err == nil && !r.Denied && r.State.Role(s.id) != "" && artwork.Verify(r.State) == nil {
		v.IconHash = artwork.Icons[""]
	} else {
		artwork.Icons = nil
	}
	for _, ch := range r.State.Channels {
		if !ch.Deleted {
			ch.IconHash = artwork.Icons[ch.ID]
		}
		v.Channels = append(v.Channels, ch)
	}
	sort.Slice(v.Members, func(i, j int) bool { return v.Members[i].ID < v.Members[j].ID })
	sort.Slice(v.Channels, func(i, j int) bool { return v.Channels[i].ID < v.Channels[j].ID })
	return v
}
func (s *Service) List() ([]Summary, error) {
	records, e := s.db.Communities()
	if e != nil {
		return nil, e
	}
	out := []Summary{}
	for _, r := range records {
		out = append(out, s.summary(r))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
func (s *Service) View(id, channel string, before uint64) (View, error) {
	r, e := s.db.Community(id)
	if e != nil {
		return View{}, e
	}
	v := View{Community: s.summary(r), Messages: []MessageView{}}
	events, e := s.db.CommunityHistory(id, channel, before, 100)
	if e != nil {
		return v, e
	}
	seen := map[string]bool{}
	for _, c := range events {
		m := c.Operation
		seen[m.ID] = true
		name := r.State.Members[m.Author].Name
		if name == "" {
			name = m.Author[:12]
		}
		status := "recebida"
		if m.Author == s.id {
			status = "confirmada"
		}
		v.Messages = append(v.Messages, MessageView{ID: m.ID, Sequence: c.Sequence, Author: m.Author, Name: name, Text: m.Body.Text, Time: c.At, Status: status})
	}
	v.HasMore = len(events) == 100
	{
		messages, err := s.db.PeerTextHistory(id, channel, before)
		if err != nil {
			return v, err
		}
		for _, entry := range messages {
			m := entry.Message.Operation
			if committed, _ := s.db.FindCommunityOperation(id, m.ID); committed > 0 {
				continue
			}
			if seen[m.ID] {
				continue
			}
			seen[m.ID] = true
			v.Messages = append(v.Messages, MessageView{ID: m.ID, Sequence: entry.Sequence, Author: m.Author, Name: r.State.Members[m.Author].Name, Text: m.Body.Text, Time: m.Issued, Status: "assinada pelo autor"})
		}
		v.HasMore = v.HasMore || len(messages) == 100
	}
	if before == 0 {
		pending, e := s.db.CommunityPending(id)
		if e != nil {
			return v, e
		}
		for _, q := range pending {
			if seen[q.Operation.ID] || q.Operation.Kind != community.Text || q.Operation.Body.ChannelID != channel {
				continue
			}
			v.Messages = append(v.Messages, MessageView{ID: q.Operation.ID, Author: q.Operation.Author, Name: "Você", Text: q.Operation.Body.Text, Time: q.Operation.Issued, Status: q.Status, Error: q.Error})
		}
	}
	return v, nil
}
