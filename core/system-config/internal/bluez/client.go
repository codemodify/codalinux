// Package bluez talks to BlueZ over the system D-Bus.
// bluetoothctl text is a fallback in applyexec/reportprobe when this fails.
package bluez

import (
	"fmt"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

const (
	busName      = "org.bluez"
	objManager   = "org.freedesktop.DBus.ObjectManager"
	propsIface   = "org.freedesktop.DBus.Properties"
	adapterIface = "org.bluez.Adapter1"
	deviceIface  = "org.bluez.Device1"
	agentMgr     = "org.bluez.AgentManager1"
	agentIface   = "org.bluez.Agent1"
	agentPath    = "/com/codalinux/systemconfig/agent"
)

type objects map[dbus.ObjectPath]map[string]map[string]dbus.Variant

func systemBus() (*dbus.Conn, error) {
	return dbus.SystemBus()
}

func managed(conn *dbus.Conn) (objects, error) {
	var objs objects
	err := conn.Object(busName, "/").Call(objManager+".GetManagedObjects", 0).Store(&objs)
	return objs, err
}

// Collect adapters and devices. Returns ok=false if BlueZ is missing.
func Collect() (protocol.BluetoothModel, bool) {
	conn, err := systemBus()
	if err != nil {
		return protocol.BluetoothModel{}, false
	}
	objs, err := managed(conn)
	if err != nil {
		return protocol.BluetoothModel{}, false
	}
	m := protocol.BluetoothModel{}
	for path, ifaces := range objs {
		if ad, ok := ifaces[adapterIface]; ok {
			if m.Adapter == "" {
				if addr, _ := ad["Address"].Value().(string); addr != "" {
					m.Adapter = addr
				} else {
					m.Adapter = string(path)
				}
			}
			if p, _ := ad["Powered"].Value().(bool); p {
				m.Powered = true
			}
			if d, _ := ad["Discovering"].Value().(bool); d {
				m.Scanning = true
			}
		}
	}
	for _, ifaces := range objs {
		dev, ok := ifaces[deviceIface]
		if !ok {
			continue
		}
		addr, _ := dev["Address"].Value().(string)
		if addr == "" {
			continue
		}
		name, _ := dev["Name"].Value().(string)
		if name == "" {
			name, _ = dev["Alias"].Value().(string)
		}
		paired, _ := dev["Paired"].Value().(bool)
		connected, _ := dev["Connected"].Value().(bool)
		trusted, _ := dev["Trusted"].Value().(bool)
		m.Devices = append(m.Devices, protocol.BTDevice{
			Address: addr, Name: name, Paired: paired, Connected: connected, Trusted: trusted,
		})
	}
	return m, m.Adapter != "" || len(m.Devices) > 0
}

func setAdapter(prop string, value any) error {
	conn, err := systemBus()
	if err != nil {
		return err
	}
	path, err := firstAdapter(conn)
	if err != nil {
		return err
	}
	return conn.Object(busName, path).Call(propsIface+".Set", 0, adapterIface, prop, dbus.MakeVariant(value)).Err
}

func firstAdapter(conn *dbus.Conn) (dbus.ObjectPath, error) {
	objs, err := managed(conn)
	if err != nil {
		return "", err
	}
	for path, ifaces := range objs {
		if _, ok := ifaces[adapterIface]; ok {
			return path, nil
		}
	}
	return "", fmt.Errorf("no bluez adapter")
}

func devicePath(conn *dbus.Conn, addr string) (dbus.ObjectPath, error) {
	want := strings.ToUpper(addr)
	objs, err := managed(conn)
	if err != nil {
		return "", err
	}
	for path, ifaces := range objs {
		dev, ok := ifaces[deviceIface]
		if !ok {
			continue
		}
		a, _ := dev["Address"].Value().(string)
		if strings.EqualFold(a, want) {
			return path, nil
		}
	}
	ad, err := firstAdapter(conn)
	if err != nil {
		return "", err
	}
	// BlueZ well-known path: /org/bluez/hci0/dev_AA_BB_CC_DD_EE_FF
	p := dbus.ObjectPath(string(ad) + "/dev_" + strings.ReplaceAll(want, ":", "_"))
	return p, nil
}

func SetPowered(on bool) error { return setAdapter("Powered", on) }

func SetDiscovering(on bool) error {
	conn, err := systemBus()
	if err != nil {
		return err
	}
	path, err := firstAdapter(conn)
	if err != nil {
		return err
	}
	obj := conn.Object(busName, path)
	method := adapterIface + ".StopDiscovery"
	if on {
		method = adapterIface + ".StartDiscovery"
	}
	return obj.Call(method, 0).Err
}

func Pair(addr, pin string) error {
	conn, err := systemBus()
	if err != nil {
		return err
	}
	if err := registerAgent(conn, pin); err != nil {
		// pairing can still work for already-known devices
		_ = err
	}
	path, err := devicePath(conn, addr)
	if err != nil {
		return err
	}
	call := conn.Object(busName, path).Call(deviceIface+".Pair", 0)
	if call.Err != nil {
		return call.Err
	}
	return nil
}

func Connect(addr string) error    { return deviceCall(addr, "Connect") }
func Disconnect(addr string) error { return deviceCall(addr, "Disconnect") }

func Trust(addr string, on bool) error {
	conn, err := systemBus()
	if err != nil {
		return err
	}
	path, err := devicePath(conn, addr)
	if err != nil {
		return err
	}
	return conn.Object(busName, path).Call(propsIface+".Set", 0, deviceIface, "Trusted", dbus.MakeVariant(on)).Err
}

func deviceCall(addr, method string) error {
	conn, err := systemBus()
	if err != nil {
		return err
	}
	path, err := devicePath(conn, addr)
	if err != nil {
		return err
	}
	return conn.Object(busName, path).Call(deviceIface+"."+method, 0).Err
}

type agent struct {
	pin string
}

func (a *agent) Release() *dbus.Error { return nil }

func (a *agent) RequestPinCode(_ dbus.ObjectPath) (string, *dbus.Error) {
	if a.pin != "" {
		return a.pin, nil
	}
	return "0000", nil
}

func (a *agent) DisplayPinCode(_ dbus.ObjectPath, _ string) *dbus.Error { return nil }

func (a *agent) RequestPasskey(_ dbus.ObjectPath) (uint32, *dbus.Error) {
	if a.pin != "" {
		var n uint32
		for _, r := range a.pin {
			if r >= '0' && r <= '9' {
				n = n*10 + uint32(r-'0')
			}
		}
		return n, nil
	}
	return 0, nil
}

func (a *agent) DisplayPasskey(_ dbus.ObjectPath, _ uint32, _ uint16) *dbus.Error {
	return nil
}

func (a *agent) RequestConfirmation(_ dbus.ObjectPath, _ uint32) *dbus.Error {
	return nil
}

func (a *agent) RequestAuthorization(_ dbus.ObjectPath) *dbus.Error { return nil }

func (a *agent) AuthorizeService(_ dbus.ObjectPath, _ string) *dbus.Error { return nil }

func (a *agent) Cancel() *dbus.Error { return nil }

func registerAgent(conn *dbus.Conn, pin string) error {
	a := &agent{pin: pin}
	if err := conn.Export(a, agentPath, agentIface); err != nil {
		return err
	}
	mgr := conn.Object(busName, "/org/bluez")
	if err := mgr.Call(agentMgr+".RegisterAgent", 0, dbus.ObjectPath(agentPath), "KeyboardDisplay").Err; err != nil {
		if !strings.Contains(err.Error(), "AlreadyExists") {
			return err
		}
	}
	_ = mgr.Call(agentMgr+".RequestDefaultAgent", 0, dbus.ObjectPath(agentPath))
	// keep the connection + export alive for the pairing call
	time.Sleep(50 * time.Millisecond)
	return nil
}
