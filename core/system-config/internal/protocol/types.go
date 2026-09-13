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
	Focused   bool    `json:"focused,omitempty"`
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

type Locale struct {
	Lang     string `json:"lang,omitempty"`
	Timezone string `json:"timezone,omitempty"`
	Keymap   string `json:"keymap,omitempty"`
}
