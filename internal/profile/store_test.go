package profile

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
)

func testKey(t *testing.T) crypto.PrivKey {
	t.Helper()
	key, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func openTestStore(t *testing.T, dir string, key crypto.PrivKey) *Store {
	t.Helper()
	store, err := Open(dir, key, "Nome prévio")
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func square(t *testing.T, side int, tint uint8) []byte {
	t.Helper()
	var out bytes.Buffer
	pixels := image.NewRGBA(image.Rect(0, 0, side, side))
	pixels.SetRGBA(0, 0, color.RGBA{R: tint, A: 255})
	if err := png.Encode(&out, pixels); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func putTestImage(t *testing.T, store *Store, side int, tint uint8) Asset {
	t.Helper()
	asset, err := store.PutImage(square(t, side, tint))
	if err != nil {
		t.Fatal(err)
	}
	return asset
}

func TestProfilePersistsRevisionNameFontAvatarAndSameIdentity(t *testing.T) {
	dir := t.TempDir()
	key := testKey(t)
	store := openTestStore(t, dir, key)
	previous := store.Local()
	if previous.DisplayName != "Nome prévio" || previous.Revision != 1 || previous.AvatarHash != "" {
		t.Fatal(previous)
	}
	asset := putTestImage(t, store, 64, 15)
	updated, err := store.Update("  Ana\u202e\x03 Café 👋  ", "space-grotesk", asset.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if updated.DisplayName != "Ana Café 👋" || updated.PeerID != previous.PeerID || updated.AvatarMIME != "image/png" || updated.DisplayFontID != "space-grotesk" || updated.Revision != previous.Revision+1 {
		t.Fatal(updated)
	}
	reopened := openTestStore(t, dir, key)
	if reopened.Local() != updated {
		t.Fatal("revision, name, font, avatar or identity did not survive restart")
	}
	if err := Verify(updated.PeerID, reopened.Signed()); err != nil {
		t.Fatal(err)
	}
	duplicate, err := reopened.Update(updated.DisplayName, updated.DisplayFontID, asset.Hash)
	if err != nil || duplicate != updated {
		t.Fatal("no-op incorrectly changed the revision", duplicate, err)
	}
	removed, err := reopened.Update(updated.DisplayName, "", "")
	if err != nil || removed.AvatarHash != "" || removed.AvatarMIME != "" || removed.DisplayFontID != "inter" || removed.Revision != updated.Revision+1 {
		t.Fatal(removed, err)
	}
	if openTestStore(t, dir, key).Local() != removed {
		t.Fatal("avatar removal did not survive restart")
	}
	if _, err := reopened.Update(strings.Repeat("á", 33), "inter", ""); err == nil {
		t.Fatal("overlong Unicode name was accepted")
	}
	if _, err := reopened.Update("none", "a;font", ""); err == nil {
		t.Fatal("unrecognized syntax accepted as a font identifier")
	}
}

func TestProfileRemoteAcceptRequiresIntactSignatureAndAsset(t *testing.T) {
	a := openTestStore(t, t.TempDir(), testKey(t))
	dirB, keyB := t.TempDir(), testKey(t)
	b := openTestStore(t, dirB, keyB)
	id := a.Local().PeerID
	firstAsset := putTestImage(t, a, 64, 100)
	_, err := a.Update("Remoto", "future-font", firstAsset.Hash)
	if err != nil {
		t.Fatal(err)
	}
	first := a.Signed()
	if _, err := b.PutExpectedImage(firstAsset.Hash, firstAsset.MIME, square(t, 64, 100)); err != nil {
		t.Fatal(err)
	}
	if err := b.Accept(id, first); err != nil {
		t.Fatal(err)
	}
	if err := b.Accept(b.Local().PeerID, first); err == nil {
		t.Fatal("profile signed by another identity accepted")
	}
	secondAsset := putTestImage(t, a, 64, 127)
	_, err = a.Update("Nome atualizado", "inter", secondAsset.Hash)
	if err != nil {
		t.Fatal(err)
	}
	second := a.Signed()
	if err := b.Accept(id, second); err == nil {
		t.Fatal("accepted metadata before the required image")
	}
	if retained, _ := b.Get(id); retained != first.Metadata {
		t.Fatal("failed fetch removed previous profile", retained)
	}
	if _, err := b.PutExpectedImage(secondAsset.Hash, secondAsset.MIME, square(t, 64, 100)); !errors.Is(err, ErrHashMismatch) {
		t.Fatal("substituted bytes accepted", err)
	}
	if _, err := b.PutExpectedImage(secondAsset.Hash, "image/webp", square(t, 64, 127)); err == nil {
		t.Fatal("substituted MIME accepted")
	}
	if _, err := b.PutExpectedImage(secondAsset.Hash, secondAsset.MIME, square(t, 64, 127)); err != nil {
		t.Fatal(err)
	}
	tampered := second
	tampered.DisplayName = "Falsificado"
	if err := b.Accept(id, tampered); err == nil {
		t.Fatal("profile metadata was accepted without a valid signature")
	}
	if err := b.Accept(id, second); err != nil {
		t.Fatal(err)
	}
	b = openTestStore(t, dirB, keyB)
	if err := b.Accept(id, first); err == nil {
		t.Fatal("accepted lower revision after restart")
	}
	known, ok := b.Get(id)
	if !ok || known != second.Metadata {
		t.Fatal("latest verified remote profile did not survive restart")
	}
	if _, _, err := b.ReadImage(secondAsset.Hash); err != nil {
		t.Fatal(err)
	}
}

func TestProfileImageContentAndLimits(t *testing.T) {
	store := openTestStore(t, t.TempDir(), testKey(t))
	for label, data := range map[string][]byte{
		"SVG":        []byte("<svg xmlns='http://www.w3.org/2000/svg'/>"),
		"oversize":   bytes.Repeat([]byte{'x'}, MaxImageBytes+1),
		"dimensions": square(t, MaxImageSide+1, 100),
		"truncated":  square(t, 64, 100)[:20],
	} {
		if _, err := store.PutImage(data); err == nil {
			t.Errorf("accepted %s image", label)
		}
	}
	if _, _, err := store.ReadImage("../identity.key"); err == nil {
		t.Fatal("accepted a non-hash path")
	}
	// Tiny synthetic WebP exports from libwebp and the actual Edge canvas.
	for label, fixture := range map[string]string{
		"libwebp":                           "UklGRkgAAABXRUJQVlA4IDwAAABQAwCdASogACAAPpFEnEqlo6KhqAgAsBIJZQC7LoAAQFEcAAD+4uC/+Wf+f3xd6nP4rfUDIRTQ8anAAAA=",
		"Edge canvas transparency and sRGB": "UklGRn4CAABXRUJQVlA4WAoAAAAwAAAAHwAAHwAASUNDUMgBAAAAAAHIAAAAAAQwAABtbnRyUkdCIFhZWiAH4AABAAEAAAAAAABhY3NwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAA9tYAAQAAAADTLQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAlkZXNjAAAA8AAAACRyWFlaAAABFAAAABRnWFlaAAABKAAAABRiWFlaAAABPAAAABR3dHB0AAABUAAAABRyVFJDAAABZAAAAChnVFJDAAABZAAAAChiVFJDAAABZAAAAChjcHJ0AAABjAAAADxtbHVjAAAAAAAAAAEAAAAMZW5VUwAAAAgAAAAcAHMAUgBHAEJYWVogAAAAAAAAb6IAADj1AAADkFhZWiAAAAAAAABimQAAt4UAABjaWFlaIAAAAAAAACSgAAAPhAAAts9YWVogAAAAAAAA9tYAAQAAAADTLXBhcmEAAAAAAAQAAAACZmYAAPKnAAANWQAAE9AAAApbAAAAAAAAAABtbHVjAAAAAAAAAAEAAAAMZW5VUwAAACAAAAAcAEcAbwBvAGcAbABlACAASQBuAGMALgAgADIAMAAxADZBTFBIHAAAAAEPMFkREUJR20ZMmZQ/x77HDQAR/Z8A2OK2HwFWUDggbAAAABAFAJ0BKiAAIAA+SR6KQ6KhoRgMBgAoBISyAF7/QCiAfgC2APQA/QDLAOlGAjfucoAA/vDXA/TNvpRFabiAqt+Zl24UPwUKJ/2grOaDAG/pYtMNO+xT7OTflsQfBv/9FulHX/rmKbtDAAAAAA==",
	} {
		webp, err := base64.StdEncoding.DecodeString(fixture)
		if err != nil {
			t.Fatal(label, err)
		}
		asset, err := store.PutImage(webp)
		if err != nil || asset.MIME != "image/webp" {
			t.Fatal(label, asset, err)
		}
	}
}

func TestProfilePrunesOnlyUnreferencedExpiredArtwork(t *testing.T) {
	store := openTestStore(t, t.TempDir(), testKey(t))
	old := putTestImage(t, store, 32, 15)
	active := putTestImage(t, store, 32, 32)
	room := putTestImage(t, store, 32, 51)
	if _, err := store.Update("Meu perfil", "inter", active.Hash); err != nil {
		t.Fatal(err)
	}
	expired := time.Now().Add(-orphanRetention - time.Hour)
	for _, asset := range []Asset{old, active, room} {
		if err := os.Chtimes(filepath.Join(store.assets, asset.Hash+".asset"), expired, expired); err != nil {
			t.Fatal(err)
		}
	}
	putTestImage(t, store, 32, 88)
	if _, _, err := store.ReadImage(old.Hash); err != nil {
		t.Fatal("artwork references were not yet loaded, so nothing is safely evictable", err)
	}
	store.SetArtworkReferences([]string{room.Hash})
	putTestImage(t, store, 32, 91)
	if _, _, err := store.ReadImage(old.Hash); !os.IsNotExist(err) {
		t.Fatal("old orphan was not pruned", err)
	}
	for _, asset := range []Asset{active, room} {
		if _, _, err := store.ReadImage(asset.Hash); err != nil {
			t.Fatal("referenced image was deleted", asset, err)
		}
	}
}
