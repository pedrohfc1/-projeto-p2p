// Build-time resource writer for our unsigned installer executables only.
package main

import (
	"encoding/binary"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
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
	put := func(kind, id uintptr, data []byte) {
		if len(data) == 0 {
			panic("empty resource")
		}
		ok, _, e := dll.NewProc("UpdateResourceW").Call(h, kind, id, 0, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)))
		if ok == 0 {
			dll.NewProc("EndUpdateResourceW").Call(h, 1)
			panic(e)
		}
	}
	icon, e := os.ReadFile("assets/brand/icon.ico")
	if e != nil {
		panic(e)
	}
	if len(icon) < 6 {
		panic("bad icon")
	}
	count := int(binary.LittleEndian.Uint16(icon[4:]))
	if count < 1 || count > 32 || len(icon) < 6+count*16 {
		panic("bad icon directory")
	}
	group := make([]byte, 6+count*14)
	copy(group, icon[:6])
	for i := 0; i < count; i++ {
		at := 6 + i*16
		n := int(binary.LittleEndian.Uint32(icon[at+8:]))
		offset := int(binary.LittleEndian.Uint32(icon[at+12:]))
		if offset < 0 || n < 1 || offset+n > len(icon) {
			panic("bad icon image")
		}
		put(3, uintptr(100+i), icon[offset:offset+n])
		entry := 6 + i*14
		copy(group[entry:entry+12], icon[at:at+12])
		binary.LittleEndian.PutUint16(group[entry+12:], uint16(100+i))
	}
	put(14, 1, group)
	fixed := make([]byte, 52)
	for i, value := range []uint32{0xfeef04bd, 0x10000, 10, 196608, 10, 196608, 0x3f, 0, 0x40004, 1, 0, 0, 0} {
		binary.LittleEndian.PutUint32(fixed[i*4:], value)
	}
	strings := [][]byte{}
	for _, field := range [][2]string{{"CompanyName", "PeeTwoPee"}, {"FileDescription", "PeeTwoPee"}, {"FileVersion", "0.11.0.0"}, {"InternalName", "PeeTwoPee"}, {"OriginalFilename", filepath.Base(os.Args[1])}, {"ProductName", "PeeTwoPee"}, {"ProductVersion", "0.11.0.0"}} {
		value := utf16Bytes(field[1])
		strings = append(strings, resourceBlock(field[0], value, uint16(len(value)/2), 1))
	}
	table := resourceBlock("040904B0", nil, 0, 1, strings...)
	stringInfo := resourceBlock("StringFileInfo", nil, 0, 1, table)
	translation := resourceBlock("Translation", []byte{9, 4, 0xb0, 4}, 4, 0)
	varInfo := resourceBlock("VarFileInfo", nil, 0, 1, translation)
	put(16, 1, resourceBlock("VS_VERSION_INFO", fixed, 52, 0, stringInfo, varInfo))
	manifest := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0"><assemblyIdentity version="0.11.0.0" processorArchitecture="amd64" name="PeeTwoPee.Setup" type="win32"/><trustInfo xmlns="urn:schemas-microsoft-com:asm.v3"><security><requestedPrivileges><requestedExecutionLevel level="asInvoker" uiAccess="false"/></requestedPrivileges></security></trustInfo></assembly>`)
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

func utf16Bytes(s string) []byte {
	v, e := windows.UTF16FromString(s)
	if e != nil {
		panic(e)
	}
	b := make([]byte, len(v)*2)
	for i, x := range v {
		binary.LittleEndian.PutUint16(b[i*2:], x)
	}
	return b
}
func resourceBlock(key string, value []byte, valueLength, kind uint16, children ...[]byte) []byte {
	b := make([]byte, 6)
	binary.LittleEndian.PutUint16(b[2:], valueLength)
	binary.LittleEndian.PutUint16(b[4:], kind)
	b = append(b, utf16Bytes(key)...)
	for len(b)%4 != 0 {
		b = append(b, 0)
	}
	b = append(b, value...)
	for _, child := range children {
		for len(b)%4 != 0 {
			b = append(b, 0)
		}
		b = append(b, child...)
	}
	binary.LittleEndian.PutUint16(b, uint16(len(b)))
	return b
}
