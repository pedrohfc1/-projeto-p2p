package chat

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type Room struct {
	Group   string `json:"group,omitempty"`
	Channel string `json:"channel,omitempty"`
	Peer    string `json:"peer,omitempty"`
}
type Event struct {
	TargetAuthor string `json:"targetAuthor,omitempty"`
	Deaf         bool   `json:"deaf,omitempty"`
	Domain       string `json:"domain"`
	ID           string `json:"id"`
	Actor        string `json:"actor"`
	Key          []byte `json:"key"`
	Room         Room   `json:"room"`
	Root         string `json:"root,omitempty"`
	Head         string `json:"head,omitempty"`
	Epoch        uint64 `json:"epoch,omitempty"`
	At           int64  `json:"at"`
	Nonce        string `json:"nonce"`
	Kind         string `json:"kind"`
	Target       string `json:"target,omitempty"`
	Text         string `json:"text,omitempty"`
	Emoji        string `json:"emoji,omitempty"`
	Enabled      bool   `json:"enabled,omitempty"`
	Signature    []byte `json:"signature"`
}
type Entry struct {
	Sequence uint64 `json:"sequence"`
	Event    Event  `json:"event"`
}
type Stamp struct {
	At int64  `json:"at"`
	ID string `json:"id"`
}
type Toggle struct {
	Stamp   Stamp `json:"stamp"`
	Enabled bool  `json:"enabled"`
}
type Overlay struct {
	ID        string            `json:"id"`
	Author    string            `json:"author"`
	Channel   string            `json:"channel"`
	Text      string            `json:"text"`
	Time      int64             `json:"time"`
	ReplyTo   string            `json:"replyTo,omitempty"`
	Edited    bool              `json:"edited,omitempty"`
	Deleted   bool              `json:"deleted,omitempty"`
	Edit      Stamp             `json:"edit"`
	Reactions map[string]Toggle `json:"reactions,omitempty"`
}
type Reaction struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
	Mine  bool   `json:"mine"`
}

var Emojis = []string{"👍", "❤️", "😂", "🎉", "👀", "😢"}

func Hex(s string, n int) bool  { b, e := hex.DecodeString(s); return e == nil && len(b) == n }
func validTarget(s string) bool { return Hex(s, 16) || Hex(s, 32) }
func (e Event) payload() []byte { e.ID = ""; e.Signature = nil; b, _ := json.Marshal(e); return b }
func Sign(key crypto.PrivKey, e Event) (Event, error) {
	e.Domain = "peetwopee/chat-activity/1"
	id, _ := peer.IDFromPrivateKey(key)
	e.Actor = id.String()
	if e.Room.Group == "" && e.Kind != "typing" && e.TargetAuthor == "" {
		e.TargetAuthor = e.Actor
	}
	e.Key, _ = crypto.MarshalPublicKey(key.GetPublic())
	e.At = time.Now().UnixMilli()
	e.Signature = nil
	e.ID = ""
	h := sha256.Sum256(e.payload())
	e.ID = hex.EncodeToString(h[:])
	var err error
	e.Signature, err = key.Sign(e.payload())
	if err != nil {
		return e, err
	}
	return e, Verify(e, time.Now())
}
func Verify(e Event, now time.Time) error {
	if !utf8.ValidString(e.Text) {
		return errors.New("invalid UTF-8")
	}
	for _, r := range e.Text {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return errors.New("invalid control character")
		}
	}
	if e.Domain != "peetwopee/chat-activity/1" || !Hex(e.Nonce, 16) || len(e.Key) > 128 || len(e.Signature) != 64 || len(e.Actor) > 128 || len(e.Text) > 4096 || len(e.Emoji) > 16 || e.At > now.Add(5*time.Minute).UnixMilli() || e.At < now.Add(-30*24*time.Hour).UnixMilli() {
		return errors.New("invalid activity bounds/time")
	}
	key, err := crypto.UnmarshalPublicKey(e.Key)
	if err != nil || key.Type() != crypto.Ed25519 {
		return errors.New("Ed25519 required")
	}
	id, err := peer.IDFromPublicKey(key)
	if err != nil || id.String() != e.Actor {
		return errors.New("activity identity mismatch")
	}
	h := sha256.Sum256(e.payload())
	if e.ID != hex.EncodeToString(h[:]) {
		return errors.New("activity hash mismatch")
	}
	ok, err := key.Verify(e.payload(), e.Signature)
	if err != nil || !ok {
		return errors.New("invalid activity signature")
	}
	if e.Room.Group != "" {
		if !Hex(e.Room.Group, 32) || len(e.Room.Channel) < 1 || len(e.Room.Channel) > 64 || !Hex(e.Root, 32) || !Hex(e.Head, 32) || e.Epoch == 0 || e.Room.Peer != "" {
			return errors.New("invalid community binding")
		}
	} else {
		if _, err = peer.Decode(e.Room.Peer); err != nil || e.Room.Channel != "" || e.Root != "" || e.Head != "" || e.Epoch != 0 {
			return errors.New("invalid direct binding")
		}
	}
	if e.Room.Group != "" && e.TargetAuthor != "" {
		return errors.New("invalid group target author")
	}
	if e.Room.Group == "" && e.Kind != "typing" {
		if _, err := peer.Decode(e.TargetAuthor); err != nil || len(e.TargetAuthor) > 128 {
			return errors.New("invalid direct target author")
		}
	}
	if e.Kind != "voice" && e.Deaf {
		return errors.New("invalid deaf flag")
	}
	switch e.Kind {
	case "reply", "edit":
		if !validTarget(e.Target) || strings.TrimSpace(e.Text) == "" || e.Emoji != "" || e.Enabled {
			return errors.New("invalid text activity")
		}
	case "delete":
		if !validTarget(e.Target) || e.Text != "" || e.Emoji != "" || e.Enabled {
			return errors.New("invalid deletion")
		}
	case "reaction":
		found := false
		for _, v := range Emojis {
			found = found || v == e.Emoji
		}
		if !validTarget(e.Target) || !found || e.Text != "" {
			return errors.New("invalid reaction")
		}
	case "voice":
		if e.Room.Group == "" || !Hex(e.Target, 16) || e.Text != "" || e.Emoji != "" || e.At < now.Add(-8*time.Second).UnixMilli() || e.At > now.Add(2*time.Second).UnixMilli() {
			return errors.New("invalid voice presence")
		}
	case "typing":
		if e.Target != "" || e.Text != "" || e.Emoji != "" || e.At < now.Add(-6*time.Second).UnixMilli() || e.At > now.Add(2*time.Second).UnixMilli() {
			return errors.New("expired typing")
		}
	default:
		return errors.New("unknown activity")
	}
	return nil
}
func Context(r Room, self, actor string) string {
	if r.Group != "" {
		return "g:" + r.Group
	}
	other := r.Peer
	if actor != self {
		other = actor
	}
	return "d:" + other
}
func Newer(a, b Stamp) bool { return a.At > b.At || a.At == b.At && a.ID > b.ID }
func Apply(p Overlay, e Event) Overlay {
	if p.Reactions == nil {
		p.Reactions = map[string]Toggle{}
	}
	stamp := Stamp{e.At, e.ID}
	switch e.Kind {
	case "edit":
		if !p.Deleted && Newer(stamp, p.Edit) {
			p.Text = e.Text
			p.Edited = true
			p.Edit = stamp
		}
	case "delete":
		p.Deleted = true
		p.Text = ""
		p.Reactions = map[string]Toggle{}
	case "reaction":
		key := e.Actor + ":" + e.Emoji
		old := p.Reactions[key]
		if !p.Deleted && Newer(stamp, old.Stamp) {
			p.Reactions[key] = Toggle{stamp, e.Enabled}
		}
	}
	return p
}
func Counts(p Overlay, self string) []Reaction {
	out := []Reaction{}
	for _, emoji := range Emojis {
		v := Reaction{Emoji: emoji}
		for key, state := range p.Reactions {
			if state.Enabled && strings.HasSuffix(key, ":"+emoji) {
				v.Count++
				v.Mine = v.Mine || strings.HasPrefix(key, self+":")
			}
		}
		if v.Count > 0 {
			out = append(out, v)
		}
	}
	return out
}
