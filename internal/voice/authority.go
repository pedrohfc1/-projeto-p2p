package voice

import (
	"encoding/json"
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pion/webrtc/v4"
	"projetop2p/internal/community"
	"projetop2p/internal/invite"
	"sync"
	"time"
)

const MaxParticipants = 2
const LeaseSeconds = 45

type Participant struct {
	ID      string `json:"id"`
	Session string `json:"session"`
	Name    string `json:"name"`
	Invite  string `json:"invite"`
	UDPPort int    `json:"udpPort"`
}
type Lease struct {
	Domain       string        `json:"domain"`
	Community    string        `json:"community"`
	Channel      string        `json:"channel"`
	Root         string        `json:"root"`
	Owner        string        `json:"owner"`
	Epoch        uint64        `json:"epoch"`
	Generation   string        `json:"generation"`
	Sequence     uint64        `json:"sequence"`
	Expires      int64         `json:"expires"`
	Participants []Participant `json:"participants"`
	Signature    []byte        `json:"signature"`
}

func (l Lease) payload() []byte { l.Signature = nil; b, _ := json.Marshal(l); return b }

type Request struct {
	Method      string                     `json:"method"`
	Community   string                     `json:"community"`
	Channel     string                     `json:"channel"`
	Participant Participant                `json:"participant"`
	Lease       *Lease                     `json:"lease,omitempty"`
	SDP         *webrtc.SessionDescription `json:"sdp,omitempty"`
	Nonce       string                     `json:"nonce,omitempty"`
	Session     string                     `json:"session,omitempty"`
}
type Response struct {
	Lease *Lease                     `json:"lease,omitempty"`
	SDP   *webrtc.SessionDescription `json:"sdp,omitempty"`
	Error string                     `json:"error,omitempty"`
}
type presence struct {
	participant Participant
	expires     time.Time
}
type room struct {
	generation string
	sequence   uint64
	people     map[string]presence
}
type Authority struct {
	mu    sync.Mutex
	key   crypto.PrivKey
	id    string
	state func(string) (community.State, error)
	rooms map[string]*room
}

func NewAuthority(key crypto.PrivKey, state func(string) (community.State, error)) *Authority {
	id, _ := peer.IDFromPrivateKey(key)
	return &Authority{key: key, id: id.String(), state: state, rooms: map[string]*room{}}
}
func permitted(s community.State, channel, id string) bool {
	ch, ok := s.Channels[channel]
	role := s.Role(id)
	return !s.Archived && role != "" && ok && !ch.Deleted && ch.Kind == "voice" && (ch.Role != "moderator" || s.Manages(id))
}
func (a *Authority) Handle(remote string, r Request) Response {
	a.mu.Lock()
	defer a.mu.Unlock()
	fail := func(s string) Response { return Response{Error: s} }
	s, e := a.state(r.Community)
	if e != nil || s.Owner != a.id {
		return fail("owner_unavailable: autoridade da sala indisponivel")
	}
	if !permitted(s, r.Channel, remote) {
		return fail("voice_forbidden: permissao de voz ausente ou revogada")
	}
	now := time.Now()
	// Reclaim empty/expired rooms across communities before applying the room budget.
	for k, candidate := range a.rooms {
		for id, p := range candidate.people {
			if !now.Before(p.expires) {
				delete(candidate.people, id)
			}
		}
		if len(candidate.people) == 0 {
			delete(a.rooms, k)
		}
	}
	key := r.Community + ":" + r.Channel
	roomState := a.rooms[key]
	if roomState == nil {
		if len(a.rooms) >= 64 {
			return fail("voice_resource_limit")
		}
		roomState = &room{generation: community.RandomID(), people: map[string]presence{}}
		a.rooms[key] = roomState
	}
	for id, p := range roomState.people {
		if now.After(p.expires) || !permitted(s, r.Channel, id) {
			delete(roomState.people, id)
		}
	}
	switch r.Method {
	case "leave":
		if p, ok := roomState.people[remote]; ok && p.participant.Session == r.Participant.Session {
			delete(roomState.people, remote)
		}
	case "join", "renew":
		p := r.Participant
		if p.ID != remote || len(p.Session) != 32 || p.UDPPort < 1024 || p.UDPPort > 65535 {
			return fail("invalid participant")
		}
		ticket, e := invite.Parse(p.Invite, now)
		if e != nil || ticket.ID != remote {
			return fail("invalid participant identity route")
		}
		if old, ok := roomState.people[remote]; ok && old.participant.Session != p.Session {
			return fail("another call session already occupies this identity")
		}
		limit := MaxParticipants
		ch := s.Channels[r.Channel]
		if ch.Limit > 0 && ch.Limit < limit {
			limit = ch.Limit
		}
		if _, exists := roomState.people[remote]; !exists && len(roomState.people) >= limit {
			return fail("voice_room_full: sala de voz cheia (v0.6.0: duas pessoas)")
		}
		p.Name = s.Members[remote].Name
		roomState.people[remote] = presence{participant: p, expires: now.Add(LeaseSeconds * time.Second)}
	default:
		return fail("unknown voice authority request")
	}
	roomState.sequence++
	lease := Lease{Domain: "projetop2p/voice-room/0.6.0", Community: s.ID, Channel: r.Channel, Root: s.Root, Owner: s.Owner, Epoch: s.InviteEpoch, Generation: roomState.generation, Sequence: roomState.sequence, Expires: now.Add(LeaseSeconds * time.Second).Unix(), Participants: []Participant{}}
	for _, p := range roomState.people {
		lease.Participants = append(lease.Participants, p.participant)
	}
	lease.Signature, e = a.key.Sign(lease.payload())
	if e != nil {
		return fail("room signature failed")
	}
	return Response{Lease: &lease}
}
func VerifyLease(l Lease, s community.State, channel string, now time.Time) error {
	if l.Domain != "projetop2p/voice-room/0.6.0" || l.Community != s.ID || l.Root != s.Root || l.Owner != s.Owner || l.Epoch != s.InviteEpoch || l.Channel != channel || len(l.Generation) != 32 || l.Sequence == 0 || l.Expires <= now.Unix() || l.Expires > now.Add((LeaseSeconds+5)*time.Second).Unix() || len(l.Participants) > MaxParticipants {
		return errors.New("invalid or expired voice room authorization")
	}
	pub, e := crypto.UnmarshalPublicKey(s.OwnerKey)
	if e != nil {
		return e
	}
	id, e := peer.IDFromPublicKey(pub)
	if e != nil || id.String() != s.Owner {
		return errors.New("owner identity mismatch")
	}
	ok, e := pub.Verify(l.payload(), l.Signature)
	if e != nil || !ok {
		return errors.New("invalid room signature")
	}
	seen := map[string]bool{}
	for _, p := range l.Participants {
		if seen[p.ID] || !permitted(s, channel, p.ID) || p.UDPPort < 1024 || p.UDPPort > 65535 || len(p.Session) != 32 {
			return errors.New("unauthorized room participant")
		}
		ticket, e := invite.Parse(p.Invite, now)
		if e != nil || ticket.ID != p.ID {
			return errors.New("invalid participant route")
		}
		seen[p.ID] = true
	}
	return nil
}
