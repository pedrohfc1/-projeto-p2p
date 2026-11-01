package desktop

import (
	"context"
	"github.com/libp2p/go-libp2p/core/peer"
	"net"
	"os"
	"path/filepath"
	"projetop2p/internal/community"
	"projetop2p/internal/localdb"
	"testing"
	"time"
)

func integratedPort(t *testing.T) int {
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
func TestRealIntegratedChatAndCommunity(t *testing.T) {
	binary := os.Getenv("P2P_ETCAT_TEST_DIR")
	if binary == "" {
		t.Skip("optional real verified userspace helper; uses community relay")
	}
	root, e := filepath.Abs("../../data")
	if e != nil {
		t.Fatal(e)
	}
	os.MkdirAll(root, 0700)
	dir, e := os.MkdirTemp(root, "integrated-core-")
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if filepath.Dir(dir) == root {
			os.RemoveAll(dir)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	a, e := Open(ctx, filepath.Join(dir, "a"))
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	b, e := Open(ctx, filepath.Join(dir, "b"))
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	for _, c := range []*Core{a, b} {
		e = c.SaveSettings(localdb.Settings{Name: "Beta QA", Mode: "integrated", Port: integratedPort(t), IntegratedBinaryDir: binary})
		if e != nil {
			t.Fatal(e)
		}
	}
	ai, e := a.Invite()
	if e != nil {
		t.Fatal(e)
	}
	bi, e := b.Invite()
	if e != nil {
		t.Fatal(e)
	}
	if e = a.AddContact(bi); e != nil {
		t.Fatal(e)
	}
	if e = b.AddContact(ai); e != nil {
		t.Fatal(e)
	}
	as, _ := a.State("")
	bs, _ := b.State("")
	if as.Network.ID == bs.Network.ID {
		t.Fatal("shared identity")
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		e = a.Send(bs.Network.ID, "Real integrated direct message")
		if e == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("real message failed", e)
		}
		time.Sleep(time.Second)
	}
	if e = b.Send(as.Network.ID, "Real integrated reply"); e != nil {
		t.Fatal(e)
	}
	group, e := a.CreateCommunity("Integrated QA", "same production logic", "")
	if e != nil {
		t.Fatal(e)
	}
	ticket, e := a.CommunityInvite(group)
	if e != nil {
		t.Fatal(e)
	}
	receipt, e := b.JoinCommunity(ticket)
	if e != nil || receipt.Status != "participante" {
		t.Fatal("community join", receipt.Status, e)
	}
	receipt, e = b.CommunityAction(group, "message.text", community.Body{ChannelID: "geral", Text: "Real integrated channel text"})
	if e != nil || receipt.Status == "falhou" {
		t.Fatal("channel message", receipt, e)
	}
	view, e := a.CommunityState(group, "geral", 0)
	if e != nil || len(view.Messages) != 1 {
		t.Fatal("channel message not persisted", e)
	}
	c, e := Open(ctx, filepath.Join(dir, "c"))
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	if e = c.SaveSettings(localdb.Settings{Name: "Third beta peer", Mode: "integrated", Port: integratedPort(t), IntegratedBinaryDir: binary}); e != nil {
		t.Fatal(e)
	}
	if e = c.AddContact(ai); e != nil {
		t.Fatal(e)
	}
	deadline = time.Now().Add(45 * time.Second)
	for {
		probe, err := c.TestConnectivity(as.Network.ID)
		if err == nil && probe.Connected {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("third authenticated connectivity", probe.Code, probe.Message, err)
		}
		time.Sleep(time.Second)
	}
	if e = c.Send(as.Network.ID, "Third participant DM"); e != nil {
		t.Fatal(e)
	}
	ticket, e = a.CommunityInvite(group)
	if e != nil {
		t.Fatal(e)
	}
	receipt, e = c.JoinCommunity(ticket)
	if e != nil || receipt.Status != "participante" {
		t.Fatal("third join", receipt, e)
	}
	receipt, e = c.CommunityAction(group, "message.text", community.Body{ChannelID: "geral", Text: "Third member channel"})
	if e != nil {
		t.Fatal(e)
	}
	if e = b.Send(as.Network.ID, "First client still connected"); e != nil {
		t.Fatal("shared credential collision", e)
	}
	// Restart the app with the same profile: identity, sealed route, contacts and history survive.
	b.Close()
	b, e = Open(ctx, filepath.Join(dir, "b"))
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	bs2, e := b.State(as.Network.ID)
	if e != nil || bs2.Network.ID != bs.Network.ID {
		t.Fatal("identity changed", e)
	}
	deadline = time.Now().Add(45 * time.Second)
	for {
		probe, err := b.TestConnectivity(as.Network.ID)
		if err == nil && probe.Connected {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("restart connectivity", probe.Code, err)
		}
		time.Sleep(time.Second)
	}
	// Compose once, deliver twice across a restarted connection: receiver must deduplicate by signed ID.
	p, _ := peer.Decode(as.Network.ID)
	msg, err := b.node.Compose(p, "same signed message twice")
	if err != nil {
		t.Fatal(err)
	}
	if err = b.node.Deliver(ctx, msg); err != nil {
		t.Fatal(err)
	}
	if err = b.node.Host.Network().ClosePeer(p); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(35 * time.Second)
	for {
		err = b.node.Deliver(ctx, msg)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("signed retry after connection close", err)
		}
		time.Sleep(time.Second)
	}
	state, err := a.State(bs.Network.ID)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, item := range state.Messages {
		if item.ID == msg.ID {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("duplicate message rows=%d", count)
	}
	// Respect the existing ten-messages/second receiver limit between different IDs.
	time.Sleep(150 * time.Millisecond)
	if e = b.Send(as.Network.ID, "After app restart"); e != nil {
		t.Fatal("route persistence", e)
	}
	t.Log("real production direct chat, reply, signed community join and channel persistence passed without external VPN")
}
