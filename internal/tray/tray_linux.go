//go:build linux

package tray

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
	"image"
	_ "image/png"
	"os"
	"sync"
)

type iconPixmap struct {
	Width, Height int32
	Data          []byte
}

func readPixmap(path string) []iconPixmap {
	data, e := os.ReadFile(path)
	if e != nil || len(data) < 22 {
		return nil
	}
	count := int(binary.LittleEndian.Uint16(data[4:]))
	for i := 0; i < count; i++ {
		entry := 6 + 16*i
		if entry+16 > len(data) {
			break
		}
		if data[entry] != 32 {
			continue
		}
		size := int(binary.LittleEndian.Uint32(data[entry+8:]))
		offset := int(binary.LittleEndian.Uint32(data[entry+12:]))
		if offset < 0 || size < 0 || offset+size > len(data) {
			return nil
		}
		img, _, e := image.Decode(bytes.NewReader(data[offset : offset+size]))
		if e != nil {
			return nil
		}
		b := img.Bounds()
		pixels := make([]byte, 0, b.Dx()*b.Dy()*4)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r, g, bl, a := img.At(x, y).RGBA()
				pixels = append(pixels, byte(a>>8), byte(r>>8), byte(g>>8), byte(bl>>8))
			}
		}
		return []iconPixmap{{int32(b.Dx()), int32(b.Dy()), pixels}}
	}
	return nil
}

const itemInterface = "org.kde.StatusNotifierItem"

type Tray struct {
	conn          *dbus.Conn
	props         *prop.Properties
	cb            Callbacks
	mu            sync.Mutex
	state         State
	normal, muted []iconPixmap
}
type menuLayout struct {
	ID         int32
	Properties map[string]dbus.Variant
	Children   []dbus.Variant
}

func Start(normal, muted string, cb Callbacks) (*Tray, error) {
	conn, e := dbus.ConnectSessionBus()
	if e != nil {
		return nil, e
	}
	fail := func(e error) (*Tray, error) { conn.Close(); return nil, e }
	t := &Tray{conn: conn, cb: cb, normal: readPixmap(normal), muted: readPixmap(muted)}
	service := fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
	if _, e = conn.RequestName(service, dbus.NameFlagDoNotQueue); e != nil {
		return fail(e)
	}
	if e = conn.Export(t, "/StatusNotifierItem", itemInterface); e != nil {
		return fail(e)
	}
	if e = conn.Export(t, "/Menu", "com.canonical.dbusmenu"); e != nil {
		return fail(e)
	}
	values := map[string]interface{}{"Category": "Communications", "Id": "peetwopee", "Title": "PeeTwoPee", "Status": "Active", "IconName": "", "IconPixmap": t.normal, "ItemIsMenu": false, "Menu": dbus.ObjectPath("/Menu")}
	props := prop.Map{itemInterface: {}}
	for name, value := range values {
		props[itemInterface][name] = &prop.Prop{Value: value, Writable: false, Emit: prop.EmitTrue}
	}
	t.props, e = prop.Export(conn, "/StatusNotifierItem", props)
	if e != nil {
		return fail(e)
	}
	_, e = prop.Export(conn, "/Menu", prop.Map{"com.canonical.dbusmenu": {"Version": &prop.Prop{Value: uint32(3), Emit: prop.EmitConst}, "TextDirection": &prop.Prop{Value: "ltr", Emit: prop.EmitConst}, "Status": &prop.Prop{Value: "normal", Emit: prop.EmitConst}}})
	if e != nil {
		return fail(e)
	}
	if e = conn.Object("org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher").Call("org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem", 0, service).Err; e != nil {
		return fail(e)
	}
	return t, nil
}
func (t *Tray) Activate(x, y int32) *dbus.Error {
	if t.cb.Open != nil {
		go t.cb.Open()
	}
	return nil
}
func (t *Tray) SecondaryActivate(x, y int32) *dbus.Error { return t.Activate(x, y) }
func (t *Tray) ContextMenu(x, y int32) *dbus.Error       { return nil }
func (t *Tray) GetLayout(parent int32, depth int32, names []string) (uint32, menuLayout, *dbus.Error) {
	t.mu.Lock()
	s := t.state
	t.mu.Unlock()
	labels := []string{"Abrir PeeTwoPee", "Silenciar microfone", "Sair da chamada", "Encerrar aplicativo"}
	if s.Muted {
		labels[1] = "Ativar microfone"
	}
	root := menuLayout{ID: 0, Properties: map[string]dbus.Variant{"children-display": dbus.MakeVariant("submenu")}, Children: []dbus.Variant{}}
	for i, label := range labels {
		item := menuLayout{ID: int32(i + 1), Properties: map[string]dbus.Variant{"label": dbus.MakeVariant(label), "enabled": dbus.MakeVariant(i == 0 || i == 3 || s.Joined)}, Children: []dbus.Variant{}}
		root.Children = append(root.Children, dbus.MakeVariant(item))
	}
	return 1, root, nil
}
func (t *Tray) Event(id int32, event string, data dbus.Variant, timestamp uint32) *dbus.Error {
	if event != "clicked" {
		return nil
	}
	callbacks := []func(){t.cb.Open, t.cb.Mute, t.cb.Leave, t.cb.Quit}
	if id > 0 && int(id) <= len(callbacks) && callbacks[id-1] != nil {
		go callbacks[id-1]()
	}
	return nil
}
func (t *Tray) AboutToShow(id int32) (bool, *dbus.Error) { return true, nil }
func (t *Tray) Update(s State) {
	t.mu.Lock()
	changed := s != t.state
	t.state = s
	t.mu.Unlock()
	if !changed {
		return
	}
	icon := "audio-headset"
	if s.Joined && s.Muted {
		icon = "microphone-sensitivity-muted"
	}
	pixels := t.normal
	if s.Joined && s.Muted {
		pixels = t.muted
	}
	t.props.SetMust(itemInterface, "IconPixmap", pixels)
	_ = icon
	t.props.SetMust(itemInterface, "Title", s.Tooltip())
	_ = t.conn.Emit("/StatusNotifierItem", itemInterface+".NewIcon")
	_ = t.conn.Emit("/Menu", "com.canonical.dbusmenu.LayoutUpdated", uint32(1), int32(0))
}
func (t *Tray) Close() { t.conn.Close() }

type menuProperties struct {
	ID         int32
	Properties map[string]dbus.Variant
}

func (t *Tray) GetGroupProperties(ids []int32, names []string) ([]menuProperties, *dbus.Error) {
	_, root, _ := t.GetLayout(0, -1, names)
	out := []menuProperties{}
	for _, v := range root.Children {
		item := v.Value().(menuLayout)
		for _, id := range ids {
			if id == item.ID {
				out = append(out, menuProperties{item.ID, item.Properties})
			}
		}
	}
	return out, nil
}
func (t *Tray) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	values, _ := t.GetGroupProperties([]int32{id}, []string{name})
	if len(values) == 1 {
		if value, ok := values[0].Properties[name]; ok {
			return value, nil
		}
	}
	return dbus.MakeVariant(""), dbus.NewError("com.canonical.dbusmenu.Error.PropertyNotFound", []interface{}{"unknown property"})
}
func (t *Tray) Notify(title, body string) bool {
	return t.conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications").Call("org.freedesktop.Notifications.Notify", 0, "PeeTwoPee", uint32(0), "", title, body, []string{}, map[string]dbus.Variant{}, int32(4000)).Err == nil
}

func (*Tray) Available() bool { return false }
