package desktop

import (
	"context"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"os"
	"path/filepath"
	"projetop2p/internal/community"
	"projetop2p/internal/localdb"
	"projetop2p/internal/voice"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func browserOffer(t *testing.T, p *voice.MediaPeer) webrtc.SessionDescription {
	t.Helper()
	d, e := p.PC.CreateOffer(nil)
	if e != nil {
		t.Fatal(e)
	}
	ready := webrtc.GatheringCompletePromise(p.PC)
	if e = p.PC.SetLocalDescription(d); e != nil {
		t.Fatal(e)
	}
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("local browser simulation SDP")
	}
	return *p.PC.LocalDescription()
}
func TestRealTwoPersonVoiceProductionService(t *testing.T) {
	binary := os.Getenv("P2P_ETCAT_TEST_DIR")
	if binary == "" {
		t.Skip("optional real userspace EasyTier media test")
	}
	root, _ := filepath.Abs("../../data")
	os.MkdirAll(root, 0700)
	folder, e := os.MkdirTemp(root, "voice-core-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(folder)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	a, e := Open(ctx, filepath.Join(folder, "a"))
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	b, e := Open(ctx, filepath.Join(folder, "b"))
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	for _, c := range []*Core{a, b} {
		if e = c.SaveSettings(localdb.Settings{Name: "Voice QA", Mode: "integrated", Port: integratedPort(t), IntegratedBinaryDir: binary}); e != nil {
			t.Fatal(e)
		}
	}
	group, e := a.CreateCommunity("Real voice QA", "production services", "")
	if e != nil {
		t.Fatal(e)
	}
	ticket, e := a.CommunityInvite(group)
	if e != nil {
		t.Fatal(e)
	}
	receipt, e := b.JoinCommunity(ticket)
	if e != nil || receipt.Status != "participante" {
		t.Fatal("voice member join", receipt, e)
	}
	var heardA, heardB atomic.Uint64
	engineA, e := voice.NewEngine("127.0.0.1")
	if e != nil {
		t.Fatal(e)
	}
	defer engineA.Close()
	engineB, e := voice.NewEngine("127.0.0.1")
	if e != nil {
		t.Fatal(e)
	}
	defer engineB.Close()
	browserA, e := engineA.Peer(func(*rtp.Packet) { heardA.Add(1) })
	if e != nil {
		t.Fatal(e)
	}
	defer browserA.Close()
	browserB, e := engineB.Peer(func(*rtp.Packet) { heardB.Add(1) })
	if e != nil {
		t.Fatal(e)
	}
	defer browserB.Close()
	answer, e := a.VoiceJoin(group, "voz", browserOffer(t, browserA))
	if e != nil {
		t.Fatal(e)
	}
	if e = browserA.PC.SetRemoteDescription(answer); e != nil {
		t.Fatal(e)
	}
	answer, e = b.VoiceJoin(group, "voz", browserOffer(t, browserB))
	if e != nil {
		t.Fatal(e)
	}
	if e = browserB.PC.SetRemoteDescription(answer); e != nil {
		t.Fatal(e)
	}
	deadline := time.Now().Add(65 * time.Second)
	for a.VoiceState().Media.State != "connected" || b.VoiceState().Media.State != "connected" {
		if time.Now().After(deadline) {
			t.Fatal("production audio connection", a.VoiceState(), b.VoiceState())
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Log("authenticated production voice connected", a.VoiceState().Media, b.VoiceState().Media)
	// Exercise malicious renegotiation through the actual authenticated stream.
	initiator, responder := a, b
	if initiator.node.SelfID() > responder.node.SelfID() {
		initiator, responder = responder, initiator
	}
	ownInvite, err := a.Invite()
	if err != nil {
		t.Fatal(err)
	}
	auth := a.voice.Handle(ctx, a.node.SelfID(), voice.Request{Method: "renew", Community: group, Channel: "voz", Participant: voice.Participant{ID: a.node.SelfID(), Session: a.VoiceState().Session, Invite: ownInvite, UDPPort: a.voice.Port()}})
	if auth.Lease == nil || auth.Error != "" {
		t.Fatal(auth)
	}
	initiatorInvite, err := initiator.Invite()
	if err != nil {
		t.Fatal(err)
	}
	request := voice.Request{Method: "signal", Community: group, Channel: "voz", Participant: voice.Participant{ID: initiator.node.SelfID(), Session: initiator.VoiceState().Session, Invite: initiatorInvite, UDPPort: initiator.voice.Port()}, Lease: auth.Lease, Session: responder.VoiceState().Session, Nonce: community.RandomID(), SDP: browserA.PC.LocalDescription()}
	rejected, err := initiator.node.VoiceRequest(ctx, responder.node.SelfID(), responder.node.Addresses(), request)
	if err != nil || !strings.Contains(rejected.Error, "replay or glare rejected") {
		t.Fatal("active media hijack", rejected, err)
	}
	badLease := *auth.Lease
	badLease.Signature = append([]byte(nil), badLease.Signature...)
	badLease.Signature[0] ^= 1
	request.Lease = &badLease
	rejected, err = initiator.node.VoiceRequest(ctx, responder.node.SelfID(), responder.node.Addresses(), request)
	if err != nil || !strings.Contains(rejected.Error, "invalid room signature") {
		t.Fatal("forged voice authorization", rejected, err)
	}
	request.Lease = auth.Lease
	request.Participant.ID = responder.node.SelfID()
	rejected, err = initiator.node.VoiceRequest(ctx, responder.node.SelfID(), responder.node.Addresses(), request)
	if err != nil || !strings.Contains(rejected.Error, "invalid audio signaling") {
		t.Fatal("voice identity spoof", rejected, err)
	}
	seq := uint16(0)
	send := func(p *voice.MediaPeer, count int) {
		for n := 0; n < count; n++ {
			seq++
			packet := &rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 111, SequenceNumber: seq, Timestamp: uint32(seq) * 960, SSRC: 124}, Payload: []byte{0xf8, 0xff, 0xfe}}
			if e = p.Write(packet); e != nil {
				t.Fatal(e)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	send(browserA, 25)
	send(browserB, 25)
	time.Sleep(300 * time.Millisecond)
	if heardA.Load() < 20 || heardB.Load() < 20 {
		t.Fatal("full audio bridge", heardA.Load(), heardB.Load(), a.VoiceState(), b.VoiceState())
	}
	// Interrupt only test-owned media helpers; chat/signaling helpers remain alive.
	interrupted := time.Now()
	a.integrated.CloseMedia(b.node.SelfID())
	b.integrated.CloseMedia(a.node.SelfID())
	time.Sleep(22 * time.Second)
	deadline = time.Now().Add(50 * time.Second)
	for a.VoiceState().Media.State != "connected" || b.VoiceState().Media.State != "connected" {
		if time.Now().After(deadline) {
			t.Fatal("media did not recover", a.VoiceState(), b.VoiceState())
		}
		time.Sleep(200 * time.Millisecond)
	}
	beforeRecovery := heardB.Load()
	send(browserA, 25)
	time.Sleep(300 * time.Millisecond)
	if heardB.Load() < beforeRecovery+20 {
		t.Fatal("reconnected ICE without usable media", a.VoiceState(), b.VoiceState())
	}
	t.Log("reconnection with fresh forwarding and authenticated SDP", time.Since(interrupted))
	a.VoiceControls(true, false)
	time.Sleep(150 * time.Millisecond)
	before := heardB.Load()
	send(browserA, 10)
	time.Sleep(200 * time.Millisecond)
	if heardB.Load() != before {
		t.Fatal("muted audio escaped")
	}
	a.VoiceControls(false, false)
	b.VoiceControls(false, true)
	before = heardB.Load()
	send(browserA, 10)
	time.Sleep(200 * time.Millisecond)
	if heardB.Load() != before {
		t.Fatal("deafened audio delivered")
	}
	b.VoiceControls(false, false)
	// Existing text channels continue while the audio session remains joined.
	result, e := b.CommunityAction(group, community.Text, community.Body{ChannelID: "geral", Text: "text while real Opus media is active"})
	if e != nil || (result.Status != "replicada" && result.Status != "salva localmente") {
		t.Fatal("chat regression during call", result, e)
	}
	peerID := b.node.Host.ID().String()
	result, e = a.CommunityAction(group, community.Remove, community.Body{Target: peerID})
	if e != nil || result.Status != "confirmada" {
		t.Fatal(e, result)
	}
	deadline = time.Now().Add(8 * time.Second)
	for a.VoiceState().Joined || b.VoiceState().Joined {
		if time.Now().After(deadline) {
			t.Fatal("revocation did not stop media", a.VoiceState(), b.VoiceState())
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Log("bidirectional synthetic Opus through real production libp2p signaling + EasyTier UDP, mute, deaf, concurrent text and revoked-member shutdown passed; not microphone or two-house test")
}
