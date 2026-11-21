package localdb

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCommunityEndpointCapabilitiesSurviveReopenSealed(t *testing.T) {
	dir := t.TempDir()
	d, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	routes := map[string]string{"test-member": "private-route-capability-for-test"}
	if e = d.SaveCommunityRoutes("test-community", routes); e != nil {
		t.Fatal(e)
	}
	d.Close()
	raw, e := os.ReadFile(filepath.Join(dir, "chat.db"))
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Contains(raw, []byte(routes["test-member"])) {
		t.Fatal("route capability stored in plaintext")
	}
	d, e = Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	got, e := d.CommunityRoutes("test-community")
	if e != nil || got["test-member"] != routes["test-member"] {
		t.Fatal(got, e)
	}
}
