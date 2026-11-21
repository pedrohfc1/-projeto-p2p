package desktop

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p/core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pion/rtp"
	"os"
	"path/filepath"
	"projetop2p/internal/community"
	"projetop2p/internal/localdb"
	"projetop2p/internal/transportaddr"
	"projetop2p/internal/voice"
	"sync/atomic"
	"testing"
	"time"
)

func TestRealCommunityOwnerRecoveryAndOfflineVoice(t *testing.T) {
	helper := os.Getenv("P2P_ETCAT_TEST_DIR")
	if helper == "" {
		t.Skip("requires audited real userspace EasyTier helper")
	}
	root, _ := filepath.Abs("../../data")
	os.MkdirAll(root, 0700)
	dir, e := os.MkdirTemp(root, "community-offline-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(dir)
	open := func(name string) *Core {
		c, e := Open(context.Background(), filepath.Join(dir, name))
		if e != nil {
			t.Fatal(e)
		}
		if e = c.SaveSettings(localdb.Settings{Name: name, Mode: "integrated", Port: integratedPort(t), IntegratedBinaryDir: helper}); e != nil {
			c.Close()
			t.Fatal(e)
		}
		return c
	}
	owner := open("owner")
	defer func() {
		if owner != nil {
			owner.Close()
		}
	}()
	b, c := open("b"), open("c")
	defer b.Close()
	defer c.Close()
	group, e := owner.CreateCommunity("Offline recovery", "", " ")
	if e != nil {
		t.Fatal(e)
	}
	ticket, e := owner.CommunityInvitePolicy(group, community.InviteOptions{Kind: "multi", MaxUses: 5, LifetimeHours: 24})
	if e != nil {
		t.Fatal(e)
	}
	for _, member := range []*Core{b, c} {
		r, e := member.JoinCommunity(ticket)
		if e != nil || r.Status != "participante" {
			t.Fatal(r, e)
		}
	}
	if e = b.SyncCommunity(group); e != nil {
		t.Fatal(e)
	}
	if e = c.SyncCommunity(group); e != nil {
		t.Fatal(e)
	}
	// Simulate a legacy community cache and an old peerstore from a previous LAN session.
	ownerID := owner.node.Host.ID()
	oldAddresses := owner.node.Addresses()
	record, _ := b.db.Community(group)
	record.Scope = "auto"
	record.Addresses = []string{fmt.Sprintf("/ip4/127.0.0.1/tcp/60129/p2p/%s", ownerID), fmt.Sprintf("/ip4/192.168.100.2/tcp/60129/p2p/%s", ownerID)}
	if e = b.db.TrackCommunity(record); e != nil {
		t.Fatal(e)
	}
	b.node.Host.Peerstore().AddAddrs(ownerID, []ma.Multiaddr{ma.StringCast("/ip4/192.168.100.2/tcp/60129")}, peerstore.PermanentAddrTTL)
	if e = b.SyncCommunity(group); e != nil {
		t.Fatal("legacy LAN cache must resolve over existing authenticated EasyTier capability", e)
	}
	var received [2]atomic.Uint64
	engines := make([]*voice.Engine, 2)
	browsers := make([]*voice.MediaPeer, 2)
	for i, member := range []*Core{b, c} {
		engines[i], e = voice.NewEngine("127.0.0.1")
		if e != nil {
			t.Fatal(e)
		}
		defer engines[i].Close()
		index := i
		browsers[i], e = engines[i].Peer(func(*rtp.Packet) { received[index].Add(1) })
		if e != nil {
			t.Fatal(e)
		}
		defer browsers[i].Close()
		answer, e := member.VoiceJoin(group, "voz", browserOffer(t, browsers[i]))
		if e != nil {
			t.Fatal(e)
		}
		if e = browsers[i].PC.SetRemoteDescription(answer); e != nil {
			t.Fatal(e)
		}
	}
	deadline := time.Now().Add(60 * time.Second)
	for b.VoiceState().Media.State != "connected" || c.VoiceState().Media.State != "connected" {
		if time.Now().After(deadline) {
			t.Fatal("voice not ready", b.VoiceState(), c.VoiceState())
		}
		time.Sleep(200 * time.Millisecond)
	}
	time.Sleep(4 * time.Second)
	owner.Close()
	owner = nil
	// Exceeds the old 45-second owner lease. These are existing, owner-authorized sessions.
	var finalBaseline [2]uint64
	for n := 1; n <= 3250; n++ {
		if n == 3150 {
			for i := range received {
				finalBaseline[i] = received[i].Load()
			}
		}
		if n == 900 {
			_ = b.node.Host.Network().ClosePeer(c.node.Host.ID())
		}
		for i, p := range browsers {
			if e = p.Write(&rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 111, SequenceNumber: uint16(n), Timestamp: uint32(n) * 960, SSRC: uint32(i + 1)}, Payload: []byte{0xf8, 0xff, 0xfe}}); e != nil {
				t.Fatal(e)
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	for i, member := range []*Core{b, c} {
		v := member.VoiceState()
		if !v.Joined || v.Media.State != "connected" || received[i].Load() < 1800 || received[i].Load()-finalBaseline[i] < 60 {
			t.Fatal("owner outage disrupted authorized voice", i, received[i].Load(), v)
		}
	}
	if _, e = b.CommunityAction(group, community.Text, community.Body{ChannelID: "geral", Text: "text while owner is offline"}); e != nil {
		t.Fatal(e)
	}
	time.Sleep(time.Second)
	view, e := c.CommunityState(group, "geral", 0)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, m := range view.Messages {
		if m.Text == "text while owner is offline" {
			found = true
		}
	}
	if !found {
		t.Fatal("offline peer text missing")
	}
	// Restart the same owner identity with changed native port settings. No new community or ticket.
	owner = open("owner")
	if owner.node.Host.ID() != ownerID {
		t.Fatal("owner identity changed")
	}
	if fmt.Sprint(owner.node.Addresses()) != fmt.Sprint(oldAddresses) {
		t.Fatal("integrated identity endpoint changed with local port setting")
	}
	deadline = time.Now().Add(70 * time.Second)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		e = b.groups.Sync(ctx, group)
		cancel()
		if e == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("owner did not recover", e)
		}
		time.Sleep(time.Second)
	}
	newcomer := open("new")
	defer newcomer.Close()
	r, e := newcomer.JoinCommunity(ticket)
	if e != nil || r.Status != "participante" {
		t.Fatal("old signed multiuse invitation could not rediscover restarted owner", r, e)
	}
	b.VoiceLeave()
	c.VoiceLeave()
	t.Log("PASS real userspace EasyTier, three authorized peers, legacy LAN/peerstore recovery, >65s owner-offline bidirectional voice/text, same-owner restart/changed port, old compact multiuse invite; same PC, not two houses")
}

func TestRealIntegratedSignedInvitationWithObsoletePort(t *testing.T) {
	helper := os.Getenv("P2P_ETCAT_TEST_DIR")
	if helper == "" {
		t.Skip("real audited helper optional")
	}
	root, _ := filepath.Abs("../../data")
	os.MkdirAll(root, 0700)
	dir, e := os.MkdirTemp(root, "old-port-invite-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(dir)
	open := func(name string) *Core {
		c, e := Open(context.Background(), filepath.Join(dir, name))
		if e != nil {
			t.Fatal(e)
		}
		if e = c.SaveSettings(localdb.Settings{Name: name, Mode: "integrated", Port: integratedPort(t), IntegratedBinaryDir: helper}); e != nil {
			c.Close()
			t.Fatal(e)
		}
		return c
	}
	owner, member := open("owner"), open("member")
	defer owner.Close()
	defer member.Close()
	group, e := owner.CreateCommunity("obsolete signed endpoint", "", "")
	if e != nil {
		t.Fatal(e)
	}
	record, _ := owner.db.Community(group)
	oldPort := integratedPort(t)
	if oldPort == transportaddr.ServicePort(owner.node.SelfID()) {
		oldPort = 4001
	}
	stale := fmt.Sprintf("/ip6/%s/tcp/%d/p2p/%s", transportaddr.LogicalIP(owner.node.SelfID()), oldPort, owner.node.SelfID())
	ticket, e := community.NewPolicyInvite(owner.key, record.State, "integrated", []string{stale}, community.InviteOptions{Kind: "multi", MaxUses: 5, LifetimeHours: 24}, owner.integrated.Token())
	if e != nil {
		t.Fatal(e)
	}
	receipt, e := member.JoinCommunity(ticket)
	if e != nil || receipt.Status != "participante" {
		t.Fatal("signed obsolete port prevented identity-based rediscovery", receipt, e)
	}
	for _, a := range member.node.Host.Peerstore().Addrs(owner.node.Host.ID()) {
		if p, e := a.ValueForProtocol(ma.P_TCP); e == nil && p == fmt.Sprint(oldPort) {
			t.Fatal("obsolete invite port retained in peerstore", a)
		}
	}
	oldNativeGroup, e := owner.CreateCommunity("native invitation after transport change", "", "")
	if e != nil {
		t.Fatal(e)
	}
	nativeRecord, _ := owner.db.Community(oldNativeGroup)
	nativeAddress := fmt.Sprintf("/ip4/192.168.100.2/tcp/60129/p2p/%s", owner.node.SelfID())
	nativeTicket, e := community.NewPolicyInvite(owner.key, nativeRecord.State, "auto", []string{nativeAddress}, community.InviteOptions{Kind: "multi", MaxUses: 5, LifetimeHours: 24})
	if e != nil {
		t.Fatal(e)
	}
	receipt, e = member.JoinCommunity(nativeTicket)
	if e != nil || receipt.Status != "participante" || receipt.RequiredMode != "" {
		t.Fatal("valid old native invitation did not follow known authenticated owner route", receipt, e)
	}
	t.Log("obsolete signed compact invitation resolved through authenticated EasyTier route; expected Peer ID verified")
}
