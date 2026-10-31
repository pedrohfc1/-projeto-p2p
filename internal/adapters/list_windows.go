package adapters

import (
	"errors"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func driverIDs() map[string]string {
	out := map[string]string{}
	key, e := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Class\{4d36e972-e325-11ce-bfc1-08002be10318}`, registry.READ)
	if e != nil {
		return out
	}
	defer key.Close()
	names, e := key.ReadSubKeyNames(-1)
	if e != nil {
		return out
	}
	for _, name := range names {
		k, e := registry.OpenKey(key, name, registry.READ)
		if e != nil {
			continue
		}
		id, _, _ := k.GetStringValue("NetCfgInstanceId")
		parts := []string{}
		for _, field := range []string{"ComponentId", "MatchingDeviceId", "DriverDesc"} {
			v, _, _ := k.GetStringValue(field)
			parts = append(parts, v)
		}
		k.Close()
		if id != "" {
			out[strings.ToLower(id)] = strings.Join(parts, " ")
		}
	}
	return out
}
func List() ([]Info, error) {
	items, e := basic()
	if e != nil {
		return nil, e
	}
	size := uint32(15000)
	var buf []byte
	var head *windows.IpAdapterAddresses
	for attempt := 0; attempt < 4; attempt++ {
		if size > 8<<20 {
			return nil, errors.New("lista de adaptadores excedeu limite")
		}
		buf = make([]byte, size)
		head = (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
		e = windows.GetAdaptersAddresses(windows.AF_INET, windows.GAA_FLAG_INCLUDE_ALL_INTERFACES, 0, head, &size)
		if e == windows.ERROR_BUFFER_OVERFLOW {
			continue
		}
		break
	}
	if e != nil {
		return nil, e
	}
	drivers := driverIDs()
	for a := head; a != nil; a = a.Next {
		found := false
		for n := range items {
			if items[n].Index == int(a.IfIndex) {
				found = true
				v := &items[n]
				v.ID = strings.ToLower(windows.BytePtrToString(a.AdapterName))
				v.Name = windows.UTF16PtrToString(a.FriendlyName)
				v.Description = windows.UTF16PtrToString(a.Description)
				v.Up = a.OperStatus == windows.IfOperStatusUp
				v.DriverID = drivers[v.ID]
			}
		}
		if !found && a.IfIndex != 0 {
			id := strings.ToLower(windows.BytePtrToString(a.AdapterName))
			items = append(items, Info{ID: id, Index: int(a.IfIndex), Name: windows.UTF16PtrToString(a.FriendlyName), Description: windows.UTF16PtrToString(a.Description), Up: a.OperStatus == windows.IfOperStatusUp, IPv4: []string{}, DriverID: drivers[id]})
		}
	}
	for n := range items {
		items[n] = Classify(items[n])
	}
	return items, nil
}
