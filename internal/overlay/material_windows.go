package overlay

import (
	"crypto/rand"
	"encoding/hex"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"unsafe"
)

// Plain material exists only in a private, delete-on-handle-close file while the helper reads it.
// The OS removes it even when the host process crashes.
func privateMaterial(dir string, data []byte) (string, func(), error) {
	u, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		return "", nil, e
	}
	sd, e := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + u.User.Sid.String() + ")(A;;FA;;;SY)")
	if e != nil {
		return "", nil, e
	}
	a := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	b := make([]byte, 16)
	if _, e = rand.Read(b); e != nil {
		return "", nil, e
	}
	path := filepath.Join(dir, "runtime-"+hex.EncodeToString(b)+".json")
	s, e := windows.UTF16PtrFromString(path)
	if e != nil {
		return "", nil, e
	}
	h, e := windows.CreateFile(s, windows.GENERIC_READ|windows.GENERIC_WRITE|windows.DELETE, windows.FILE_SHARE_READ|windows.FILE_SHARE_DELETE, &a, windows.CREATE_NEW, windows.FILE_ATTRIBUTE_TEMPORARY|windows.FILE_FLAG_DELETE_ON_CLOSE, 0)
	if e != nil {
		return "", nil, e
	}
	f := os.NewFile(uintptr(h), path)
	if _, e = f.Write(data); e != nil {
		f.Close()
		return "", nil, e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return "", nil, e
	}
	return path, func() { f.Close() }, nil
}
