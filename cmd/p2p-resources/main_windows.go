// Build-time resource writer for our unsigned installer executables only.
package main

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"unsafe"
)

func main() {
	if len(os.Args) != 2 {
		panic("expected installer executable")
	}
	path, e := windows.UTF16PtrFromString(os.Args[1])
	if e != nil {
		panic(e)
	}
	dll := windows.NewLazySystemDLL("kernel32.dll")
	h, _, e := dll.NewProc("BeginUpdateResourceW").Call(uintptr(unsafe.Pointer(path)), 0)
	if h == 0 {
		panic(e)
	}
	manifest := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0"><assemblyIdentity version="0.4.1.0" processorArchitecture="amd64" name="ProjetoP2P.Setup" type="win32"/><trustInfo xmlns="urn:schemas-microsoft-com:asm.v3"><security><requestedPrivileges><requestedExecutionLevel level="asInvoker" uiAccess="false"/></requestedPrivileges></security></trustInfo></assembly>`)
	ok, _, e := dll.NewProc("UpdateResourceW").Call(h, 24, 1, 0, uintptr(unsafe.Pointer(&manifest[0])), uintptr(len(manifest)))
	if ok == 0 {
		dll.NewProc("EndUpdateResourceW").Call(h, 1)
		panic(e)
	}
	ok, _, e = dll.NewProc("EndUpdateResourceW").Call(h, 0)
	if ok == 0 {
		panic(e)
	}
	fmt.Println("Manifest asInvoker embedded")
}
