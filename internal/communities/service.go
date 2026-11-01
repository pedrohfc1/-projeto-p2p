// Package communities coordinates signed log replication over authenticated libp2p streams.
package communities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"projetop2p/internal/community"
	"projetop2p/internal/endpoint"
	"projetop2p/internal/localdb"
	"strings"
	"sync"
	"time"
)

type Transport interface {
	SelfID() string
	Addresses() []string
	CommunityRequest(context.Context, string, []string, community.Request) (community.Response, error)
}
type Service struct {
	db        *localdb.DB
	key       crypto.PrivKey
	id        string
	mu        sync.Mutex
	envMu     sync.RWMutex
	transport Transport
	scope     string
	syncSlot  chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}
type Receipt struct {
	ID           string `json:"id"`
	CommunityID  string `json:"communityId"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	RequiredMode string `json:"requiredMode,omitempty"`
}

func New(ctx context.Context, db *localdb.DB, key crypto.PrivKey) *Service {
	id, _ := peer.IDFromPrivateKey(key)
	ctx, cancel := context.WithCancel(ctx)
	s := &Service{db: db, key: key, id: id.String(), scope: "auto", syncSlot: make(chan struct{}, 1), ctx: ctx, cancel: cancel}
	s.wg.Add(1)
	go s.watch()
	return s
}
func (s *Service) Close() { s.cancel(); s.wg.Wait() }
func (s *Service) Attach(t Transport, scope string) {
	s.envMu.Lock()
	s.transport = t
	s.scope = scope
	s.envMu.Unlock()
}
func (s *Service) environment() (Transport, string) {
	s.envMu.RLock()
	defer s.envMu.RUnlock()
	return s.transport, s.scope
}
func (s *Service) watch() {
	defer s.wg.Done()
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()
	cursor := 0
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-tick.C:
			records, e := s.db.Communities()
			if e != nil || len(records) == 0 {
				continue
			}
			for count := 0; count < 2 && count < len(records); count++ {
				r := records[cursor%len(records)]
				cursor++
				if r.Owner == s.id {
					continue
				}
				ctx, cancel := context.WithTimeout(s.ctx, 8*time.Second)
				_ = s.Sync(ctx, r.ID)
				cancel()
			}
		}
	}
}
func (s *Service) Create(name, description, icon string) (string, error) {
	settings, e := s.db.Settings()
	if e != nil {
		return "", e
	}
	o, e := community.SignOperation(s.key, "", "", community.Create, community.Body{Name: strings.TrimSpace(name), Description: description, Icon: icon, Nickname: settings.Name})
	if e != nil {
		return "", e
	}
	c, next, e := community.MakeCommit(s.key, community.State{}, o)
	if e != nil {
		return "", e
	}
	_, scope := s.environment()
	r := localdb.CommunityRecord{ID: o.CommunityID, Owner: s.id, Root: next.Root, Name: next.Name, Scope: scope, Addresses: []string{}}
	s.mu.Lock()
	defer s.mu.Unlock()
	if e = s.db.CreateOwnedCommunity(r, c); e != nil {
		return "", e
	}
	return r.ID, nil
}
func (s *Service) ownerSubmit(remote string, o community.Operation) community.Response {
	s.mu.Lock()
	defer s.mu.Unlock()
	fail := func(code string, e error) community.Response {
		return community.Response{Code: code, Error: e.Error(), Commits: []community.Commit{}}
	}
	if remote != o.Author {
		return fail("forbidden", errors.New("transport identity differs from operation signer"))
	}
	if e := community.VerifyOperation(o); e != nil {
		return fail("invalid", e)
	}
	r, e := s.db.Community(o.CommunityID)
	if e != nil || r.Owner != s.id {
		return fail("not_owner", errors.New("this node is not the pinned community owner"))
	}
	if old, e := s.db.FindCommunityOperation(r.ID, o.ID); e != nil {
		return fail("storage", e)
	} else if old > 0 {
		return community.Response{Ack: o.ID, Sequence: r.State.Sequence, Head: r.State.Head, Commits: []community.Commit{}}
	}
	c, _, e := community.MakeCommit(s.key, r.State, o)
	if e != nil {
		code := "rejected"
		if strings.Contains(e.Error(), "conflict:") {
			code = "conflict"
		}
		return fail(code, e)
	}
	if _, e = s.db.CommunityCommit(c); e != nil {
		return fail("storage", e)
	}
	return community.Response{Ack: o.ID, Sequence: c.Sequence, Head: c.Hash, Commits: []community.Commit{}}
}
func (s *Service) Handle(ctx context.Context, remote string, req community.Request) community.Response {
	errorResponse := func(code, message string) community.Response {
		return community.Response{Error: message, Code: code, Commits: []community.Commit{}}
	}
	if !community.ValidID(req.CommunityID) {
		return errorResponse("invalid", "invalid community ID")
	}
	if req.Method == "submit" {
		if req.Operation == nil || req.Operation.CommunityID != req.CommunityID {
			return errorResponse("invalid", "operation required")
		}
		return s.ownerSubmit(remote, *req.Operation)
	}
	// Snapshot under the service lock; no remote caller can choose its effective role.
	s.mu.Lock()
	defer s.mu.Unlock()
	r, e := s.db.Community(req.CommunityID)
	if e != nil || r.Owner != s.id || r.State.Sequence == 0 {
		return errorResponse("not_owner", "owner log unavailable on this node")
	}
	if r.State.Role(remote) == "" {
		return errorResponse("forbidden", "current membership required; access may have been revoked")
	}
	switch req.Method {
	case "invite":
		if !r.State.Manages(remote) {
			return errorResponse("forbidden", "owner or moderator role required")
		}
		t, scope := s.environment()
		if t == nil {
			return errorResponse("offline", "network unavailable")
		}
		raw, e := community.NewInvite(s.key, r.State, scope, t.Addresses())
		if e != nil {
			return errorResponse("invalid", e.Error())
		}
		return community.Response{Invite: raw, Commits: []community.Commit{}}
	case "sync":
		if req.After > r.State.Sequence {
			return errorResponse("fork", "owner log is behind the replica; rollback/fork must be investigated")
		}
		if req.After == 0 {
			if req.Head != "" {
				return errorResponse("fork", "unexpected genesis cursor")
			}
		} else {
			previous, e := s.db.CommunityEvent(r.ID, req.After)
			if e != nil || previous.Hash != req.Head {
				return errorResponse("fork", "replica prefix differs from owner log")
			}
		}
		events, e := s.db.CommunityEvents(r.ID, req.After, community.BatchSize)
		if e != nil {
			return errorResponse("storage", e.Error())
		}
		response := community.Response{Sequence: r.State.Sequence, Head: r.State.Head, Commits: events}
		for len(response.Commits) > 0 {
			b, _ := json.Marshal(response)
			if len(b)+1 <= community.MaxResponse {
				break
			}
			response.Commits = response.Commits[:len(response.Commits)-1]
		}
		return response
	default:
		return errorResponse("invalid", "unknown community method")
	}
}
func (s *Service) request(ctx context.Context, r localdb.CommunityRecord, req community.Request) (community.Response, error) {
	t, scope := s.environment()
	if t == nil {
		return community.Response{}, errors.New("rede indisponível; operação permanece pendente")
	}
	if r.Scope != "" && r.Scope != scope && scope != "interface" {
		return community.Response{}, fmt.Errorf("ative o modo %s usado pelo proprietário", r.Scope)
	}
	addresses := []string{}
	for _, raw := range r.Addresses {
		a, e := ma.NewMultiaddr(raw)
		if e == nil && endpoint.Allowed(a, scope) {
			addresses = append(addresses, raw)
		}
	}
	if len(addresses) == 0 {
		return community.Response{}, errors.New("sem endereço compatível do proprietário; obtenha convite atualizado")
	}
	return t.CommunityRequest(ctx, r.Owner, addresses, req)
}
func (s *Service) Invite(ctx context.Context, id string) (string, error) {
	r, e := s.db.Community(id)
	if e != nil {
		return "", e
	}
	if !r.State.Manages(s.id) {
		return "", errors.New("somente proprietário/moderador pode criar convites")
	}
	if r.Owner == s.id {
		res := s.Handle(ctx, s.id, community.Request{Method: "invite", CommunityID: id})
		if res.Error != "" {
			return "", errors.New(res.Error)
		}
		return res.Invite, nil
	}
	res, e := s.request(ctx, r, community.Request{Method: "invite", CommunityID: id})
	if e != nil {
		return "", e
	}
	if res.Error != "" {
		return "", errors.New(res.Error)
	}
	ticket, e := community.ParseInvite(res.Invite, time.Now())
	if e != nil || ticket.CommunityID != r.ID || ticket.Owner != r.Owner || ticket.Root != r.Root {
		return "", errors.New("owner returned an invalid invitation")
	}
	return res.Invite, nil
}
func (s *Service) Join(ctx context.Context, raw string) (Receipt, error) {
	ticket, e := community.ParseInvite(strings.TrimSpace(raw), time.Now())
	if e != nil {
		return Receipt{}, e
	}
	_, scope := s.environment()
	result := Receipt{ID: ticket.CommunityID, CommunityID: ticket.CommunityID, Status: "pendente"}
	if scope != ticket.Scope && scope != "interface" {
		result.RequiredMode = ticket.Scope
		result.Message = "Escolha a rede do convite antes de ingressar"
		return result, nil
	}
	old, oldErr := s.db.Community(ticket.CommunityID)
	if oldErr == nil && (old.Owner != ticket.Owner || old.Root != ticket.Root) {
		return result, errors.New("community owner/root does not match pinned identity")
	}
	record := localdb.CommunityRecord{ID: ticket.CommunityID, Owner: ticket.Owner, Root: ticket.Root, Name: ticket.Name, Scope: ticket.Scope, Addresses: ticket.Addresses}
	if ticket.Owner == s.id && oldErr != nil {
		return result, errors.New("owner log missing; restore backup")
	}
	if e = s.db.TrackCommunity(record); e != nil {
		return result, e
	}
	if oldErr == nil && old.State.Role(s.id) != "" && !old.Denied {
		e = s.Sync(ctx, record.ID)
		if e == nil {
			result.Status = "participante"
			return result, nil
		}
		current, _ := s.db.Community(record.ID)
		if !current.Denied {
			result.Message = e.Error()
			return result, nil
		}
		// Cached membership was revoked while offline. A fresh signed ticket may rejoin,
		// but never treat the old cached role as a new authorization.
	}
	if ticket.Owner == s.id {
		return result, errors.New("owner log missing; restore the community backup instead of trusting a new snapshot")
	}
	settings, _ := s.db.Settings()
	base := ""
	if oldErr == nil {
		base = old.State.Head
	}
	op, e := community.SignOperation(s.key, record.ID, base, community.Join, community.Body{Name: settings.Name, Invite: raw})
	if e != nil {
		return result, e
	}
	if e = s.db.QueueCommunity(op); e != nil {
		return result, e
	}
	if e = s.Sync(ctx, record.ID); e != nil {
		result.Message = e.Error()
		pending, _ := s.db.CommunityPending(record.ID)
		for _, q := range pending {
			if q.Operation.ID == op.ID && q.Status == "falhou" {
				result.Status = "falhou"
				result.Message = q.Error
			}
		}
		return result, nil
	}
	latest, _ := s.db.Community(record.ID)
	if latest.State.Role(s.id) != "" {
		result.Status = "participante"
		return result, nil
	}
	result.Message = "Ingresso aguardando confirmação assinada do proprietário"
	return result, nil
}
func (s *Service) Action(ctx context.Context, id, kind string, body community.Body) (Receipt, error) {
	r, e := s.db.Community(id)
	if e != nil {
		return Receipt{}, e
	}
	role := r.State.Role(s.id)
	if role == "" || r.Denied {
		return Receipt{}, errors.New("current membership required")
	}
	switch kind {
	case community.Text:
		if strings.TrimSpace(body.Text) == "" {
			return Receipt{}, errors.New("mensagem vazia")
		}
	case community.ChannelCreate, community.ChannelDelete, community.Remove:
		if !r.State.Manages(s.id) {
			return Receipt{}, errors.New("moderator or owner role required")
		}
		if kind == community.ChannelCreate {
			body.ChannelID = community.RandomID()
		}
	case community.SetRole, community.Metadata, community.Archive:
		if role != "owner" {
			return Receipt{}, errors.New("owner role required")
		}
	case community.Leave:
	default:
		return Receipt{}, errors.New("unsupported community action")
	}
	if kind == community.Text {
		ch, ok := r.State.Channels[body.ChannelID]
		if !ok || ch.Deleted || ch.Kind != "text" || r.State.Archived {
			return Receipt{}, errors.New("channel unavailable for text")
		}
	}
	if kind == community.Remove && role == "moderator" && r.State.Members[body.Target].Role != "member" {
		return Receipt{}, errors.New("moderators may only remove ordinary members")
	}
	op, e := community.SignOperation(s.key, id, r.State.Head, kind, body)
	if e != nil {
		return Receipt{}, e
	}
	receipt := Receipt{ID: op.ID, CommunityID: id, Status: "pendente"}
	if r.Owner == s.id {
		response := s.ownerSubmit(s.id, op)
		if response.Error != "" {
			return receipt, errors.New(response.Error)
		}
		receipt.Status = "confirmada"
		return receipt, nil
	}
	if e = s.db.QueueCommunity(op); e != nil {
		return receipt, e
	}
	if e = s.Sync(ctx, id); e != nil {
		receipt.Message = e.Error()
		return receipt, nil
	}
	if n, _ := s.db.FindCommunityOperation(id, op.ID); n > 0 {
		receipt.Status = "confirmada"
		return receipt, nil
	}
	pending, _ := s.db.CommunityPending(id)
	for _, q := range pending {
		if q.Operation.ID == op.ID && q.Status == "falhou" {
			receipt.Status = q.Status
			receipt.Message = q.Error
			if kind != community.Text {
				_ = s.db.CommunityPendingStatus(id, op.ID, "delete", "")
			}
		}
	}
	return receipt, nil
}
func (s *Service) flush(ctx context.Context, r localdb.CommunityRecord, joinsOnly bool) error {
	pending, e := s.db.CommunityPending(r.ID)
	if e != nil {
		return e
	}
	count := 0
	for _, q := range pending {
		if q.Status != "pendente" || (q.Operation.Kind == community.Join) != joinsOnly {
			continue
		}
		if count >= 8 {
			break
		}
		count++
		res, e := s.request(ctx, r, community.Request{Method: "submit", CommunityID: r.ID, Operation: &q.Operation})
		if e != nil {
			return e
		}
		if res.Error != "" {
			if e = s.db.CommunityPendingStatus(r.ID, q.Operation.ID, "falhou", res.Error); e != nil {
				return e
			}
			continue
		}
		if res.Ack != q.Operation.ID {
			return errors.New("owner acknowledgement does not match pending operation")
		}
		if e = s.db.CommunityPendingStatus(r.ID, q.Operation.ID, "aguardando sincronização", ""); e != nil {
			return e
		}
	}
	return nil
}
func (s *Service) read(ctx context.Context, id string) error {
	for page := 0; page < 8; page++ {
		r, e := s.db.Community(id)
		if e != nil {
			return e
		}
		res, e := s.request(ctx, r, community.Request{Method: "sync", CommunityID: id, After: r.State.Sequence, Head: r.State.Head})
		if e != nil {
			return e
		}
		if res.Error != "" {
			_ = s.db.CommunityStatus(id, res.Error, res.Code == "forbidden")
			return errors.New(res.Error)
		}
		if len(res.Commits) > community.BatchSize {
			return errors.New("oversized sync batch")
		}
		for _, c := range res.Commits {
			if c.CommunityID != id {
				return errors.New("response contains another community")
			}
			s.mu.Lock()
			_, e = s.db.CommunityCommit(c)
			s.mu.Unlock()
			if e != nil {
				return e
			}
		}
		current, e := s.db.Community(id)
		if e != nil {
			return e
		}
		if current.State.Sequence == res.Sequence {
			if current.State.Head != res.Head {
				return errors.New("owner head differs from verified replica")
			}
			return s.db.CommunityStatus(id, "", false)
		}
		if current.State.Sequence > res.Sequence || len(res.Commits) == 0 {
			return errors.New("owner supplied an inconsistent sync cursor")
		}
	}
	return nil // Partial progress is retained; background sync continues from the verified cursor.
}
func (s *Service) Sync(ctx context.Context, id string) error {
	select {
	case s.syncSlot <- struct{}{}:
		defer func() { <-s.syncSlot }()
	case <-ctx.Done():
		return ctx.Err()
	}
	r, e := s.db.Community(id)
	if e != nil {
		return e
	}
	if r.Owner == s.id {
		return nil
	}
	fail := func(e error) error {
		if e != nil {
			latest, _ := s.db.Community(id)
			_ = s.db.CommunityStatus(id, e.Error(), latest.Denied)
		}
		return e
	}
	if e = s.flush(ctx, r, true); e != nil {
		return fail(e)
	}
	if e = s.read(ctx, id); e != nil {
		return fail(e)
	}
	r, e = s.db.Community(id)
	if e != nil {
		return e
	}
	if e = s.flush(ctx, r, false); e != nil {
		return fail(e)
	}
	return fail(s.read(ctx, id))
}
func (s *Service) Discard(id, op string) error {
	return s.db.CommunityPendingStatus(id, op, "delete", "")
}
