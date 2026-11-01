package network

import (
	"context"
	"errors"
	libnet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"projetop2p/internal/security"
	"projetop2p/internal/voice"
	"time"
)

const VoiceProtocol = protocol.ID("/projeto-p2p/voice/0.6.0")
const maxVoiceFrame = 65536

func (n *Node) SetVoiceHandler(handler func(context.Context, string, voice.Request) voice.Response) {
	if handler == nil {
		n.Host.RemoveStreamHandler(VoiceProtocol)
		return
	}
	gate := &security.StreamGate{}
	n.Host.SetStreamHandler(VoiceProtocol, func(stream libnet.Stream) {
		defer stream.Close()
		release, ok := gate.Acquire(stream.Conn().RemotePeer().String(), time.Now())
		if !ok {
			stream.Reset()
			return
		}
		defer release()
		ctx, cancel := context.WithTimeout(n.ctx, 38*time.Second)
		defer cancel()
		stop := context.AfterFunc(ctx, func() { stream.Reset() })
		defer stop()
		stream.SetDeadline(time.Now().Add(38 * time.Second))
		var req voice.Request
		if readCommunityFrame(stream, maxVoiceFrame, &req) != nil {
			stream.Reset()
			return
		}
		res := handler(ctx, stream.Conn().RemotePeer().String(), req)
		if writeCommunityFrame(stream, maxVoiceFrame, res) != nil {
			stream.Reset()
		}
	})
}
func (n *Node) VoiceRequest(ctx context.Context, id string, addresses []string, req voice.Request) (voice.Response, error) {
	var result voice.Response
	pid, e := peer.Decode(id)
	if e != nil {
		return result, e
	}
	if len(addresses) > 16 {
		return result, errors.New("too many voice endpoints")
	}
	info := peer.AddrInfo{ID: pid}
	for _, raw := range addresses {
		a, e := ParseAddress(raw)
		if e != nil || a.ID != pid {
			return result, errors.New("voice endpoint identity mismatch")
		}
		info.Addrs = append(info.Addrs, a.Addrs...)
	}
	if e = n.Connect(ctx, info); e != nil {
		return result, e
	}
	stream, e := n.Host.NewStream(libnet.WithAllowLimitedConn(ctx, "authenticated voice signaling"), pid, VoiceProtocol)
	if e != nil {
		return result, e
	}
	defer stream.Close()
	stop := context.AfterFunc(ctx, func() { stream.Reset() })
	defer stop()
	deadline := time.Now().Add(38 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	stream.SetDeadline(deadline)
	if e = writeCommunityFrame(stream, maxVoiceFrame, req); e != nil {
		stream.Reset()
		return result, e
	}
	e = readCommunityFrame(stream, maxVoiceFrame, &result)
	return result, e
}
