package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/codemodify/codalinux/core/system-config/internal/client"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

const (
	keyUp = iota + 1000
	keyDown
	keyLeft
	keyRight
	keyEnter
	keyEsc
	keyBack
	keyQuit
)

func (s *session) runUI() error {
	in := int(os.Stdin.Fd())
	restore, err := makeRaw(in)
	if err != nil {
		return fmt.Errorf("raw terminal: %w (use -dump)", err)
	}
	defer restore()
	fmt.Fprint(os.Stdout, "\x1b[?1049h\x1b[?25l")
	defer fmt.Fprint(os.Stdout, "\x1b[?25h\x1b[?1049l")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	keys := make(chan int, 8)
	go readKeys(os.Stdin, keys)
	events := make(chan protocol.Response, 8)
	go s.watchLoop(ctx, events)

	for {
		s.draw()
		select {
		case ev := <-events:
			s.onWatch(ev)
		case k, ok := <-keys:
			if !ok {
				return nil
			}
			if !s.handleKey(k) {
				return nil
			}
		}
	}
}

func (s *session) watchLoop(ctx context.Context, out chan<- protocol.Response) {
	c, err := client.Dial(sockpath.Daemon())
	if err != nil {
		return
	}
	defer c.Close()
	if _, err := c.WatchFollow(""); err != nil {
		return
	}
	for {
		ev, err := c.Next(ctx)
		if err != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case out <- ev:
		}
	}
}

func (s *session) onWatch(ev protocol.Response) {
	if ev.Path == "" || ev.Note == protocol.WatchNoteFollow {
		return
	}
	if s.dirtyPath(ev.Path) {
		s.status = ev.Path + " observed update (dirty — r to reload)"
		return
	}
	s.applyObserved(ev.Path, ev.Observed)
	s.snapshot(ev.Path)
	s.status = ev.Path + " updated"
}

func (s *session) handleKey(k int) bool {
	if s.help {
		s.help = false
		return true
	}
	if s.edit {
		return s.handleEdit(k)
	}
	switch k {
	case 'q', 'Q', keyQuit, 3:
		return false
	case '?':
		s.help = true
	case 'r', 'R':
		s.reloadPath(s.path())
		s.status = "refreshed " + s.path()
	case 'a', 'A':
		if err := s.applyPath(s.path()); err != nil {
			s.status = err.Error()
		} else {
			s.status = "applied " + s.path()
		}
	case 'u', 'U':
		s.reloadPath(s.path())
		s.status = "reverted " + s.path()
	case keyUp, 'k', 'K':
		s.move(-1)
	case keyDown, 'j', 'J':
		s.move(1)
	case keyLeft, 'h', 'H':
		s.pane = 0
	case keyRight, 'l', 'L':
		if len(s.page().edits) > 0 {
			s.pane = 1
		}
	case keyEnter, '\r', '\n':
		s.activate()
	case ' ':
		s.toggle()
	}
	return true
}

func (s *session) move(delta int) {
	if s.pane == 0 {
		s.idx = (s.idx + delta + len(pages)) % len(pages)
		s.field = 0
		return
	}
	n := len(s.page().edits)
	if n == 0 {
		s.pane = 0
		return
	}
	s.field = (s.field + delta + n) % n
}

func (s *session) activate() {
	if s.pane == 0 {
		if len(s.page().edits) > 0 {
			s.pane = 1
		}
		return
	}
	edits := s.page().edits
	if s.field < 0 || s.field >= len(edits) {
		return
	}
	f := edits[s.field]
	switch f.kind {
	case fieldAction:
		if f.act != nil {
			f.act()
			s.status = "staged " + f.label
		}
	case fieldBool:
		s.toggle()
	case fieldText, fieldFloat, fieldPick:
		s.edit = true
		if f.get != nil {
			s.buf = f.get()
		} else {
			s.buf = ""
		}
	}
}

func (s *session) toggle() {
	if s.pane != 1 {
		return
	}
	edits := s.page().edits
	if s.field < 0 || s.field >= len(edits) {
		return
	}
	f := edits[s.field]
	if f.toggle != nil {
		f.toggle()
		s.status = "toggled " + f.label
	}
}

func (s *session) handleEdit(k int) bool {
	switch k {
	case keyEsc, 3:
		s.edit = false
		s.buf = ""
		s.status = "edit cancelled"
	case keyEnter, '\r', '\n':
		edits := s.page().edits
		if s.field >= 0 && s.field < len(edits) && edits[s.field].set != nil {
			edits[s.field].set(s.buf)
			s.status = "edited " + edits[s.field].label
		}
		s.edit = false
		s.buf = ""
	case keyBack, 127, 8:
		if s.buf != "" {
			_, n := utf8.DecodeLastRuneInString(s.buf)
			s.buf = s.buf[:len(s.buf)-n]
		}
	default:
		if k >= 32 && k < 127 {
			s.buf += string(rune(k))
		}
	}
	return true
}

func (s *session) draw() {
	cols, rows := winsize(int(os.Stdout.Fd()))
	if cols < 60 {
		cols = 60
	}
	if rows < 16 {
		rows = 16
	}
	var b strings.Builder
	b.WriteString("\x1b[H\x1b[J")
	title := " Coda Settings  (system-config-tui → D only) "
	b.WriteString(reverse(pad(title, cols)) + "\n")

	if s.help {
		b.WriteString(s.helpText(cols, rows-3))
		b.WriteString(s.footer(cols))
		fmt.Fprint(os.Stdout, b.String())
		return
	}

	sideW := 18
	pg := s.page()
	bodyRows := rows - 4
	side := s.sidebarLines()
	right := s.pageLines(pg, cols-sideW-3)
	for i := 0; i < bodyRows; i++ {
		sl, rl := "", ""
		if i < len(side) {
			sl = side[i]
		}
		if i < len(right) {
			rl = right[i]
		}
		b.WriteString(fit(sl, sideW))
		b.WriteString(" │ ")
		b.WriteString(fit(rl, cols-sideW-3))
		b.WriteByte('\n')
	}
	applyHint := "Apply off"
	if s.dirtyPath(s.path()) {
		applyHint = "Apply ready (a)"
	} else if !protocol.Settable(s.path()) {
		applyHint = "observe-only"
	}
	b.WriteString(strings.Repeat("─", cols) + "\n")
	b.WriteString(s.footerLine(cols, applyHint))
	fmt.Fprint(os.Stdout, b.String())
}

func (s *session) sidebarLines() []string {
	out := make([]string, 0, len(pages)+1)
	out = append(out, "Sections")
	for i, p := range pages {
		mark := "  "
		if i == s.idx {
			if s.pane == 0 {
				mark = "> "
			} else {
				mark = "· "
			}
		}
		star := ""
		if s.dirtyPath(p.Path) {
			star = "*"
		}
		out = append(out, mark+p.Label+star)
	}
	return out
}

func (s *session) pageLines(pg pageView, width int) []string {
	var out []string
	out = append(out, pg.title)
	if pg.help != "" {
		out = append(out, pg.help)
	}
	out = append(out, pg.info...)
	if len(pg.edits) == 0 {
		out = append(out, "", "Refresh only (r).")
		return out
	}
	out = append(out, "", "Fields")
	for i, f := range pg.edits {
		cur := ""
		if f.get != nil {
			cur = f.get()
		}
		prefix := "  "
		if s.pane == 1 && i == s.field {
			prefix = "> "
			if s.edit {
				cur = s.buf + "█"
			}
		}
		line := fmt.Sprintf("%s%-16s %s", prefix, f.label, cur)
		if f.hint != "" && s.pane == 1 && i == s.field && !s.edit {
			line += "   (" + f.hint + ")"
		}
		out = append(out, fit(line, width))
	}
	return out
}

func (s *session) helpText(cols, rows int) string {
	lines := []string{
		"",
		"  Every KnownPath is a page. Clients talk to system-configd only.",
		"  Apply is per section (dirty vs that page’s baseline), like the GUI.",
		"",
		"  ↑↓ j k     move in sidebar or fields",
		"  ←→ h l     sidebar ↔ fields",
		"  Enter      edit text / stage action",
		"  Space      toggle bool",
		"  r          refresh this path (and devices extras)",
		"  a          apply this path if dirty (D allowlist)",
		"  u          revert this path to last refresh/apply",
		"  q          quit",
		"",
		"  watch follow updates observed live unless the section is dirty.",
	}
	for len(lines) < rows {
		lines = append(lines, "")
	}
	for i := range lines {
		lines[i] = fit(lines[i], cols)
	}
	return strings.Join(lines, "\n") + "\n"
}

func (s *session) footer(cols int) string {
	return s.footerLine(cols, "")
}

func (s *session) footerLine(cols int, extra string) string {
	msg := "q quit  ? help  r refresh  a apply  u revert  " + extra + "  " + s.status
	return reverse(fit(msg, cols))
}

func readKeys(r io.Reader, out chan<- int) {
	br := bufio.NewReader(r)
	for {
		b, err := br.ReadByte()
		if err != nil {
			close(out)
			return
		}
		if b != 27 {
			out <- int(b)
			continue
		}
		next, err := br.ReadByte()
		if err != nil {
			out <- keyEsc
			continue
		}
		if next != '[' {
			out <- keyEsc
			continue
		}
		code, err := br.ReadByte()
		if err != nil {
			out <- keyEsc
			continue
		}
		switch code {
		case 'A':
			out <- keyUp
		case 'B':
			out <- keyDown
		case 'C':
			out <- keyRight
		case 'D':
			out <- keyLeft
		default:
			out <- keyEsc
		}
	}
}

func pad(s string, n int) string {
	if utf8.RuneCountInString(s) >= n {
		return fit(s, n)
	}
	return s + strings.Repeat(" ", n-utf8.RuneCountInString(s))
}

func fit(s string, n int) string {
	if n <= 0 {
		return ""
	}
	rs := []rune(s)
	if len(rs) > n {
		return string(rs[:n])
	}
	return string(rs) + strings.Repeat(" ", n-len(rs))
}

func reverse(s string) string { return "\x1b[7m" + s + "\x1b[0m" }
