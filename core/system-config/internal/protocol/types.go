package protocol

// DisplayModel is desired or observed display state.
type DisplayModel struct {
	Outputs []Output `json:"outputs"`
}

type Output struct {
	Name      string  `json:"name"`
	Width     int     `json:"width,omitempty"`
	Height    int     `json:"height,omitempty"`
	RefreshHz int     `json:"refresh_hz,omitempty"`
	Scale     float64 `json:"scale,omitempty"`
	Mode      string  `json:"mode,omitempty"`
	X         int     `json:"x,omitempty"`
	Y         int     `json:"y,omitempty"`
	Position  string  `json:"position,omitempty"` // auto | WxH (hypr "0x0")
	Focused   bool    `json:"focused,omitempty"`
}

type NetworkModel struct {
	Links    []NetLink `json:"links"`
	WiFi     WiFiState `json:"wifi"`
	Routes   []Route   `json:"routes,omitempty"`
	Airplane bool      `json:"airplane"`
}

type NetLink struct {
	Name      string   `json:"name"`
	Type      string   `json:"type,omitempty"`
	OperState string   `json:"oper_state,omitempty"`
	Enabled   bool     `json:"enabled"`
	Method    string   `json:"method,omitempty"` // dhcp | static
	Addresses []string `json:"addresses,omitempty"`
	Gateway   string   `json:"gateway,omitempty"`
	DNS       []string `json:"dns,omitempty"`
	Search    []string `json:"search,omitempty"`
}

type WiFiState struct {
	Device     string   `json:"device,omitempty"`
	Connected  string   `json:"connected,omitempty"`
	Connect    string   `json:"connect,omitempty"`
	Disconnect bool     `json:"disconnect,omitempty"`
	PSK        string   `json:"psk,omitempty"`
	Hidden     bool     `json:"hidden,omitempty"`
	Scanning   bool     `json:"scanning,omitempty"`
	Networks   []SSID   `json:"networks,omitempty"`
	Known      []string `json:"known,omitempty"`
}

type SSID struct {
	SSID     string `json:"ssid"`
	Signal   int    `json:"signal,omitempty"`
	Security string `json:"security,omitempty"`
}

type Route struct {
	Dst     string `json:"dst,omitempty"`
	Gateway string `json:"gateway,omitempty"`
	Dev     string `json:"dev,omitempty"`
}

type AudioModel struct {
	DefaultSink   string      `json:"default_sink,omitempty"`
	DefaultSource string      `json:"default_source,omitempty"`
	Volume        float64     `json:"volume,omitempty"`
	Mute          *bool       `json:"mute,omitempty"`
	Sinks         []AudioNode `json:"sinks,omitempty"`
	Sources       []AudioNode `json:"sources,omitempty"`
}

type AudioNode struct {
	ID      string  `json:"id"`
	Name    string  `json:"name,omitempty"`
	Volume  float64 `json:"volume,omitempty"`
	Mute    bool    `json:"mute,omitempty"`
	Default bool    `json:"default,omitempty"`
}

type BluetoothModel struct {
	Powered    bool       `json:"powered"`
	Scanning   bool       `json:"scanning,omitempty"`
	Adapter    string     `json:"adapter,omitempty"`
	Devices    []BTDevice `json:"devices,omitempty"`
	Pair       []string   `json:"pair,omitempty"`
	Connect    []string   `json:"connect,omitempty"`
	Disconnect []string   `json:"disconnect,omitempty"`
	Trust      []string   `json:"trust,omitempty"`
	PIN        string     `json:"pin,omitempty"`
}

type BTDevice struct {
	Address   string `json:"address"`
	Name      string `json:"name,omitempty"`
	Paired    bool   `json:"paired,omitempty"`
	Connected bool   `json:"connected,omitempty"`
	Trusted   bool   `json:"trusted,omitempty"`
}

type InputModel struct {
	KBLayout      string  `json:"kb_layout,omitempty"`
	Keymap        string  `json:"keymap,omitempty"`
	PointerSpeed  float64 `json:"pointer_speed"`
	NaturalScroll bool    `json:"natural_scroll"`
	TapToClick    bool    `json:"tap_to_click"`
}

type DateTimeModel struct {
	Timezone string `json:"timezone,omitempty"`
	NTP      bool   `json:"ntp"`
	Time     string `json:"time,omitempty"`
	RTCLocal bool   `json:"rtc_local,omitempty"`
}

type Locale struct {
	Lang     string `json:"lang,omitempty"`
	Timezone string `json:"timezone,omitempty"`
	Keymap   string `json:"keymap,omitempty"`
}

type DevicesSummary struct {
	Product  string `json:"product,omitempty"`
	Vendor   string `json:"vendor,omitempty"`
	Board    string `json:"board,omitempty"`
	PCICount int    `json:"pci_count"`
	USBCount int    `json:"usb_count"`
}

type PCIDevice struct {
	ID     string `json:"id"`
	Vendor string `json:"vendor,omitempty"`
	Device string `json:"device,omitempty"`
	Class  string `json:"class,omitempty"`
}

type PCIList struct {
	Devices []PCIDevice `json:"devices"`
}

type USBDevice struct {
	ID           string `json:"id"`
	Vendor       string `json:"vendor,omitempty"`
	Product      string `json:"product,omitempty"`
	Manufacturer string `json:"manufacturer,omitempty"`
}

type USBList struct {
	Devices []USBDevice `json:"devices"`
}

type DMI struct {
	Vendor      string `json:"vendor,omitempty"`
	Product     string `json:"product,omitempty"`
	Version     string `json:"version,omitempty"`
	Board       string `json:"board,omitempty"`
	BiosDate    string `json:"bios_date,omitempty"`
	BiosVersion string `json:"bios_version,omitempty"`
	Chassis     string `json:"chassis,omitempty"`
}

type SessionModel struct {
	Sessions    []LoginSession `json:"sessions"`
	Seats       []Seat         `json:"seats,omitempty"`
	IdleInhibit []Inhibit      `json:"idle_inhibit,omitempty"`
	IdleHint    bool           `json:"idle_hint,omitempty"`
	Action      string         `json:"action,omitempty"` // lock
}

type Seat struct {
	ID       string   `json:"id"`
	Sessions []string `json:"sessions,omitempty"`
}

type Inhibit struct {
	Who  string `json:"who,omitempty"`
	Why  string `json:"why,omitempty"`
	Mode string `json:"mode,omitempty"`
	UID  int    `json:"uid,omitempty"`
}

type LoginSession struct {
	ID    string `json:"id"`
	UID   int    `json:"uid,omitempty"`
	User  string `json:"user,omitempty"`
	Seat  string `json:"seat,omitempty"`
	TTY   string `json:"tty,omitempty"`
	Type  string `json:"type,omitempty"`
	Class string `json:"class,omitempty"`
	State string `json:"state,omitempty"`
}

type PowerModel struct {
	Action        string `json:"action,omitempty"` // suspend | hibernate
	Brightness    int    `json:"brightness,omitempty"`
	MaxBrightness int    `json:"max_brightness,omitempty"`
	Backlight     string `json:"backlight,omitempty"`
	Lid           string `json:"lid,omitempty"` // ignore | suspend | lock | poweroff
	CanSuspend    bool   `json:"can_suspend"`
	CanHibernate  bool   `json:"can_hibernate"`
}

type PrintersModel struct {
	Printers []Printer `json:"printers"`
	Default  string    `json:"default,omitempty"`
}

type Printer struct {
	Name      string `json:"name"`
	URI       string `json:"uri,omitempty"`
	State     string `json:"state,omitempty"`
	Enabled   *bool  `json:"enabled,omitempty"`
	Accepting *bool  `json:"accepting,omitempty"`
	Default   bool   `json:"default,omitempty"`
}

type UsersModel struct {
	Users []LocalUser `json:"users"`
}

type LocalUser struct {
	Name  string `json:"name"`
	UID   int    `json:"uid"`
	GID   int    `json:"gid,omitempty"`
	Home  string `json:"home,omitempty"`
	Shell string `json:"shell,omitempty"`
}

type StorageModel struct {
	Block []BlockDev `json:"block"`
}

type BlockDev struct {
	Name   string `json:"name"`
	Type   string `json:"type,omitempty"`
	Size   string `json:"size,omitempty"`
	Mount  string `json:"mount,omitempty"`
	Model  string `json:"model,omitempty"`
	FS     string `json:"fstype,omitempty"`
	Action string `json:"action,omitempty"` // mount | unmount
}
