package network

import (
	"context"
	"errors"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	client "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
)

type RelayProbe struct {
	RelayID         string    `json:"relayId"`
	ClientID        string    `json:"clientId"`
	Reserved        bool      `json:"reserved"`
	VoucherVerified bool      `json:"voucherVerified"`
	Expires         time.Time `json:"expires"`
	LimitSeconds    int64     `json:"limitSeconds"`
	LimitBytes      uint64    `json:"limitBytes"`
	Addresses       []string  `json:"addresses"`
}

// CheckRelay uses an isolated ephemeral identity. It never borrows or stores the
// application's private key. The reservation is released when this probe ends.
func CheckRelay(ctx context.Context, address string) (RelayProbe, error) {
	result := RelayProbe{Addresses: []string{}}
	p, e := ParseAddress(address)
	if e != nil {
		return result, e
	}
	h, e := libp2p.New(libp2p.NoListenAddrs, libp2p.Security(noise.ID, noise.New))
	if e != nil {
		return result, e
	}
	defer h.Close()
	// Reserve sets its own stream deadline. Closing this probe-only host on
	// cancellation also interrupts a silent peer after stream negotiation.
	stop := context.AfterFunc(ctx, func() { _ = h.Close() })
	defer stop()
	r, e := client.Reserve(ctx, h, p)
	if e != nil {
		return result, e
	}
	if r.Voucher == nil || r.Voucher.Relay != p.ID || r.Voucher.Peer != h.ID() || !r.Voucher.Expiration.Equal(r.Expiration) {
		return result, errors.New("voucher de reserva ausente ou não corresponde ao relay/cliente/expiração")
	}
	result.RelayID = p.ID.String()
	result.ClientID = h.ID().String()
	result.Reserved = true
	result.VoucherVerified = true
	result.Expires = r.Expiration.UTC()
	result.LimitSeconds = int64(r.LimitDuration / time.Second)
	result.LimitBytes = r.LimitData
	for _, a := range r.Addrs {
		result.Addresses = append(result.Addresses, a.String())
	}
	return result, nil
}
