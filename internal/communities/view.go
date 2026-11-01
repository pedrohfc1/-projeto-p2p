package communities

import (
	"projetop2p/internal/community"
	"projetop2p/internal/localdb"
	"sort"
)

type Summary struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Icon        string              `json:"icon"`
	Owner       string              `json:"owner"`
	Role        string              `json:"role"`
	Scope       string              `json:"scope"`
	Sequence    uint64              `json:"sequence"`
	Head        string              `json:"head"`
	Archived    bool                `json:"archived"`
	Denied      bool                `json:"denied"`
	LastSync    int64               `json:"lastSync"`
	Error       string              `json:"error"`
	Members     []community.Member  `json:"members"`
	Channels    []community.Channel `json:"channels"`
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
	v := Summary{ID: r.ID, Name: r.Name, Description: r.State.Description, Icon: r.State.Icon, Owner: r.Owner, Role: r.State.Role(s.id), Scope: r.Scope, Sequence: r.State.Sequence, Head: r.State.Head, Archived: r.State.Archived, Denied: r.Denied, LastSync: r.LastSync, Error: r.LastError, Members: []community.Member{}, Channels: []community.Channel{}}
	for _, m := range r.State.Members {
		v.Members = append(v.Members, m)
	}
	for _, ch := range r.State.Channels {
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
	for _, c := range events {
		m := c.Operation
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
	if before == 0 {
		pending, e := s.db.CommunityPending(id)
		if e != nil {
			return v, e
		}
		for _, q := range pending {
			if q.Operation.Kind != community.Text || q.Operation.Body.ChannelID != channel {
				continue
			}
			v.Messages = append(v.Messages, MessageView{ID: q.Operation.ID, Author: q.Operation.Author, Name: "Você", Text: q.Operation.Body.Text, Time: q.Operation.Issued, Status: q.Status, Error: q.Error})
		}
	}
	return v, nil
}
