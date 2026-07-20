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
	if m.mode == ColumnsMode {
		return strings.Join([]string{
			header,
			m.renderColumnSettings(max(1, m.height-2)),
			m.renderFooter(),
		}, "\n")
	}

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
	case ColumnsMode:
		mode = "COLUMNS"
	}

	text := fmt.Sprintf(" wwlog | mode=%s | rows=%d/%d | selected=%d ",
		mode,
		len(m.records),
		m.total,
		m.selectedIndex+1,
	)
	state := m.state
	if m.loading {
		state = "loading"
	}
	if state != "" {
		text += fmt.Sprintf("| %s", state)
	}
	if m.parseErrors > 0 {
		text += fmt.Sprintf(" | parse_errors=%d", m.parseErrors)
	}

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
	columns := visibleColumns(m.columns)
	if len(columns) == 0 {
		return "No columns"
	}

	dataHeight := max(0, height-1)
	widths := browseColumnWidths(columns, m.records, max(0, width-2))
	lines := make([]string, 0, height)
	lines = append(lines, "  "+truncate(formatBrowseRow(columns, widths, nil), max(0, width-2)))

	if len(m.records) == 0 {
		lines = append(lines, "  No logs")
		return strings.Join(lines, "\n")
	}

	start := 0
	if m.selectedIndex >= dataHeight {
		start = m.selectedIndex - dataHeight + 1
	}

	end := start + dataHeight
	if end > len(m.records) {
		end = len(m.records)
	}

	for i := start; i < end; i++ {
		record := m.records[i]

		line := truncate(formatBrowseRow(columns, widths, &record), max(0, width-2))
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

	lines := formatDetail(*record, width, m.detailRaw)
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
		return footerStyle.Width(m.width).Render(" / filter | j/k move | l detail | c columns | ? help | q quit ")
	case FilterMode:
		return footerStyle.Width(m.width).Render(" enter search | esc cancel ")
	case DetailMode:
		return footerStyle.Width(m.width).Render(" j/k scroll | r attributes/raw | h browse | esc back ")
	case ColumnsMode:
		return footerStyle.Width(m.width).Render(" j/k select | space toggle | J/K reorder | enter apply | esc cancel ")
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
		"h / l        switch pane",
		"d            detail pane",
		"r            toggle attributes/raw in detail",
		"c            column settings",
		"?            toggle help",
	}, "\n")

	return helpStyle.Width(m.width).Height(m.height).Render(content)
}

func (m Model) renderColumnSettings(height int) string {
	title := "Browse columns"
	lines := []string{title, ""}
	listHeight := max(1, height-len(lines))
	start := min(m.columnOffset, max(0, len(m.columnDraft)-listHeight))
	end := min(len(m.columnDraft), start+listHeight)

	for i := start; i < end; i++ {
		column := m.columnDraft[i]
		check := " "
		if column.Visible {
			check = "x"
		}
		cursor := "  "
		if i == m.columnCursor {
			cursor = "> "
		}
		line := fmt.Sprintf("%s[%s] %s", cursor, check, column.Name)
		lines = append(lines, truncate(line, max(0, m.width-helpStyle.GetHorizontalFrameSize())))
	}

	return helpStyle.
		Width(max(0, m.width-helpStyle.GetHorizontalFrameSize())).
		Height(max(0, height-helpStyle.GetVerticalFrameSize())).
		Render(strings.Join(lines, "\n"))
}

func browseColumnWidths(columns []Column, records []logrecord.LogRecord, available int) []int {
	widths := make([]int, len(columns))
	for i, column := range columns {
		widths[i] = lipgloss.Width(column.Name)
		for _, record := range records {
			value := formatColumnValue(
				column.Name,
				record.Attributes,
				record.Timestamp,
				record.Status,
				record.Service,
				record.Message,
			)
			widths[i] = max(widths[i], lipgloss.Width(value))
		}
	}

	contentWidth := max(0, available-max(0, len(columns)-1))
	overflow := sum(widths) - contentWidth
	if overflow <= 0 {
		return widths
	}

	for i, column := range columns {
		if column.Name != MessageColumn {
			continue
		}
		reduction := min(overflow, max(0, widths[i]-1))
		widths[i] -= reduction
		overflow -= reduction
		break
	}

	for overflow > 0 {
		widest := -1
		for i, width := range widths {
			if width > 1 && (widest == -1 || width > widths[widest]) {
				widest = i
			}
		}
		if widest == -1 {
			break
		}
		widths[widest]--
		overflow--
	}

	return widths
}

func formatBrowseRow(columns []Column, widths []int, record *logrecord.LogRecord) string {
	cells := make([]string, len(columns))
	for i, column := range columns {
		value := column.Name
		if record != nil {
			value = formatColumnValue(
				column.Name,
				record.Attributes,
				record.Timestamp,
				record.Status,
				record.Service,
				record.Message,
			)
		}
		cells[i] = fitLine(value, widths[i])
	}
	return strings.Join(cells, " ")
}

func sum(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

func formatDetail(record logrecord.LogRecord, width int, raw bool) []string {
	lines := []string{
		"timestamp: " + record.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		"service:   " + record.Service,
		"status:    " + record.Status,
		"message:   " + record.Message,
		"",
	}

	if raw {
		lines = append(lines, "raw:")
		lines = append(lines, indentLines(formatRaw(record.Raw), "  ")...)
	} else {
		lines = append(lines, "attributes:")
		lines = append(lines, indentLines(formatJSON(record.Attributes), "  ")...)
	}

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
