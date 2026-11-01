// Package community defines signed, owner-ordered community events. It does not encrypt content.
package community

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	MaxText       = 4096
	MaxMembers    = 256
	MaxChannels   = 256
	MaxInviteUses = 4096
	BatchSize     = 16
	MaxRequest    = 32768
	MaxResponse   = 262144
)
const (
	Create        = "community.create"
	Metadata      = "community.update"
	Archive       = "community.archive"
	ChannelCreate = "channel.create"
	ChannelDelete = "channel.delete"
	ChannelUpdate = "channel.update"
	InviteRevoke  = "invite.revoke"
	Join          = "member.join"
	SetRole       = "member.role"
	Remove        = "member.remove"
	Leave         = "member.leave"
	Text          = "message.text"
)

type Body struct {
	Limit       int    `json:"limit,omitempty"`
	Name        string `json:"name,omitempty"`
	Nickname    string `json:"nickname,omitempty"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	ChannelID   string `json:"channelId,omitempty"`
	ChannelKind string `json:"channelKind,omitempty"`
	Target      string `json:"target,omitempty"`
	Role        string `json:"role,omitempty"`
	Text        string `json:"text,omitempty"`
	Invite      string `json:"invite,omitempty"`
}
type Operation struct {
	Domain      string `json:"domain"`
	ID          string `json:"id"`
	CommunityID string `json:"communityId"`
	Nonce       string `json:"nonce"`
	Author      string `json:"author"`
	PublicKey   []byte `json:"publicKey"`
	Issued      int64  `json:"issued"`
	Base        string `json:"base"`
	Kind        string `json:"kind"`
	Body        Body   `json:"body"`
	Signature   []byte `json:"signature"`
}
type Commit struct {
	Domain      string    `json:"domain"`
	CommunityID string    `json:"communityId"`
	Sequence    uint64    `json:"sequence"`
	Previous    string    `json:"previous"`
	Hash        string    `json:"hash"`
	Owner       string    `json:"owner"`
	OwnerKey    []byte    `json:"ownerKey"`
	At          int64     `json:"at"`
	Operation   Operation `json:"operation"`
	Signature   []byte    `json:"signature"`
}
type Member struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	Removed bool   `json:"removed"`
}
type Channel struct {
	Role    string `json:"role,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Deleted bool   `json:"deleted"`
}
type State struct {
	MemberLimit    int                `json:"memberLimit,omitempty"`
	InviteCounts   map[string]uint32  `json:"inviteCounts,omitempty"`
	RevokedInvites map[string]bool    `json:"revokedInvites,omitempty"`
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	Icon           string             `json:"icon"`
	Owner          string             `json:"owner"`
	OwnerKey       []byte             `json:"ownerKey"`
	Root           string             `json:"root"`
	Sequence       uint64             `json:"sequence"`
	Head           string             `json:"head"`
	InviteEpoch    uint64             `json:"inviteEpoch"`
	Archived       bool               `json:"archived"`
	Members        map[string]Member  `json:"members"`
	Channels       map[string]Channel `json:"channels"`
	InviteUses     map[string]string  `json:"inviteUses"`
}

func RandomID() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func hash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func ValidID(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 32 && s == strings.ToLower(s)
}
func ChannelID(s string) bool {
	if s == "geral" || s == "voz" {
		return true
	}
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 16 && s == strings.ToLower(s)
}
func SafeText(s string, max int, multiline bool) bool {
	if !utf8.ValidString(s) || len(s) > max {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) && !(multiline && (r == '\n' || r == '\t')) {
			return false
		}
	}
	return true
}
func NameOK(s string) bool          { return strings.TrimSpace(s) != "" && SafeText(s, 80, false) }
func (o Operation) payload() []byte { o.ID = ""; o.Signature = nil; b, _ := json.Marshal(o); return b }
func (c Commit) payload() []byte    { c.Hash = ""; c.Signature = nil; b, _ := json.Marshal(c); return b }
func signedBy(id string, key, payload, signature []byte) error {
	if len(key) > 128 || len(signature) != 64 {
		return errors.New("invalid Ed25519 key/signature size")
	}
	pub, e := crypto.UnmarshalPublicKey(key)
	if e != nil || pub.Type() != crypto.Ed25519 {
		return errors.New("Ed25519 key required")
	}
	pid, e := peer.IDFromPublicKey(pub)
	if e != nil || pid.String() != id {
		return errors.New("public key does not match Peer ID")
	}
	ok, e := pub.Verify(payload, signature)
	if e != nil || !ok {
		return errors.New("invalid signature")
	}
	return nil
}
func SignOperation(key crypto.PrivKey, id, base, kind string, body Body) (Operation, error) {
	if !bodyShape(kind, body) {
		return Operation{}, errors.New("unexpected fields for operation kind")
	}
	if e := fieldBounds(body, base); e != nil {
		return Operation{}, e
	}
	pid, e := peer.IDFromPrivateKey(key)
	if e != nil {
		return Operation{}, e
	}
	pub, e := crypto.MarshalPublicKey(key.GetPublic())
	if e != nil {
		return Operation{}, e
	}
	o := Operation{Domain: "projetop2p/community-operation/1", CommunityID: id, Nonce: RandomID(), Author: pid.String(), PublicKey: pub, Issued: time.Now().Unix(), Base: base, Kind: kind, Body: body}
	if kind == Create {
		o.CommunityID = hash([]byte(o.Author + ":" + o.Nonce))
	}
	o.ID = hash(o.payload())
	o.Signature, e = key.Sign(o.payload())
	if e != nil {
		return o, e
	}
	return o, VerifyOperation(o)
}
func VerifyOperation(o Operation) error {
	if e := fieldBounds(o.Body, o.Base); e != nil {
		return e
	}
	if len(o.PublicKey) > 128 || len(o.Signature) != 64 || len(o.Author) > 128 || len(o.Kind) > 32 || len(o.Domain) > 64 || len(o.Nonce) > 32 || len(o.ID) > 64 || len(o.CommunityID) > 64 {
		return errors.New("operation envelope exceeds limit")
	}
	nonce, e := hex.DecodeString(o.Nonce)
	if o.Domain != "projetop2p/community-operation/1" || !ValidID(o.CommunityID) || o.ID != hash(o.payload()) || e != nil || len(nonce) != 16 || o.Issued < 1 {
		return errors.New("invalid operation envelope")
	}
	if !SafeText(o.Body.Text, MaxText, true) || !SafeText(o.Body.Description, 1024, true) || !SafeText(o.Body.Name, 80, false) || !SafeText(o.Body.Icon, 32, false) || len(o.Body.Invite) > 16000 || len(o.Body.ChannelID) > 64 || !SafeText(o.Body.Nickname, 80, false) || len(o.Body.Target) > 128 || len(o.Body.Role) > 16 || len(o.Body.ChannelKind) > 8 || len(o.Base) > 64 {
		return errors.New("invalid operation fields or size")
	}
	switch o.Kind {
	case Create, Metadata, Archive, ChannelCreate, ChannelDelete, ChannelUpdate, InviteRevoke, Join, SetRole, Remove, Leave, Text:
	default:
		return errors.New("unknown operation")
	}
	if !bodyShape(o.Kind, o.Body) {
		return errors.New("unexpected fields for operation kind")
	}
	return signedBy(o.Author, o.PublicKey, o.payload(), o.Signature)
}
func (s State) Role(id string) string {
	m, ok := s.Members[id]
	if !ok || m.Removed {
		return ""
	}
	return m.Role
}
func (s State) Manages(id string) bool { r := s.Role(id); return r == "owner" || r == "moderator" }
func copyState(s State) State {
	t := s
	t.Members = map[string]Member{}
	for k, v := range s.Members {
		t.Members[k] = v
	}
	t.Channels = map[string]Channel{}
	for k, v := range s.Channels {
		t.Channels[k] = v
	}
	t.InviteUses = map[string]string{}
	for k, v := range s.InviteUses {
		t.InviteUses[k] = v
	}
	t.InviteCounts = map[string]uint32{}
	for k, v := range s.InviteCounts {
		t.InviteCounts[k] = v
	}
	t.RevokedInvites = map[string]bool{}
	for k, v := range s.RevokedInvites {
		t.RevokedInvites[k] = v
	}
	return t
}
func MakeCommit(key crypto.PrivKey, s State, o Operation) (Commit, State, error) {
	if e := VerifyOperation(o); e != nil {
		return Commit{}, s, e
	}
	now := time.Now()
	if o.Issued > now.Add(5*time.Minute).Unix() || o.Issued < now.Add(-30*24*time.Hour).Unix() {
		return Commit{}, s, errors.New("operation timestamp outside accepted window")
	}
	id, _ := peer.IDFromPrivateKey(key)
	pub, _ := crypto.MarshalPublicKey(key.GetPublic())
	c := Commit{Domain: "projetop2p/community-commit/1", CommunityID: o.CommunityID, Sequence: s.Sequence + 1, Previous: s.Head, Owner: id.String(), OwnerKey: pub, At: now.Unix(), Operation: o}
	c.Hash = hash(c.payload())
	var e error
	c.Signature, e = key.Sign(c.payload())
	if e != nil {
		return c, s, e
	}
	next, e := Apply(s, c)
	return c, next, e
}
func VerifyCommit(c Commit) error {
	if c.Domain != "projetop2p/community-commit/1" || c.Hash != hash(c.payload()) || c.CommunityID != c.Operation.CommunityID || c.Sequence < 1 || c.At < 1 {
		return errors.New("invalid commit envelope")
	}
	if e := VerifyOperation(c.Operation); e != nil {
		return e
	}
	return signedBy(c.Owner, c.OwnerKey, c.payload(), c.Signature)
}
func Apply(s State, c Commit) (State, error) {
	if e := VerifyCommit(c); e != nil {
		return s, e
	}
	if c.Sequence != s.Sequence+1 || c.Previous != s.Head {
		return s, errors.New("log gap, duplicate or fork detected")
	}
	o := c.Operation
	b := o.Body
	t := copyState(s)
	if s.Sequence == 0 {
		if o.Kind != Create || o.Author != c.Owner || o.CommunityID != hash([]byte(o.Author+":"+o.Nonce)) || o.Base != "" || !NameOK(b.Name) {
			return s, errors.New("invalid community genesis")
		}
		t = State{MemberLimit: b.Limit, ID: o.CommunityID, Name: b.Name, Description: b.Description, Icon: b.Icon, Owner: c.Owner, OwnerKey: c.OwnerKey, Root: c.Hash, InviteEpoch: 1, Members: map[string]Member{o.Author: {ID: o.Author, Name: o.Body.Nickname, Role: "owner"}}, Channels: map[string]Channel{"geral": {ID: "geral", Name: "geral", Kind: "text"}, "voz": {ID: "voz", Name: "Sala de voz", Kind: "voice"}}, InviteUses: map[string]string{}}
	} else {
		if s.ID != o.CommunityID || s.Owner != c.Owner || string(s.OwnerKey) != string(c.OwnerKey) || s.Archived {
			return s, errors.New("wrong owner/community or archived community")
		}
		role := s.Role(o.Author)
		if o.Kind != Text && o.Kind != Join && o.Base != s.Head {
			return s, errors.New("conflict: synchronize and retry the control operation")
		}
		switch o.Kind {
		case Metadata:
			if role != "owner" || !NameOK(b.Name) {
				return s, errors.New("only owner may edit community")
			}
			t.Name = b.Name
			t.Description = b.Description
			t.Icon = b.Icon
		case Archive:
			if role != "owner" {
				return s, errors.New("only owner may archive community")
			}
			t.Archived = true
		case ChannelCreate:
			if b.Role != "" && b.Role != "member" && b.Role != "moderator" {
				return s, errors.New("invalid voice role")
			}
			if b.ChannelKind == "voice" && b.Limit > 6 || b.ChannelKind == "text" && b.Limit != 0 {
				return s, errors.New("invalid channel participant limit")
			}
			if s.MemberLimit > 0 {
				count := 0
				for _, ch := range s.Channels {
					if !ch.Deleted && ch.Kind == b.ChannelKind {
						count++
					}
				}
				limit := 50
				if b.ChannelKind == "voice" {
					limit = 20
				}
				if count >= limit {
					return s, errors.New("channel_limit: limite de canais atingido")
				}
			}
			if !s.Manages(o.Author) || !ChannelID(b.ChannelID) || !NameOK(b.Name) || (b.ChannelKind != "text" && b.ChannelKind != "voice") || len(s.Channels) >= MaxChannels {
				return s, errors.New("channel creation not permitted or limit reached")
			}
			if _, ok := s.Channels[b.ChannelID]; ok {
				return s, errors.New("channel ID already exists")
			}
			t.Channels[b.ChannelID] = Channel{ID: b.ChannelID, Name: b.Name, Kind: b.ChannelKind, Limit: b.Limit, Role: b.Role}
		case InviteRevoke:
			if !s.Manages(o.Author) || len(b.Invite) != 32 {
				return s, errors.New("invite revocation not permitted")
			}
			if len(t.RevokedInvites) >= MaxInviteUses && !t.RevokedInvites[b.Invite] {
				return s, errors.New("revocation registry full")
			}
			t.RevokedInvites[b.Invite] = true
		case ChannelUpdate:
			ch, ok := s.Channels[b.ChannelID]
			if !s.Manages(o.Author) || !ok || ch.Deleted || !NameOK(b.Name) {
				return s, errors.New("channel update not permitted")
			}
			ch.Name = b.Name
			if ch.Kind == "voice" && b.Limit > 6 || ch.Kind == "text" && b.Limit != 0 {
				return s, errors.New("invalid channel limit")
			}
			if b.Role != "" && b.Role != "member" && b.Role != "moderator" {
				return s, errors.New("invalid channel role")
			}
			ch.Role = b.Role
			ch.Limit = b.Limit
			t.Channels[ch.ID] = ch
		case ChannelDelete:
			ch, ok := s.Channels[b.ChannelID]
			if !s.Manages(o.Author) || !ok || ch.Deleted {
				return s, errors.New("channel deletion not permitted")
			}
			ch.Deleted = true
			t.Channels[ch.ID] = ch
		case Join:
			ticket, e := ParseInvite(b.Invite, time.Unix(c.At, 0))
			if e != nil {
				return s, e
			}
			if role != "" || ticket.CommunityID != s.ID || ticket.Owner != s.Owner || ticket.Root != s.Root || !NameOK(b.Name) {
				return s, errors.New("join invitation is stale or inconsistent")
			}
			if ticket.Epoch != s.InviteEpoch || s.RevokedInvites[ticket.Nonce] {
				return s, errors.New("invite_revoked: convite revogado")
			}
			used := s.InviteCounts[ticket.Nonce]
			if used == 0 && s.InviteUses[ticket.Nonce] != "" {
				used = 1
			}
			limit := ticket.MaxUses
			if ticket.Policy == "" || ticket.Policy == "single" {
				limit = 1
			}
			if limit > 0 && used >= limit {
				if limit == 1 {
					return s, errors.New("invite_used: convite utilizado")
				}
				return s, errors.New("invite_exhausted: limite de utilizacoes atingido")
			}
			if used == 0 && len(s.InviteUses) >= MaxInviteUses {
				return s, errors.New("invite_registry_full: limite de registros de convites")
			}
			t.InviteCounts[ticket.Nonce] = used + 1
			limitMembers := MaxMembers
			if s.MemberLimit > 0 {
				limitMembers = s.MemberLimit
			}
			if _, exists := s.Members[o.Author]; !exists && len(s.Members) >= limitMembers {
				return s, errors.New("community_full: comunidade cheia")
			}
			t.InviteUses[ticket.Nonce] = o.Author
			t.Members[o.Author] = Member{ID: o.Author, Name: b.Name, Role: "member"}
		case SetRole:
			m, ok := s.Members[b.Target]
			if role != "owner" || !ok || m.Removed || m.Role == "owner" || (b.Role != "moderator" && b.Role != "member") {
				return s, errors.New("only owner may grant/revoke moderator; ownership is immutable in v0.5")
			}
			m.Role = b.Role
			t.Members[m.ID] = m
			t.InviteEpoch++
		case Remove, Leave:
			target := b.Target
			if o.Kind == Leave {
				target = o.Author
			}
			m, ok := s.Members[target]
			allowed := role == "owner" || (role == "moderator" && m.Role == "member") || (o.Kind == Leave && target == o.Author && role != "")
			if !allowed || !ok || m.Removed || m.Role == "owner" {
				return s, errors.New("member removal not permitted")
			}
			m.Removed = true
			t.Members[m.ID] = m
			t.InviteEpoch++
		case Text:
			ch, ok := s.Channels[b.ChannelID]
			if role == "" || !ok || ch.Deleted || ch.Kind != "text" || strings.TrimSpace(b.Text) == "" {
				return s, errors.New("text message not permitted in this channel")
			}
		default:
			return s, fmt.Errorf("operation %s not permitted here", o.Kind)
		}
	}
	t.Sequence = c.Sequence
	t.Head = c.Hash
	return t, nil
}

func fieldBounds(b Body, base string) error {
	if b.Limit < 0 || b.Limit > MaxMembers {
		return errors.New("invalid resource limit")
	}
	if !SafeText(b.Text, MaxText, true) || !SafeText(b.Description, 1024, true) || !SafeText(b.Name, 80, false) || !SafeText(b.Nickname, 80, false) || !SafeText(b.Icon, 32, false) || len(b.ChannelID) > 64 || len(b.Invite) > 16000 || len(b.Target) > 128 || len(b.Role) > 16 || len(b.ChannelKind) > 8 || (base != "" && !ValidID(base)) {
		return errors.New("invalid or oversized operation fields")
	}
	return nil
}

func bodyShape(kind string, b Body) bool {
	allowed := Body{}
	switch kind {
	case Create:
		allowed = Body{Name: b.Name, Description: b.Description, Icon: b.Icon, Nickname: b.Nickname, Limit: b.Limit}
	case Metadata:
		allowed = Body{Name: b.Name, Description: b.Description, Icon: b.Icon}
	case ChannelCreate:
		allowed = Body{Name: b.Name, ChannelID: b.ChannelID, ChannelKind: b.ChannelKind, Limit: b.Limit, Role: b.Role}
	case ChannelUpdate:
		allowed = Body{ChannelID: b.ChannelID, Name: b.Name, Limit: b.Limit, Role: b.Role}
	case InviteRevoke:
		allowed = Body{Invite: b.Invite}
	case ChannelDelete:
		allowed = Body{ChannelID: b.ChannelID}
	case Join:
		allowed = Body{Name: b.Name, Invite: b.Invite}
	case SetRole:
		allowed = Body{Target: b.Target, Role: b.Role}
	case Remove:
		allowed = Body{Target: b.Target}
	case Text:
		allowed = Body{ChannelID: b.ChannelID, Text: b.Text}
	case Archive, Leave:
	default:
		return false
	}
	return b == allowed
}
