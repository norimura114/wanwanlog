package tui

import (
	"fmt"
	"strings"
	"wanwanlog/internal/logrecord"
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
	rightWidth := m.width - leftWidth - 1

	list := m.renderList(leftWidth, height)
	detail := m.renderDetail(rightWidth, height)

	lines := make([]string, 0, height)

	listLines := strings.Split(list, "\n")
	detailLines := strings.Split(detail, "\n")

	for i := 0; i < height; i++ {
		left := ""
		right := ""

		if i < len(listLines) {
			left = listLines[i]
		}
		if i < len(detailLines) {
			right = detailLines[i]
		}

		lines = append(lines,
			listPaneStyle.Width(leftWidth).Render(padRight(left, leftWidth))+
				detailPaneStyle.Width(rightWidth).Render(padRight(right, rightWidth)),
		)
	}

	return strings.Join(lines, "\n")
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

	lines := formatDetail(*record)

	if m.detailOffset > len(lines)-1 {
		m.detailOffset = max(0, len(lines)-1)
	}

	end := m.detailOffset + height
	if end > len(lines) {
		end = len(lines)
	}

	return strings.Join(lines[m.detailOffset:end], "\n")
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

func formatDetail(record logrecord.LogRecord) []string {
	lines := []string{
		"timestamp: " + record.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		"service:   " + record.Service,
		"status:    " + record.Status,
		"message:   " + record.Message,
		"",
		"attributes:",
	}

	for key, value := range record.Attributes {
		lines = append(lines, fmt.Sprintf("  %s: %v", key, value))
	}

	lines = append(lines, "", "raw:", record.Raw)

	return lines
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

	runes := []rune(s)
	if len(runes) <= width {
		return s
	}

	if width <= 1 {
		return string(runes[:width])
	}

	return string(runes[:width-1]) + "…"
}

func padRight(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return s
	}

	return s + strings.Repeat(" ", width-len(runes))
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
