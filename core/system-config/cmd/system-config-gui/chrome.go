package main

import (
	"github.com/codemodify/uitoolkit"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

// prefs chrome is the composed Settings shell. uitoolkit@dev has no PrefsPage
// or NavRail (and the current pin has no ListView.Sidebar / NewForm either),
// so this is Splitter + ListView + section header + pinned per-section
// Apply/Refresh — same contract as before, clearer prefs chrome.

func (s *session) navLabel(i int) string {
	if i < 0 || i >= len(nav) {
		return ""
	}
	label := nav[i].Label
	if s.dirtyPath(nav[i].Path) {
		return label + "  •"
	}
	return label
}

func (s *session) sectionChip(path string) string {
	if s.cli == nil {
		return "system-configd not running — Apply is disabled"
	}
	if !protocol.Settable(path) {
		return "Observe only"
	}
	if s.dirtyPath(path) {
		return "Unapplied changes in this section"
	}
	return "No staged changes"
}

func (s *session) sectionHint(path string) string {
	if !protocol.Settable(path) {
		return "Refresh reloads observed hardware. This page has no Apply."
	}
	return "Apply writes this section only. Other pages keep their own staged edits."
}

func (s *session) fieldRow(label string, field uitoolkit.Component) uitoolkit.Component {
	return uitoolkit.NewRow(uitoolkit.NewLabel(label), field).WithGap(8)
}

func (s *session) prefsPage(path string, body uitoolkit.Component) uitoolkit.Component {
	meta := pageMeta(path)
	title := uitoolkit.NewTitle(meta.Label)
	group := uitoolkit.NewLabel(meta.Group)
	blurb := uitoolkit.NewLabel(meta.Blurb)
	chip := uitoolkit.NewLabel(s.sectionChip(path))
	header := uitoolkit.NewColumn(title, group, blurb, chip).WithGap(4)

	scroll := uitoolkit.NewScrollView(body)
	actions := s.pageActions(path)
	hint := uitoolkit.NewLabel(s.sectionHint(path))
	footer := uitoolkit.NewRow(actions, hint).WithGap(12)

	page := uitoolkit.NewColumn(header, uitoolkit.NewSeparator(), scroll, uitoolkit.NewSeparator(), footer).WithGap(10)
	page.AddFlex(scroll, 1)
	return page
}

func (s *session) navRail() uitoolkit.Component {
	list := uitoolkit.NewListView(len(nav), s.navLabel, func(i int) {
		if i < 0 || i >= len(nav) || i == s.page {
			return
		}
		s.page = i
		s.rebuild()
	})
	list.Selected = s.page
	list.RowHeight = 32
	s.navList = list

	brand := uitoolkit.NewColumn(
		uitoolkit.NewTitle("Settings"),
		uitoolkit.NewLabel("system-config"),
	).WithGap(2)
	side := uitoolkit.NewColumn(brand, uitoolkit.NewSeparator(), list).WithGap(8).WithPad(10)
	side.AddFlex(list, 1)
	return side
}

func (s *session) shellStatus() string {
	if s.cli == nil {
		return "system-configd not running — CLI/GUI still paint; Apply is disabled"
	}
	return "system-configd connected"
}

func (s *session) build() uitoolkit.Component {
	side := s.navRail()
	page := s.pageFor(s.path())
	split := uitoolkit.NewSplitter(true, side, uitoolkit.NewPad(16, page))
	split.Ratio = 0.22

	s.status = uitoolkit.NewStatusBar(s.shellStatus(), sockpath.Daemon(), "v1")
	chrome := uitoolkit.NewTitleBar("Coda Settings", "Apply is per section — staged edits stay until that section’s Apply or Refresh")
	root := uitoolkit.NewColumn(chrome, split, s.status)
	root.AddFlex(split, 1)
	return root
}

func (s *session) syncApply() {
	if s.applyBtn != nil {
		s.applyBtn.SetEnabled(s.dirtyPath(s.path()) && s.cli != nil)
	}
	if s.navList != nil {
		s.navList.Invalidate()
	}
}
