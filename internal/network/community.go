package network

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	libnet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"io"
	"projetop2p/internal/community"
	"projetop2p/internal/security"
	"time"
)

const CommunityProtocol = protocol.ID("/projeto-p2p/community/0.5.0")

func readCommunityFrame(reader io.Reader, max int, value any) error {
	b, e := bufio.NewReader(io.LimitReader(reader, int64(max)+1)).ReadBytes('\n')
	if e != nil || len(b) > max {
		return errors.New("community frame incomplete or too large")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(value); e != nil {
		return e
	}
	var extra any
	if e = d.Decode(&extra); e != io.EOF {
		return errors.New("trailing JSON in frame")
	}
	return nil
}
func writeCommunityFrame(w io.Writer, max int, value any) error {
	b, e := json.Marshal(value)
	if e != nil {
		return e
	}
	if len(b)+1 > max {
		return errors.New("community frame too large")
	}
	b = append(b, '\n')
	_, e = w.Write(b)
	return e
}
func (n *Node) SetCommunityHandler(handler func(context.Context, string, community.Request) community.Response) {
	gate := &security.StreamGate{}
	n.Host.SetStreamHandler(CommunityProtocol, func(s libnet.Stream) {
		defer s.Close()
		release, ok := gate.Acquire(s.Conn().RemotePeer().String(), time.Now())
		if !ok {
			s.Reset()
			return
		}
		defer release()
		ctx, cancel := context.WithTimeout(n.ctx, 10*time.Second)
		defer cancel()
		stop := context.AfterFunc(ctx, func() { s.Reset() })
		defer stop()
		_ = s.SetDeadline(time.Now().Add(10 * time.Second))
		var request community.Request
		if e := readCommunityFrame(s, community.MaxRequest, &request); e != nil {
			s.Reset()
			return
		}
		response := handler(ctx, s.Conn().RemotePeer().String(), request)
		if e := writeCommunityFrame(s, community.MaxResponse, response); e != nil {
			s.Reset()
		}
	})
}
func (n *Node) SelfID() string { return n.Host.ID().String() }
func (n *Node) CommunityRequest(ctx context.Context, id string, addresses []string, request community.Request) (community.Response, error) {
	var response community.Response
	pid, e := peer.Decode(id)
	if e != nil {
		return response, e
	}
	info := peer.AddrInfo{ID: pid}
	if len(addresses) > 8 {
		return response, errors.New("too many community endpoints")
	}
	for _, raw := range addresses {
		p, e := ParseAddress(raw)
		if e != nil || p.ID != pid {
			return response, errors.New("invalid owner endpoint")
		}
		info.Addrs = append(info.Addrs, p.Addrs...)
	}
	if e = n.Connect(ctx, info); e != nil {
		return response, e
	}
	s, e := n.Host.NewStream(libnet.WithAllowLimitedConn(ctx, "community via authenticated relay"), pid, CommunityProtocol)
	if e != nil {
		return response, e
	}
	defer s.Close()
	stop := context.AfterFunc(ctx, func() { s.Reset() })
	defer stop()
	deadline := time.Now().Add(10 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = s.SetDeadline(deadline)
	if e = writeCommunityFrame(s, community.MaxRequest, request); e != nil {
		s.Reset()
		return response, e
	}
	e = readCommunityFrame(s, community.MaxResponse, &response)
	if e != nil {
		s.Reset()
	}
	return response, e
}
