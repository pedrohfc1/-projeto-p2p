package voice

import (
	"bytes"
	"context"
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"net"
	"projetop2p/internal/community"
	"projetop2p/internal/invite"
	"sync/atomic"
	"testing"
	"time"
)

type hotfixFixture struct {
	owner     crypto.PrivKey
	state     community.State
	people    []Participant
	authority *Authority
}

func hotfixRoom(t *testing.T) hotfixFixture {
	t.Helper()
	owner := testKey(t)
	op, _ := community.SignOperation(owner, "", "", community.Create, community.Body{Name: "recovery"})
	_, state, e := community.MakeCommit(owner, community.State{}, op)
	if e != nil {
		t.Fatal(e)
	}
	keys := []crypto.PrivKey{owner, testKey(t), testKey(t)}
	f := hotfixFixture{owner: owner}
	for i, key := range keys {
		id, _ := peer.IDFromPrivateKey(key)
		if i > 0 {
			ticket, e := community.NewInvite(owner, state, "interface", []string{"/ip4/127.0.0.1/tcp/4001/p2p/" + state.Owner})
			if e != nil {
				t.Fatal(e)
			}
			op, _ = community.SignOperation(key, state.ID, state.Head, community.Join, community.Body{Name: "member", Invite: ticket})
			_, state, e = community.MakeCommit(owner, state, op)
			if e != nil {
				t.Fatal(e)
			}
		}
		raw, e := invite.NewScoped(key, "fixture", []string{"/ip4/127.0.0.1/tcp/4001/p2p/" + id.String()}, "interface")
		if e != nil {
			t.Fatal(e)
		}
		f.people = append(f.people, Participant{ID: id.String(), Session: community.RandomID(), Invite: raw, UDPPort: 4002})
	}
	f.state = state
	f.authority = NewAuthority(owner, func(string) (community.State, error) { return state, nil })
	return f
}
func (f hotfixFixture) join(t *testing.T, p Participant) Lease {
	t.Helper()
	r := f.authority.Handle(p.ID, Request{Method: "join", Community: f.state.ID, Channel: "voz", Participant: p})
	if r.Error != "" || r.Lease == nil {
		t.Fatal(r.Error)
	}
	return *r.Lease
}

type renewTransport struct {
	f           hotfixFixture
	called      chan struct{}
	syncStarted *atomic.Bool
}

func (r renewTransport) VoiceRequest(ctx context.Context, id string, a []string, q Request) (Response, error) {
	if r.syncStarted.Load() {
		return Response{}, errors.New("sync ran before renewal")
	}
	if ctx.Err() != nil {
		return Response{}, ctx.Err()
	}
	out := r.f.authority.Handle(q.Participant.ID, q)
	select {
	case r.called <- struct{}{}:
	default:
	}
	return out, nil
}
func TestRenewalPrecedesBlockedCommunitySync(t *testing.T) {
	f := hotfixRoom(t)
	p := f.people[1]
	lease := f.join(t, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var started atomic.Bool
	called := make(chan struct{}, 1)
	c := &call{ctx: ctx, cancel: cancel, group: f.state.ID, channel: "voz", session: p.Session, owner: f.state.Owner, lease: lease, peers: map[string]*roomPeer{}}
	s := &Service{ctx: ctx, current: c, id: p.ID, options: Options{State: func(string) (community.State, error) { return f.state, nil }, Addresses: func(string) []string { return nil }, SelfInvite: func() (string, error) { return p.Invite, nil }, Sync: func(ctx context.Context, _ string) error { started.Store(true); <-ctx.Done(); return ctx.Err() }}}
	engine, err := NewEngine("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	s.engine = engine
	s.transport = renewTransport{f, called, &started}
	s.wg.Add(1)
	go s.watch()
	select {
	case <-called:
	case <-time.After(5 * time.Second):
		cancel()
		s.wg.Wait()
		t.Fatal("renewal starved behind sync")
	}
	cancel()
	s.wg.Wait()
	if c.lease.Expires < lease.Expires {
		t.Fatal("authorization went backwards")
	}
}
func TestInvalidAuthorizationPausesWithoutCorruptingSignedLease(t *testing.T) {
	f := hotfixRoom(t)
	p := f.people[1]
	lease := f.join(t, p)
	original := append([]byte(nil), lease.payload()...)
	changed := f.state
	changed.InviteEpoch++
	ctx, cancel := context.WithCancel(context.Background())
	c := &call{ctx: ctx, cancel: cancel, group: f.state.ID, channel: "voz", session: p.Session, lease: lease, peers: map[string]*roomPeer{}}
	s := &Service{ctx: ctx, current: c, id: p.ID, options: Options{State: func(string) (community.State, error) { return changed, nil }}}
	s.wg.Add(1)
	go s.guard()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		paused := c.authorizationInvalid
		s.mu.Unlock()
		if paused {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	s.wg.Wait()
	if !c.authorizationInvalid || s.current != c {
		t.Fatal("media and participation not separated")
	}
	if !bytes.Equal(original, c.lease.payload()) || VerifyLease(c.lease, f.state, "voz", time.Now()) != nil {
		t.Fatal("guard mutated signed capability")
	}
}
func TestNewSignedLeaseRecoversExpiredLocalAuthorizationAndRejectsReplay(t *testing.T) {
	f := hotfixRoom(t)
	a, b := f.people[1], f.people[2]
	if a.ID < b.ID {
		a, b = b, a
	}
	f.join(t, a)
	fresh := f.join(t, b)
	expired := fresh
	expired.Expires = time.Now().Add(-time.Second).Unix()
	expired.Signature, _ = f.owner.Sign(expired.payload())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &call{ctx: ctx, cancel: cancel, group: f.state.ID, channel: "voz", session: a.Session, lease: expired, authorizationInvalid: true, peers: map[string]*roomPeer{}}
	s := &Service{ctx: ctx, current: c, id: a.ID, options: Options{State: func(string) (community.State, error) { return f.state, nil }}}
	left, right := net.Pipe()
	defer right.Close()
	r := Request{Method: "media", Community: f.state.ID, Channel: "voz", Session: a.Session, Participant: b, Nonce: community.RandomID(), Lease: &fresh}
	if err := s.AcceptMedia(b.ID, r, left); err != nil {
		left.Close()
		t.Fatal(err)
	}
	c.peers[b.ID].stream.Close()
	if c.authorizationInvalid || VerifyLease(c.lease, f.state, "voz", time.Now()) != nil {
		t.Fatal("new owner signature was not adopted")
	}
	l2, r2 := net.Pipe()
	defer l2.Close()
	defer r2.Close()
	if s.AcceptMedia(b.ID, r, l2) == nil {
		t.Fatal("replay accepted")
	}
	r.Nonce = community.RandomID()
	r.Lease = &expired
	if s.AcceptMedia(b.ID, r, l2) == nil {
		t.Fatal("expired authorization accepted")
	}
}
func TestAuthenticatedNewJoinSupersedesCrashedSessionButOldRenewCannotReclaim(t *testing.T) {
	f := hotfixRoom(t)
	p := f.people[1]
	f.join(t, p)
	next := p
	next.Session = community.RandomID()
	fresh := f.join(t, next)
	if fresh.Participants[0].Session != next.Session {
		t.Fatal("stale session kept slot")
	}
	r := f.authority.Handle(p.ID, Request{Method: "renew", Community: f.state.ID, Channel: "voz", Participant: p})
	if r.Error == "" {
		t.Fatal("old session reclaimed replacement")
	}
}

type blockedMediaTransport struct {
	calls   atomic.Int32
	started chan struct{}
	conn    MediaConn
}

func (t *blockedMediaTransport) VoiceRequest(context.Context, string, []string, Request) (Response, error) {
	return Response{}, errors.New("unused")
}
func (t *blockedMediaTransport) OpenVoiceMedia(ctx context.Context, _ string, _ []string, _ Request) (MediaConn, error) {
	t.calls.Add(1)
	select {
	case t.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return t.conn, nil
}
func TestUpdatedRosterRoutesReplaceOldPeerAndCanceledAttemptsCannotAttach(t *testing.T) {
	f := hotfixRoom(t)
	a, b := f.people[1], f.people[2]
	if a.ID > b.ID {
		a, b = b, a
	}
	f.join(t, a)
	lease := f.join(t, b)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &call{ctx: ctx, cancel: cancel, group: f.state.ID, channel: "voz", session: a.Session, lease: lease, peers: map[string]*roomPeer{b.ID: {participant: Participant{ID: b.ID, Session: b.Session, Invite: "obsolete"}, slot: 0}}}
	s := &Service{ctx: ctx, current: c, id: a.ID}
	left, right := net.Pipe()
	defer right.Close()
	transport := &blockedMediaTransport{started: make(chan struct{}, 2), conn: left}
	s.reconcileMesh(c, lease, transport)
	select {
	case <-transport.started:
	case <-time.After(time.Second):
		t.Fatal("missing attempt")
	}
	s.reconcileMesh(c, lease, transport)
	s.mu.Lock()
	route := c.peers[b.ID].participant.Invite
	s.mu.Unlock()
	if route != b.Invite {
		t.Fatal("stale invite retained")
	}
	next := lease
	next.Sequence++
	next.Participants = []Participant{a}
	next.Signature, _ = f.owner.Sign(next.payload())
	s.mu.Lock()
	c.lease = next
	s.mu.Unlock()
	s.reconcileMesh(c, next, transport)
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("canceled attempt blocked recovery")
	}
	if transport.calls.Load() != 1 {
		t.Fatal("duplicate parallel attempts")
	}
	right.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
	if _, e := right.Write([]byte{1}); e == nil {
		t.Fatal("obsolete stream was attached")
	}
}

func TestDelayedRenewalCannotRollbackLeaseReceivedWhileSyncWasBlocked(t *testing.T) {
	f := hotfixRoom(t)
	p := f.people[1]
	initial := f.join(t, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var started atomic.Bool
	called := make(chan struct{}, 1)
	entered := make(chan struct{})
	release := make(chan struct{})
	exited := make(chan struct{})
	c := &call{ctx: ctx, cancel: cancel, group: f.state.ID, channel: "voz", session: p.Session, owner: f.state.Owner, lease: initial, peers: map[string]*roomPeer{}}
	engine, e := NewEngine("127.0.0.1")
	if e != nil {
		t.Fatal(e)
	}
	defer engine.Close()
	s := &Service{ctx: ctx, current: c, id: p.ID, engine: engine, options: Options{State: func(string) (community.State, error) { return f.state, nil }, Addresses: func(string) []string { return nil }, SelfInvite: func() (string, error) { return p.Invite, nil }, Sync: func(ctx context.Context, _ string) error {
		started.Store(true)
		close(entered)
		select {
		case <-release:
		case <-ctx.Done():
		}
		close(exited)
		return nil
	}}}
	s.transport = renewTransport{f, called, &started}
	s.wg.Add(1)
	go s.watch()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		cancel()
		s.wg.Wait()
		t.Fatal("renewal did not reach sync")
	}
	newer := f.join(t, p)
	s.mu.Lock()
	c.lease = newer
	s.mu.Unlock()
	close(release)
	<-exited
	cancel()
	s.wg.Wait()
	if c.lease.Sequence != newer.Sequence || c.lease.Generation != newer.Generation {
		t.Fatal("late renewal replaced newer signed capability")
	}
}

func TestScreenAuthorizationLookupDoesNotBlockAudioLock(t *testing.T) {
	f := hotfixRoom(t)
	p := f.people[1]
	lease := f.join(t, p)
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	svc := &Service{id: p.ID, options: Options{State: func(string) (community.State, error) { close(entered); <-release; return f.state, nil }}}
	sc := &screenCall{parent: &call{lease: lease, group: f.state.ID, channel: "voz"}}
	go func() {
		defer close(done)
		_, ok := svc.validateScreenLease(sc)
		if !ok {
			t.Error("valid grant rejected")
		}
	}()
	<-entered
	acquired := make(chan struct{})
	go func() { svc.mu.Lock(); svc.mu.Unlock(); close(acquired) }()
	select {
	case <-acquired:
	case <-time.After(200 * time.Millisecond):
		close(release)
		<-done
		t.Fatal("screen DB lookup blocked audio/media mutex")
	}
	close(release)
	<-done
}
