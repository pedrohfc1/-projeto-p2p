package messaging

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

const MaxFrame = 32768
const MaxText = 4096

type Message struct {
	ID        string
	From      string
	To        string
	Time      int64
	Text      string
	Signature []byte
}

func (m Message) payload() []byte { m.Signature = nil; b, _ := json.Marshal(m); return b }
func New(key crypto.PrivKey, to peer.ID, text string) (Message, error) {
	id, err := peer.IDFromPrivateKey(key)
	if err != nil {
		return Message{}, err
	}
	nonce := make([]byte, 16)
	if _, err = rand.Read(nonce); err != nil {
		return Message{}, err
	}
	m := Message{ID: hex.EncodeToString(nonce), From: id.String(), To: to.String(), Time: time.Now().Unix(), Text: text}
	if len(text) == 0 || len(text) > MaxText || !utf8.ValidString(text) {
		return Message{}, errors.New("invalid text")
	}
	m.Signature, err = key.Sign(m.payload())
	return m, err
}
func (m Message) Validate(remote, local peer.ID, key crypto.PubKey, now time.Time) error {
	nonce, err := hex.DecodeString(m.ID)
	if err != nil || len(nonce) != 16 || m.From != remote.String() || m.To != local.String() || len(m.Text) == 0 || len(m.Text) > MaxText || !utf8.ValidString(m.Text) {
		return errors.New("invalid message")
	}
	if m.Time < now.Add(-2*time.Minute).Unix() || m.Time > now.Add(2*time.Minute).Unix() {
		return errors.New("expired message")
	}
	if key == nil {
		return errors.New("missing public key")
	}
	id, err := peer.IDFromPublicKey(key)
	if err != nil || id != remote {
		return errors.New("invalid identity")
	}
	ok, err := key.Verify(m.payload(), m.Signature)
	if err != nil || !ok {
		return errors.New("invalid signature")
	}
	return nil
}
