package reportprobe

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/rpc"
)

func (p *Probe) printers() (json.RawMessage, error) {
	m := protocol.PrintersModel{}
	raw, err := p.cmd("lpstat", "-a")
	if err != nil {
		return rpc.Raw(m), nil
	}
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pr := protocol.Printer{Name: fields[0], State: strings.Join(fields[1:], " ")}
		m.Printers = append(m.Printers, pr)
	}
	return rpc.Raw(m), nil
}

func (p *Probe) users() (json.RawMessage, error) {
	m := protocol.UsersModel{}
	b, err := os.ReadFile(p.root("etc/passwd"))
	if err != nil {
		return rpc.Raw(m), nil
	}
	for _, line := range strings.Split(string(b), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) < 7 {
			continue
		}
		uid, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}
		if uid != 0 && uid < 1000 {
			continue
		}
		shell := parts[6]
		if strings.Contains(shell, "nologin") || strings.HasSuffix(shell, "/false") {
			continue
		}
		gid, _ := strconv.Atoi(parts[3])
		m.Users = append(m.Users, protocol.LocalUser{
			Name: parts[0], UID: uid, GID: gid, Home: parts[5], Shell: shell,
		})
	}
	return rpc.Raw(m), nil
}

func (p *Probe) storage() (json.RawMessage, error) {
	m := protocol.StorageModel{}
	if raw, err := p.cmd("lsblk", "-J", "-o", "NAME,TYPE,SIZE,MOUNTPOINT,MODEL,FSTYPE"); err == nil {
		if parseLsblk(raw, &m) {
			return rpc.Raw(m), nil
		}
	}
	if raw, err := p.cmd("udisksctl", "status"); err == nil && strings.TrimSpace(raw) != "" {
		for _, line := range strings.Split(raw, "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 || strings.EqualFold(fields[0], "MODEL") {
				continue
			}
			m.Block = append(m.Block, protocol.BlockDev{Model: fields[0], Name: fields[len(fields)-1]})
		}
		if len(m.Block) > 0 {
			return rpc.Raw(m), nil
		}
	}
	return rpc.Raw(m), nil
}

func parseLsblk(raw string, m *protocol.StorageModel) bool {
	var tree struct {
		Blockdevices []lsblkNode `json:"blockdevices"`
	}
	if json.Unmarshal([]byte(raw), &tree) != nil {
		return false
	}
	var walk func(lsblkNode)
	walk = func(n lsblkNode) {
		if n.Type != "loop" && n.Type != "ram" && n.Name != "" {
			m.Block = append(m.Block, protocol.BlockDev{
				Name: n.Name, Type: n.Type, Size: n.Size, Mount: n.Mount, Model: n.Model, FS: n.FS,
			})
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	for _, n := range tree.Blockdevices {
		walk(n)
	}
	return true
}

type lsblkNode struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Size     string      `json:"size"`
	Mount    string      `json:"mountpoint"`
	Model    string      `json:"model"`
	FS       string      `json:"fstype"`
	Children []lsblkNode `json:"children"`
}
