package communities

import (
	"context"
	"errors"
	"projetop2p/internal/community"
	"projetop2p/internal/invite"
	"projetop2p/internal/localdb"
	"time"
)

type peerRoute struct {
	raw     string
	expires time.Time
}

func (s *Service) selfRoute() (string, error) {
	t, scope := s.environment()
	if t == nil {
		return "", errors.New("connectivity_failure: rede indisponivel")
	}
	settings, e := s.db.Settings()
	if e != nil {
		return "", e
	}
	if scope == "integrated" {
		return invite.NewIntegrated(s.key, settings.Name, t.Addresses(), invitationRoute(t))
	}
	if scope == "auto" {
		return invite.New(s.key, settings.Name, t.Addresses(), nil)
	}
	return invite.NewScoped(s.key, settings.Name, t.Addresses(), scope)
}
func (s *Service) rememberPeer(r localdb.CommunityRecord, raw string) error {
	i, e := invite.Parse(raw, time.Now())
	if e != nil {
		return e
	}
	_, scope := s.environment()
	iscope := i.Scope
	if iscope == "" {
		iscope = "auto"
	}
	if r.State.Role(i.ID) == "" || (scope != iscope && scope != "interface") {
		return errors.New("peer route membership/network mismatch")
	}
	t, _ := s.environment()
	if i.Scope == "integrated" {
		reg, ok := t.(interface{ RegisterInvitationRoute(string, string) error })
		if !ok {
			return errors.New("integrated transport unavailable")
		}
		if e = reg.RegisterInvitationRoute(i.ID, i.Route); e != nil {
			return e
		}
	}
	s.peerMu.Lock()
	defer s.peerMu.Unlock()
	if s.routes[r.ID] == nil {
		s.routes[r.ID] = map[string]peerRoute{}
	}
	s.routes[r.ID][i.ID] = peerRoute{raw: raw, expires: time.Now().Add(10 * time.Minute)}
	return nil
}
func (s *Service) peerRoutes(r localdb.CommunityRecord) map[string]string {
	s.peerMu.Lock()
	defer s.peerMu.Unlock()
	out := map[string]string{}
	for id, p := range s.routes[r.ID] {
		if time.Now().After(p.expires) || r.State.Role(id) == "" {
			delete(s.routes[r.ID], id)
			continue
		}
		out[id] = p.raw
	}
	return out
}
func (s *Service) peerHandle(remote string, req community.Request) community.Response {
	fail := func(e error) community.Response {
		return community.Response{Code: "sync_failure", Error: e.Error(), Commits: []community.Commit{}}
	}
	r, e := s.db.Community(req.CommunityID)
	if e != nil {
		return fail(e)
	}
	if r.State.Sequence == 0 || r.Denied || r.State.Role(s.id) == "" || r.State.Role(remote) == "" {
		return fail(errors.New("current membership required"))
	}
	if req.Method == "peer.directory" {
		if r.Owner != s.id {
			return fail(errors.New("not owner directory"))
		}
		i, e := invite.Parse(req.PeerInvite, time.Now())
		if e != nil || i.ID != remote {
			return fail(errors.New("authenticated peer route required"))
		}
		if e = s.rememberPeer(r, req.PeerInvite); e != nil {
			return fail(e)
		}
		own, e := s.selfRoute()
		if e != nil {
			return fail(e)
		}
		_ = s.rememberPeer(r, own)
		out := community.Response{Commits: []community.Commit{}, PeerInvites: []string{}}
		// Bound the response independently of membership and attacker-controlled route size.
		size := 0
		for _, raw := range s.peerRoutes(r) {
			if size+len(raw) > 160000 {
				break
			}
			out.PeerInvites = append(out.PeerInvites, raw)
			size += len(raw)
		}
		return out
	}
	if req.Method == "peer.text" {
		if req.PeerText == nil || req.PeerText.Operation.Author != remote || req.PeerText.Operation.CommunityID != r.ID {
			return fail(errors.New("authenticated peer author mismatch"))
		}
		if e = s.db.SavePeerText(*req.PeerText, s.id); e != nil {
			return fail(e)
		}
		return community.Response{Ack: req.PeerText.Operation.ID, Commits: []community.Commit{}}
	}
	entries, e := s.db.PeerTexts(r.ID, req.After, 16)
	if e != nil {
		return fail(e)
	}
	res := community.Response{Commits: []community.Commit{}, PeerTexts: []community.PeerTextEntry{}, PeerCursor: req.After}
	for _, entry := range entries {
		res.PeerCursor = entry.Sequence
		if community.AcceptPeerText(entry.Message, r.State, time.Now()) == nil {
			res.PeerTexts = append(res.PeerTexts, entry)
		}
	}
	return res
}
func (s *Service) peerRequest(ctx context.Context, id, raw string, req community.Request) (community.Response, error) {
	t, _ := s.environment()
	if t == nil {
		return community.Response{}, errors.New("network unavailable")
	}
	i, e := invite.Parse(raw, time.Now())
	if e != nil || i.ID != id {
		return community.Response{}, errors.New("invalid peer route")
	}
	return t.CommunityRequest(ctx, id, i.Addresses, req)
}
func (s *Service) postPeerText(ctx context.Context, r localdb.CommunityRecord, body community.Body) (Receipt, error) {
	m, e := community.NewPeerText(s.key, r.State, body)
	if e != nil {
		return Receipt{}, e
	}
	if e = s.db.SavePeerText(m, s.id); e != nil {
		return Receipt{}, e
	}
	receipt := Receipt{ID: m.Operation.ID, CommunityID: r.ID, Status: "salva localmente", Message: "Assinada pelo autor; replicacao entre membros, sem comprovante do proprietario"}
	count := 0
	for id, raw := range s.peerRoutes(r) {
		if id == s.id {
			continue
		}
		if count >= 8 {
			break
		}
		count++
		targetCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
		res, e := s.peerRequest(targetCtx, id, raw, community.Request{Method: "peer.text", CommunityID: r.ID, PeerText: &m})
		cancel()
		if e == nil && res.Error == "" && res.Ack == m.Operation.ID {
			receipt.Status = "replicada"
		}
	}
	return receipt, nil
}
func (s *Service) peerWatch() {
	defer s.wg.Done()
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()
	cursor := 0
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-tick.C:
		}
		records, e := s.db.Communities()
		if e != nil || len(records) == 0 {
			continue
		}
		r := records[cursor%len(records)]
		cursor++
		if r.State.MemberLimit == 0 || r.Denied || r.State.Role(s.id) == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(s.ctx, 6*time.Second)
		_ = s.DiscoverPeers(ctx, r.ID)
		cancel()
		routes := s.peerRoutes(r)
		n := 0
		for id, raw := range routes {
			if id == s.id {
				continue
			}
			if n >= 2 {
				break
			}
			n++
			key := r.ID + ":" + id
			s.peerMu.Lock()
			after := s.cursors[key]
			s.peerMu.Unlock()
			ctx, cancel := context.WithTimeout(s.ctx, 6*time.Second)
			res, e := s.peerRequest(ctx, id, raw, community.Request{Method: "peer.sync", CommunityID: r.ID, After: after})
			cancel()
			if e != nil || res.Error != "" || len(res.PeerTexts) > 16 || res.PeerCursor < after {
				continue
			}
			valid := true
			for _, entry := range res.PeerTexts {
				if entry.Message.Operation.CommunityID != r.ID || entry.Sequence <= after || entry.Sequence > res.PeerCursor {
					valid = false
					break
				}
				if e = s.db.SavePeerText(entry.Message, s.id); e != nil {
					valid = false
					break
				}
			}
			if valid {
				s.peerMu.Lock()
				s.cursors[key] = res.PeerCursor
				s.peerMu.Unlock()
			}
		}
	}
}

// DiscoverPeers exchanges signed endpoint invitations only among current members.
// Failure does not invalidate cached routes or require owner approval for text.
func (s *Service) DiscoverPeers(ctx context.Context, id string) error {
	r, e := s.db.Community(id)
	if e != nil {
		return e
	}
	if r.Denied || r.State.Role(s.id) == "" {
		return errors.New("current membership required")
	}
	own, e := s.selfRoute()
	if e != nil {
		return e
	}
	if r.Owner == s.id {
		return s.rememberPeer(r, own)
	}
	res, e := s.request(ctx, r, community.Request{Method: "peer.directory", CommunityID: id, PeerInvite: own})
	if e != nil {
		return e
	}
	if res.Error != "" {
		return errors.New(res.Error)
	}
	if len(res.PeerInvites) > 100 {
		return errors.New("directory exceeds member limit")
	}
	for _, raw := range res.PeerInvites {
		if e = s.rememberPeer(r, raw); e != nil {
			return e
		}
	}
	return nil
}
