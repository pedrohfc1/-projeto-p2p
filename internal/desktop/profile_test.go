package desktop

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"projetop2p/internal/community"
)

func artworkImage(t *testing.T, tone uint8) *ImageAsset {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, 32, 32))
	im.SetRGBA(0, 0, color.RGBA{R: tone, A: 255})
	var out bytes.Buffer
	if err := png.Encode(&out, im); err != nil {
		t.Fatal(err)
	}
	return &ImageAsset{MIME: "image/png", Data: base64.StdEncoding.EncodeToString(out.Bytes())}
}

func TestProfileArtworkSyncPreservesCommunityAuthority(t *testing.T) {
	ownerDir, memberDir := t.TempDir(), t.TempDir()
	owner := communityCore(t, ownerDir, "Dona", freePort(t))
	defer func() { owner.Close() }()
	member := communityCore(t, memberDir, "Membro", freePort(t))
	defer func() { member.Close() }()
	stranger := communityCore(t, t.TempDir(), "Outro peer", freePort(t))
	defer stranger.Close()

	group, err := owner.CreateCommunity("Arte compartilhada", "", "🎨")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := owner.CommunityInvite(group)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := member.JoinCommunity(inv)
	if err != nil || receipt.Status != "participante" {
		t.Fatal(receipt, err)
	}
	channel := "voz"
	img1, img2 := artworkImage(t, 123), artworkImage(t, 211)
	first, err := owner.SaveCommunityArtwork(group, "", img1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := owner.SaveCommunityArtwork(group, channel, img2)
	if err != nil || second.Revision <= first.Revision {
		t.Fatal(second, err)
	}
	if _, err := member.SaveCommunityArtwork(group, channel, img1); err == nil {
		t.Fatal("ordinary member changed artwork")
	}
	if _, err := owner.SaveCommunityArtwork(group, "geral", img1); err == nil {
		t.Fatal("text channel incorrectly accepted")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	denied, err := stranger.node.CommunityRequest(ctx, owner.selfID(), owner.node.Addresses(), community.Request{Method: community.ArtworkRequest, CommunityID: group})
	if err != nil || denied.Artwork != nil || denied.Code != "membership_required" {
		t.Fatal(denied, err)
	}
	if _, err := stranger.node.FetchProfileAsset(ctx, owner.selfID(), first.Icons[""]); err == nil {
		t.Fatal("outsider fetched private community icon by hash")
	}
	if err = member.SyncCommunity(group); err != nil {
		t.Fatal(err)
	}
	verify := func() {
		t.Helper()
		v, err := member.CommunityState(group, "geral", 0)
		if err != nil || v.Community.IconHash != first.Icons[""] || v.Community.Icon != "🎨" {
			t.Fatal(v.Community, err)
		}
		for _, ch := range v.Community.Channels {
			if ch.ID == channel && ch.IconHash != second.Icons[channel] {
				t.Fatal(ch)
			}
		}
		for hash, img := range map[string]*ImageAsset{first.Icons[""]: img1, second.Icons[channel]: img2} {
			asset, err := member.ProfileAsset(hash)
			if err != nil || asset == nil || *asset != *img {
				t.Fatal("missing or modified file", hash, err)
			}
		}
	}
	verify()
	if err = os.Remove(filepath.Join(memberDir, "profile-assets", first.Icons[""]+".asset")); err != nil {
		t.Fatal(err)
	}
	if err = member.SyncCommunity(group); err != nil {
		t.Fatal(err)
	}
	verify()
	if err = member.db.SaveArtwork(first, member.selfID()); err == nil {
		t.Fatal("replayed older revision")
	}
	settings, e := member.db.Settings()
	if e != nil {
		t.Fatal(e)
	}
	memberPort := port(settings)
	member.Close()
	member = communityCore(t, memberDir, "Membro", memberPort)
	verify()
	if _, err = owner.SaveCommunityArtwork(group, "", nil); err != nil {
		t.Fatal(err)
	}
	if err = member.SyncCommunity(group); err != nil {
		t.Fatal(err)
	}
	v, err := member.CommunityState(group, "geral", 0)
	if err != nil || v.Community.IconHash != "" || v.Community.Icon != "🎨" {
		t.Fatal(v.Community, err)
	}

	node := owner.node
	profile, err := owner.SaveProfile(ProfileChange{DisplayName: "Dona Nova", DisplayFontID: "space-grotesk", Avatar: img1})
	if err != nil || profile.DisplayName != "Dona Nova" || owner.node != node {
		t.Fatal(profile, err)
	}
	if state, err := owner.State(""); err != nil || state.Settings.Name != "Dona Nova" {
		t.Fatal(state.Settings.Name, err)
	}
}

func TestProfileUnicodeDoesNotExceedLegacyNameSize(t *testing.T) {
	c := communityCore(t, t.TempDir(), "Nome", freePort(t))
	defer c.Close()
	full := strings.Repeat("🌿", 32)
	m, err := c.SaveProfile(ProfileChange{DisplayName: full, DisplayFontID: "inter"})
	if err != nil || m.DisplayName != full {
		t.Fatal(m, err)
	}
	settings, err := c.db.Settings()
	if err != nil || len(settings.Name) != 80 {
		t.Fatal(settings.Name, err)
	}
	if err = c.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	again, err := c.Profiles([]string{m.PeerID})
	if err != nil || len(again) != 1 || again[0].DisplayName != full {
		t.Fatal(again, err)
	}
}
