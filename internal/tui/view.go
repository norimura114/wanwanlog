package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"wanwanlog/internal/logrecord"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	if m.mode == HelpMode {
		return m.renderHelp()
	}

	header := m.renderHeader()
	filter := m.renderFilter()

	bodyHeight := m.height - 6
	if bodyHeight < 5 {
		bodyHeight = 5
	}

	body := m.renderBody(bodyHeight)
	footer := m.renderFooter()

	return strings.Join([]string{
		header,
		filter,
		body,
		footer,
	}, "\n")
}

func (m Model) renderHeader() string {
	mode := "BROWSE"

	switch m.mode {
	case BrowseMode:
		mode = "BROWSE"
	case FilterMode:
		mode = "FILTER"
	case DetailMode:
		mode = "DETAIL"
	case HelpMode:
		mode = "HELP"
	}

	text := fmt.Sprintf(" wwlog | mode=%s | rows=%d | selected=%d ",
		mode,
		len(m.records),
		m.selectedIndex+1,
	)

	return headerStyle.Width(m.width).Render(text)
}

func (m Model) renderFilter() string {
	if m.mode == FilterMode {
		return filterStyle.Width(m.width).Render(m.filterInput.View())
	}

	query := m.filterText
	if query == "" {
		query = "(no filter)"
	}

	return filterStyle.Width(m.width).Render("> " + query)
}

func (m Model) renderBody(height int) string {
	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth
	listStyle, detailStyle := paneStyles(m.mode)
	leftContentWidth := paneContentWidth(listStyle, leftWidth)
	rightContentWidth := paneContentWidth(detailStyle, rightWidth)
	contentHeight := paneContentHeight(listStyle, height)

	list := m.renderList(leftContentWidth, contentHeight)
	detail := m.renderDetail(rightContentWidth, contentHeight)

	leftPane := listStyle.
		Width(paneRenderWidth(listStyle, leftWidth)).
		Height(paneRenderHeight(listStyle, height)).
		Render(list)
	rightPane := detailStyle.
		Width(paneRenderWidth(detailStyle, rightWidth)).
		Height(paneRenderHeight(detailStyle, height)).
		Render(detail)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}

func paneStyles(mode Mode) (lipgloss.Style, lipgloss.Style) {
	listStyle := listPaneStyle
	detailStyle := detailPaneStyle

	switch mode {
	case BrowseMode:
		listStyle = listStyle.BorderForeground(activePaneBorderColor)
	case DetailMode:
		detailStyle = detailStyle.BorderForeground(activePaneBorderColor)
	}

	return listStyle, detailStyle
}

func (m Model) renderList(width int, height int) string {
	if len(m.records) == 0 {
		return "No logs"
	}

	start := 0
	if m.selectedIndex >= height {
		start = m.selectedIndex - height + 1
	}

	end := start + height
	if end > len(m.records) {
		end = len(m.records)
	}

	lines := make([]string, 0, height)

	for i := start; i < end; i++ {
		record := m.records[i]

		line := formatLogLine(record, width-2)
		if i == m.selectedIndex {
			line = selectedLineStyle.Render("> " + line)
		} else {
			line = "  " + line
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderDetail(width int, height int) string {
	record := m.selectedRecord()
	if record == nil {
		return "No selected log"
	}

	lines := formatDetail(*record, width)
	offset := min(m.detailOffset, max(0, len(lines)-height))

	end := offset + height
	if end > len(lines) {
		end = len(lines)
	}

	return strings.Join(lines[offset:end], "\n")
}

func (m Model) renderFooter() string {
	if m.err != nil {
		return footerErrorStyle.Width(m.width).Render(" error: " + m.err.Error())
	}

	if m.loading {
		return footerStyle.Width(m.width).Render(" loading...")
	}

	switch m.mode {
	case BrowseMode:
		return footerStyle.Width(m.width).Render(" / filter | j/k move | d detail | ? help | q quit ")
	case FilterMode:
		return footerStyle.Width(m.width).Render(" enter search | esc cancel ")
	case DetailMode:
		return footerStyle.Width(m.width).Render(" j/k scroll | esc back ")
	default:
		return footerStyle.Width(m.width).Render("")
	}
}

func (m Model) renderHelp() string {
	content := strings.Join([]string{
		"wwlog help",
		"",
		"q / ctrl+c   quit",
		"/            filter mode",
		"enter        execute filter or open detail",
		"esc          back to browse mode",
		"j / down     next log",
		"k / up       previous log",
		"g            first log",
		"G            last log",
		"d            detail mode",
		"?            toggle help",
	}, "\n")

	return helpStyle.Width(m.width).Height(m.height).Render(content)
}

func formatLogLine(record logrecord.LogRecord, width int) string {
	timestamp := record.Timestamp.Format("15:04:05")
	status := record.Status
	service := record.Service
	message := singleLine(record.Message)

	line := fmt.Sprintf("%s %-5s %-12s %s", timestamp, status, service, message)
	return truncate(line, width)
}

func formatDetail(record logrecord.LogRecord, width int) []string {
	lines := []string{
		"timestamp: " + record.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		"service:   " + record.Service,
		"status:    " + record.Status,
		"message:   " + record.Message,
		"",
		"attributes:",
	}

	lines = append(lines, indentLines(formatJSON(record.Attributes), "  ")...)

	lines = append(lines, "", "raw:")
	lines = append(lines, indentLines(formatRaw(record.Raw), "  ")...)

	return wrapLines(lines, width)
}

func singleLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}

	if lipgloss.Width(s) <= width {
		return s
	}

	if width <= 1 {
		return ansi.Truncate(s, width, "")
	}

	return ansi.Truncate(s, width, "…")
}

func padRight(s string, width int) string {
	cellWidth := lipgloss.Width(s)
	if cellWidth >= width {
		return s
	}

	return s + strings.Repeat(" ", width-cellWidth)
}

func fitLine(s string, width int) string {
	return padRight(truncate(s, width), width)
}

func paneContentWidth(style lipgloss.Style, width int) int {
	return max(0, width-style.GetHorizontalFrameSize())
}

func paneRenderWidth(style lipgloss.Style, width int) int {
	return max(0, width-style.GetHorizontalMargins()-style.GetHorizontalBorderSize())
}

func paneContentHeight(style lipgloss.Style, height int) int {
	return max(0, height-style.GetVerticalFrameSize())
}

func paneRenderHeight(style lipgloss.Style, height int) int {
	return max(0, height-style.GetVerticalMargins()-style.GetVerticalBorderSize())
}

func formatJSON(value any) []string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return []string{fmt.Sprintf("%v", value)}
	}

	return strings.Split(string(data), "\n")
}

func formatRaw(raw string) []string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(raw), "", "  "); err == nil {
		return strings.Split(buf.String(), "\n")
	}

	return []string{singleLine(raw)}
}

func indentLines(lines []string, prefix string) []string {
	indented := make([]string, 0, len(lines))
	for _, line := range lines {
		indented = append(indented, prefix+line)
	}
	return indented
}

func wrapLines(lines []string, width int) []string {
	if width <= 0 {
		return []string{""}
	}

	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			wrapped = append(wrapped, "")
			continue
		}

		for lipgloss.Width(line) > width {
			wrapped = append(wrapped, ansi.Truncate(line, width, ""))
			line = strings.TrimLeft(ansi.TruncateLeft(line, width, ""), " ")
			if line == "" {
				break
			}
		}
		if line != "" {
			wrapped = append(wrapped, line)
		}
	}

	return wrapped
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
