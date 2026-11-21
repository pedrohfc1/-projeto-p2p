package community

import (
	"encoding/json"
	"errors"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

const ArtworkRequest = "artwork.get/1"
const maxArtworkRevision = 1<<53 - 1

// Artwork is a signed extension scoped to the existing pinned owner and root.
// An empty channel key denotes the community itself. A missing key clears it.
type Artwork struct {
	Community string            `json:"community"`
	Root      string            `json:"root"`
	Owner     string            `json:"owner"`
	Revision  uint64            `json:"revision"`
	Icons     map[string]string `json:"icons"`
	Signature []byte            `json:"signature"`
}

func (a Artwork) payload() []byte {
	a.Signature = nil
	b, _ := json.Marshal(a)
	return append([]byte("peetwopee/community-artwork/1\x00"), b...)
}

func (a Artwork) Verify(state State) error {
	if a.Community != state.ID || a.Root != state.Root || a.Owner != state.Owner ||
		a.Revision == 0 || a.Revision > maxArtworkRevision || a.Icons == nil || len(a.Icons) > MaxChannels+1 {
		return errors.New("invalid community artwork")
	}
	for channel, hash := range a.Icons {
		if (channel != "" && !ChannelID(channel)) || !ValidID(hash) {
			return errors.New("invalid community artwork reference")
		}
	}
	return signedBy(state.Owner, state.OwnerKey, a.payload(), a.Signature)
}

func SignArtwork(key crypto.PrivKey, state State, revision uint64, icons map[string]string) (Artwork, error) {
	id, err := peer.IDFromPrivateKey(key)
	if err != nil {
		return Artwork{}, err
	}
	a := Artwork{Community: state.ID, Root: state.Root, Owner: id.String(), Revision: revision, Icons: icons}
	a.Signature, err = key.Sign(a.payload())
	if err != nil {
		return Artwork{}, err
	}
	return a, a.Verify(state)
}
