package tui

import (
	"fmt"
	"strings"
)

type SendModal struct {
	visible  bool
	target   string
	message  string
	focus    int // 0 = target, 1 = message
	feedback string
	err      string
}

func NewSendModal() SendModal {
	return SendModal{}
}

func (m *SendModal) Show() {
	m.visible = true
	m.target = ""
	m.message = ""
	m.focus = 0
	m.feedback = ""
	m.err = ""
}

func (m *SendModal) Hide() {
	m.visible = false
}

func (m *SendModal) IsVisible() bool {
	return m.visible
}

func (m *SendModal) Target() string {
	return m.target
}

func (m *SendModal) Message() string {
	return m.message
}

func (m *SendModal) SetFeedback(msg string) {
	m.feedback = msg
	m.err = ""
}

func (m *SendModal) SetError(msg string) {
	m.err = msg
	m.feedback = ""
}

func (m *SendModal) SwitchFocus() {
	m.focus = (m.focus + 1) % 2
}

func (m *SendModal) AddChar(ch rune) {
	if m.focus == 0 {
		m.target += string(ch)
	} else {
		m.message += string(ch)
	}
}

func (m *SendModal) DeleteChar() {
	if m.focus == 0 && len(m.target) > 0 {
		m.target = m.target[:len(m.target)-1]
	} else if m.focus == 1 && len(m.message) > 0 {
		m.message = m.message[:len(m.message)-1]
	}
}

func (m SendModal) View(width, height int) string {
	if !m.visible {
		return ""
	}

	var b strings.Builder
	b.WriteString(StyleModalTitle.Render("Send Message"))
	b.WriteString("\n\n")

	targetCursor := " "
	msgCursor := " "
	if m.focus == 0 {
		targetCursor = "▶"
	} else {
		msgCursor = "▶"
	}

	b.WriteString(fmt.Sprintf("%s To:      %s█\n", targetCursor, m.target))
	b.WriteString(fmt.Sprintf("%s Message: %s█\n", msgCursor, m.message))

	if m.feedback != "" {
		b.WriteString("\n")
		b.WriteString(StyleSuccess.Render("  " + m.feedback))
	}
	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(StyleError.Render("  " + m.err))
	}

	b.WriteString("\n\n")
	b.WriteString(StyleHelp.Render("  [tab] switch field  [enter] send  [esc] cancel"))

	content := b.String()
	modal := StyleModal.Render(content)

	padTop := max(0, (height-strings.Count(modal, "\n"))/2)
	padLeft := max(0, (width-60)/2)

	var out strings.Builder
	for i := 0; i < padTop; i++ {
		out.WriteString("\n")
	}
	for _, line := range strings.Split(modal, "\n") {
		out.WriteString(strings.Repeat(" ", padLeft))
		out.WriteString(line)
		out.WriteString("\n")
	}

	return out.String()
}
