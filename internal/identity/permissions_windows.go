package identity

import (
	"errors"
	"golang.org/x/sys/windows"
	"unsafe"
)

// DPAPI protects private keys with the Windows user profile. ACLs are inherited.
func protect(path string, directory bool) error { return nil }
func seal(b []byte) ([]byte, error)             { return dpapi(b, true) }
func unseal(b []byte) ([]byte, error)           { return dpapi(b, false) }
func dpapi(b []byte, encrypt bool) ([]byte, error) {
	if len(b) == 0 {
		return nil, errors.New("empty identity")
	}
	in := windows.DataBlob{Size: uint32(len(b)), Data: &b[0]}
	var out windows.DataBlob
	var err error
	if encrypt {
		err = windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	} else {
		err = windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	}
	if err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, int(out.Size))...), nil
}
