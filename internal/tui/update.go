package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Init() tea.Cmd {
	return waitFollowEvent(m.followEvents)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampDetailOffset()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case searchDoneMsg:
		if msg.generation != m.generation {
			return m, nil
		}
		m.loading = false
		m.searching = false
		m.err = nil
		m.applySearchResult(msg.result)
		if m.dirty {
			m.dirty = false
			return m.startSearch()
		}
		return m, nil
	case searchFailedMsg:
		if msg.generation != m.generation {
			return m, nil
		}
		m.loading = false
		m.searching = false
		m.err = msg.err
		return m, nil
	case FollowEvent:
		m.state = msg.State
		if msg.ParseErrorCount > 0 {
			m.parseErrors += msg.ParseErrorCount
		}
		m.addAttributeNames(msg.AttributeNames)
		if msg.Err != nil || msg.State == "live" {
			m.err = msg.Err
		}
		cmds := []tea.Cmd{waitFollowEvent(m.followEvents)}
		if msg.AddedCount > 0 || msg.Reset {
			if m.searching {
				m.dirty = true
			} else {
				var cmd tea.Cmd
				m, cmd = m.startSearch()
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}

	if m.mode == FilterMode {
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case BrowseMode:
		return m.handleBrowseKey(msg)
	case FilterMode:
		return m.handleFilterKey(msg)
	case DetailMode:
		return m.handleDetailKey(msg)
	case HelpMode:
		return m.handleHelpKey(msg)
	case ColumnsMode:
		return m.handleColumnsKey(msg)
	default:
		return m, nil
	}
}

func (m Model) handleBrowseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "/":
		m.mode = FilterMode
		m.filterInput.Focus()
		m.filterInput.SetValue(m.filterText)
		return m, nil

	case "j", "down":
		if m.selectedIndex < len(m.records)-1 {
			m.selectedIndex++
		}
		return m, nil

	case "k", "up":
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}
		return m, nil

	case "g":
		m.selectedIndex = 0
		return m, nil

	case "G":
		if len(m.records) > 0 {
			m.selectedIndex = len(m.records) - 1
		}
		return m, nil

	case "d", "enter", "l":
		m.mode = DetailMode
		m.clampDetailOffset()
		return m, nil

	case "?":
		m.mode = HelpMode
		return m, nil

	case "c":
		m.mode = ColumnsMode
		m.columnDraft = cloneColumns(m.columns)
		m.columnCursor = 0
		m.columnOffset = 0
		return m, nil
	}

	return m, nil
}

func (m Model) handleColumnsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = BrowseMode
		m.columnDraft = nil
		return m, nil

	case "enter":
		m.columns = cloneColumns(m.columnDraft)
		m.columnDraft = nil
		m.mode = BrowseMode
		return m, nil

	case "j", "down":
		if m.columnCursor < len(m.columnDraft)-1 {
			m.columnCursor++
		}
		m.clampColumnOffset()
		return m, nil

	case "k", "up":
		if m.columnCursor > 0 {
			m.columnCursor--
		}
		m.clampColumnOffset()
		return m, nil

	case " ":
		if len(m.columnDraft) == 0 {
			return m, nil
		}
		if m.columnDraft[m.columnCursor].Visible && len(visibleColumns(m.columnDraft)) == 1 {
			return m, nil
		}
		m.columnDraft[m.columnCursor].Visible = !m.columnDraft[m.columnCursor].Visible
		return m, nil

	case "J":
		if m.columnCursor < len(m.columnDraft)-1 {
			m.columnDraft[m.columnCursor], m.columnDraft[m.columnCursor+1] =
				m.columnDraft[m.columnCursor+1], m.columnDraft[m.columnCursor]
			m.columnCursor++
		}
		m.clampColumnOffset()
		return m, nil

	case "K":
		if m.columnCursor > 0 {
			m.columnDraft[m.columnCursor], m.columnDraft[m.columnCursor-1] =
				m.columnDraft[m.columnCursor-1], m.columnDraft[m.columnCursor]
			m.columnCursor--
		}
		m.clampColumnOffset()
		return m, nil
	}

	return m, nil
}

func (m *Model) clampColumnOffset() {
	height := max(1, m.height-5)
	if m.columnCursor < m.columnOffset {
		m.columnOffset = m.columnCursor
	}
	if m.columnCursor >= m.columnOffset+height {
		m.columnOffset = m.columnCursor - height + 1
	}
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = BrowseMode
		m.filterInput.Blur()
		return m, nil

	case "enter":
		query := m.filterInput.Value()
		m.filterText = query
		m.mode = BrowseMode
		m.filterInput.Blur()
		m.err = nil
		return m.startSearch()
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	return m, cmd
}

func (m Model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "h":
		m.mode = BrowseMode
		return m, nil

	case "j", "down":
		if m.detailOffset < m.maxDetailOffset() {
			m.detailOffset++
		}
		return m, nil

	case "k", "up":
		if m.detailOffset > 0 {
			m.detailOffset--
		}
		return m, nil

	case "r":
		m.detailRaw = !m.detailRaw
		m.clampDetailOffset()
		return m, nil
	}

	return m, nil
}

func (m *Model) clampDetailOffset() {
	m.detailOffset = min(m.detailOffset, m.maxDetailOffset())
}

func (m Model) maxDetailOffset() int {
	record := m.selectedRecord()
	if record == nil {
		return 0
	}

	rightWidth := m.width - m.width/2
	contentWidth := paneContentWidth(detailPaneStyle, rightWidth)
	bodyHeight := m.height - 6
	if bodyHeight < 5 {
		bodyHeight = 5
	}
	contentHeight := paneContentHeight(detailPaneStyle, bodyHeight)

	return max(0, len(formatDetail(*record, contentWidth, m.detailRaw))-contentHeight)
}

func (m Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "?":
		m.mode = BrowseMode
		return m, nil
	}

	return m, nil
}

type searchDoneMsg struct {
	result     SearchResult
	generation uint64
}

type searchFailedMsg struct {
	err        error
	generation uint64
}

func waitFollowEvent(events <-chan FollowEvent) tea.Cmd {
	if events == nil {
		return nil
	}
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			return nil
		}
		return event
	}
}

func (m Model) startSearch() (Model, tea.Cmd) {
	m.loading = true
	m.searching = true
	m.generation++
	generation := m.generation
	query := m.filterText
	return m, func() tea.Msg {
		result, err := m.search(m.ctx, query)
		if err != nil {
			return searchFailedMsg{err: err, generation: generation}
		}
		return searchDoneMsg{result: result, generation: generation}
	}
}

func (m *Model) applySearchResult(result SearchResult) {
	oldIndex := m.selectedIndex
	oldID := ""
	atEnd := len(m.records) > 0 && oldIndex == len(m.records)-1
	if record := m.selectedRecord(); record != nil {
		oldID = record.ID
	}
	m.records, m.total = result.Records, result.Total
	if len(m.records) == 0 {
		m.selectedIndex = 0
		return
	}
	if atEnd {
		m.selectedIndex = len(m.records) - 1
		return
	}
	for i := range m.records {
		if m.records[i].ID == oldID {
			m.selectedIndex = i
			return
		}
	}
	m.selectedIndex = min(oldIndex, len(m.records)-1)
	m.clampDetailOffset()
}

func (m *Model) addAttributeNames(names []string) {
	existing := make(map[string]bool, len(m.columns))
	for _, column := range m.columns {
		existing[column.Name] = true
	}
	for _, name := range names {
		if !existing[name] {
			m.columns = append(m.columns, Column{Name: name})
			existing[name] = true
		}
	}
}

func (m Model) validateSearchFunc() error {
	if m.search == nil {
		return fmt.Errorf("search function is nil")
	}
	return nil
}
