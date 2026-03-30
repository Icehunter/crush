package chat

import (
	"strings"

	"github.com/charmbracelet/crush/internal/ui/styles"
)

// SystemNoticeItem renders a system notice (like help text or status output)
// persistently in the chat view. Unlike InfoMsg which goes to the status bar
// with a TTL, this stays in the chat history.
type SystemNoticeItem struct {
	*cachedMessageItem

	id   string
	text string
	sty  *styles.Styles
}

// NewSystemNoticeItem creates a new SystemNoticeItem.
func NewSystemNoticeItem(sty *styles.Styles, id, text string) MessageItem {
	return &SystemNoticeItem{
		cachedMessageItem: &cachedMessageItem{},
		id:                id,
		text:              text,
		sty:               sty,
	}
}

// ID implements MessageItem.
func (s *SystemNoticeItem) ID() string {
	return s.id
}

// RawRender implements MessageItem.
func (s *SystemNoticeItem) RawRender(width int) string {
	return s.text
}

// Render implements MessageItem.
func (s *SystemNoticeItem) Render(width int) string {
	prefix := s.sty.Chat.Message.SectionHeader.Render()
	lines := strings.Split(s.RawRender(width), "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
