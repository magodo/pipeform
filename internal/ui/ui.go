package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/magodo/pipeform/internal/clipboard"
	"github.com/magodo/pipeform/internal/csv"
	"github.com/magodo/pipeform/internal/log"
	"github.com/magodo/pipeform/internal/state"
	"github.com/muesli/reflow/indent"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/magodo/pipeform/internal/reader"
	"github.com/magodo/pipeform/internal/terraform/views"
	"github.com/magodo/pipeform/internal/terraform/views/json"
)

const (
	padding     = 2
	indentLevel = 2
)

type UIModel struct {
	startTime time.Time
	logger    *log.Logger
	reader    reader.Reader

	// state is the actual state of the process
	state         ViewState
	visitedStates []ViewState
	// viewState is the state of the current view. It is nil until EOF received.
	// After which, users can select different view.
	viewState *ViewState

	lastLog           string
	userOperationInfo string

	isEOF bool

	diags Diags

	refreshInfos state.ResourceOperationInfos
	planInfos    state.PlanInfos
	applyInfos   state.ResourceOperationInfos
	outputInfos  state.OutputInfos

	versionMsg *string

	// These are read from the ChangeSummaryMsg
	operation json.Operation
	totalCnt  int

	doneCnt int

	keymap KeyMap

	help      help.Model
	spinner   spinner.Model
	table     table.Model
	progress  progress.Model
	paginator paginator.Model

	tableSize Size

	cp clipboard.Clipboard

	followed bool
}

func NewRuntimeModel(logger *log.Logger, reader reader.Reader, startTime time.Time, follow bool) UIModel {
	t := table.New(table.WithFocused(true))
	t.SetStyles(StyleTableFunc())

	cp := clipboard.NewClipboard()

	keymap := NewKeyMap(cp.Enabled())

	p := paginator.New()
	p.KeyMap = keymap.PaginatorMap
	p.Type = paginator.Dots
	p.ActiveDot = StyleActiveDot
	p.InactiveDot = StyleInactiveDot

	model := UIModel{
		startTime:     startTime,
		logger:        logger,
		reader:        reader,
		state:         ViewStateIdle,
		visitedStates: []ViewState{ViewStateIdle},
		keymap:        keymap,
		help:          help.New(),
		spinner:       spinner.New(),
		table:         t,
		progress:      progress.New(),
		paginator:     p,
		cp:            cp,
		followed:      follow,
	}

	return model
}

func (m UIModel) Diags() Diags {
	return m.diags
}

func (m UIModel) IsEOF() bool {
	return m.isEOF
}

func (m UIModel) nextMessage() tea.Msg {
	msg, err := m.reader.Next()
	if err != nil {
		if err == io.EOF {
			return receiverEOFMsg{}
		}
		return receiverErrorMsg{err: err}
	}
	return receiverMsg{msg: msg}
}

func (m UIModel) Init() tea.Cmd {
	return tea.Batch(m.nextMessage, m.spinner.Tick, tickCmd())
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.logger.Trace("Message received", "type", fmt.Sprintf("%T", msg))
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.userOperationInfo = ""
		switch {
		case key.Matches(msg, m.keymap.Quit):
			m.logger.Warn("Interrupt key received, quit the program")
			return m, tea.Quit
		case key.Matches(msg, m.keymap.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		case key.Matches(msg, m.keymap.Follow):
			m.followed = !m.followed
			return m, nil
		case key.Matches(msg, m.keymap.Copy):
			m.copyTableRow()
			return m, nil
		case key.Matches(msg, m.keymap.PaginatorMap.PrevPage):
			if m.viewState == nil {
				return m, nil
			}
			m.paginator.PrevPage()
			idx, _ := m.paginator.GetSliceBounds(len(m.visitedStates))
			m.viewState = &m.visitedStates[idx]
			m.resetTableNonEmpty()
			return m, nil
		case key.Matches(msg, m.keymap.PaginatorMap.NextPage):
			if m.viewState == nil {
				return m, nil
			}
			m.paginator.NextPage()
			idx, _ := m.paginator.GetSliceBounds(len(m.visitedStates))
			m.viewState = &m.visitedStates[idx]
			m.resetTableNonEmpty()
			return m, nil
		default:
			table, cmd := m.table.Update(msg)
			m.table = table
			return m, cmd
		}
	case tea.WindowSizeMsg:
		progressWidth := msg.Width - padding*2
		m.progress.Width = progressWidth

		m.tableSize = Size{
			Width:  msg.Width - padding*2 - 10,
			Height: msg.Height - padding*2 - 10,
		}
		m.setTableOutlook()
		m.setTableRows()

		return m, nil

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)

		return m, cmd

	case tickMsg:
		m.setTableRows()
		return m, tickCmd()

	// Log the receiver error message
	case receiverErrorMsg:
		m.logger.Error("Receiver error", "error", msg.Error())
		return m, m.nextMessage

	case receiverEOFMsg:
		m.logger.Info("Receiver reaches EOF")
		m.isEOF = true
		m.lastLog = fmt.Sprintf("Time spent: %s", time.Now().Sub(m.startTime).Truncate(time.Second))

		// Enable paginator
		m.paginator.SetTotalPages(len(m.visitedStates))
		for i := 0; i < len(m.visitedStates); i++ {
			m.paginator.NextPage()
		}
		m.viewState = &m.state
		m.keymap.EnablePaginator()

		return m, nil

	case receiverMsg:
		m.logger.Debug("Message receiverMsg received", "type", fmt.Sprintf("%T", msg.msg))

		cmds := []tea.Cmd{m.nextMessage}

		m.lastLog = msg.msg.BaseMessage().Message

		switch msg := msg.msg.(type) {
		case views.VersionMsg:
			m.versionMsg = &msg.BaseMsg.Message

		case views.LogMsg:
			// There's no much useful information for now.
		case views.DiagnosticsMsg:
			switch strings.ToLower(msg.Level) {
			case "warn", "error":
				m.diags = append(m.diags, *msg.Diagnostic)
			}

		case views.ResourceDriftMsg:
			// There's no much useful information for now.

		case views.PlannedChangeMsg:
			m.planInfos = append(m.planInfos, &state.PlanInfo{
				Resource:     msg.Change.Resource,
				Action:       msg.Change.Action,
				PrevResource: msg.Change.PreviousResource,
				Reason:       msg.Change.Reason,
			})

			// Normally, we don't need to handle the PlannedChangeMsg here, as the ChangeSummaryMsg has all these information.
			// The exception is that when apply with a plan file, there is no ChangeSummaryMsg sent from Terraform at this moment.
			// (see: https://github.com/magodo/pipeform/issues/1)
			// The counting here is a fallback logic to cover the case above. Otherwise, it will just be overwritten by ChangeSummaryMsg.
			//
			// TODO: Once https://github.com/hashicorp/terraform/pull/36245 merged, remove this part.
			//
			// Referencing the logic of terraform: internal/command/views/operation.go
			// But we also count the "import"
			switch msg.Change.Action {
			case json.ActionCreate:
				m.totalCnt++
			case json.ActionDelete:
				m.totalCnt++
			case json.ActionUpdate:
				m.totalCnt++
			case json.ActionReplace:
				m.totalCnt += 2
			case json.ActionImport:
				m.totalCnt++
			}

		case views.ChangeSummaryMsg:
			changes := msg.Changes
			m.logger.Debug("Change summary", "add", changes.Add, "change", changes.Change, "import", changes.Import, "remove", changes.Remove)
			m.totalCnt = changes.Add + changes.Change + changes.Import + changes.Remove
			m.operation = changes.Operation

			// Specifically, if the total count is 0, we update the progress bar directly as it is 100% anyway.
			if m.totalCnt == 0 {
				cmds = append(cmds, m.progress.SetPercent(1))
			}

		case views.OutputMsg:
			for name, o := range msg.Outputs {
				if o.Action != "" {
					continue
				}
				m.outputInfos = append(m.outputInfos, &state.OutputInfo{
					Name:      name,
					Sensitive: o.Sensitive,
					Type:      o.Type,
					ValueStr:  o.Value,
					Action:    o.Action,
				})
			}

		case views.HookMsg:
			m.logger.Debug("Hook message", "type", fmt.Sprintf("%T", msg.Hook))
			switch hook := msg.Hook.(type) {
			case json.RefreshStart:
				res := &state.ResourceOperationInfo{
					Idx:             len(m.refreshInfos) + 1,
					RawResourceAddr: hook.Resource,
					Loc: state.ResourceOperationInfoLocator{
						Module:       hook.Resource.Module,
						ResourceAddr: hook.Resource.Addr,
						Action:       "refresh",
					},
					Status:    state.ResourceOperationStatusStart,
					StartTime: msg.TimeStamp,
				}
				m.refreshInfos = append(m.refreshInfos, res)

			case json.RefreshComplete:
				loc := state.ResourceOperationInfoLocator{
					Module:       hook.Resource.Module,
					ResourceAddr: hook.Resource.Addr,
					Action:       "refresh",
				}
				status := state.ResourceOperationStatusComplete
				update := state.ResourceOperationInfoUpdate{
					Status:  &status,
					Endtime: &msg.TimeStamp,
				}
				if m.refreshInfos.Update(loc, update) == nil {
					m.logger.Error("RefreshComplete hook can't find the resource info", "module", hook.Resource.Module, "addr", hook.Resource.Addr, "action", "refresh")
					break
				}

			case json.OperationStart:
				res := &state.ResourceOperationInfo{
					Idx:             len(m.applyInfos) + 1,
					RawResourceAddr: hook.Resource,
					Loc: state.ResourceOperationInfoLocator{
						Module:       hook.Resource.Module,
						ResourceAddr: hook.Resource.Addr,
						Action:       string(hook.Action),
					},
					Status:    state.ResourceOperationStatusStart,
					StartTime: msg.TimeStamp,
				}
				m.applyInfos = append(m.applyInfos, res)

			case json.OperationProgress:
				// Ignore

			case json.OperationComplete:
				loc := state.ResourceOperationInfoLocator{
					Module:       hook.Resource.Module,
					ResourceAddr: hook.Resource.Addr,
					Action:       string(hook.Action),
				}
				status := state.ResourceOperationStatusComplete
				update := state.ResourceOperationInfoUpdate{
					Status:  &status,
					Endtime: &msg.TimeStamp,
				}
				if m.applyInfos.Update(loc, update) == nil {
					m.logger.Error("OperationComplete hook can't find the resource info", "module", hook.Resource.Module, "addr", hook.Resource.Addr, "action", hook.Action)
					break
				}

				m.doneCnt += 1
				percentage := float64(m.doneCnt) / float64(m.totalCnt)
				cmds = append(cmds, m.progress.SetPercent(percentage))

			case json.OperationErrored:
				loc := state.ResourceOperationInfoLocator{
					Module:       hook.Resource.Module,
					ResourceAddr: hook.Resource.Addr,
					Action:       string(hook.Action),
				}
				status := state.ResourceOperationStatusErrored
				update := state.ResourceOperationInfoUpdate{
					Status:  &status,
					Endtime: &msg.TimeStamp,
				}
				if m.applyInfos.Update(loc, update) == nil {
					m.logger.Error("OperationErrored hook can't find the resource info", "module", hook.Resource.Module, "addr", hook.Resource.Addr, "action", hook.Action)
					break
				}

				m.doneCnt += 1
				percentage := float64(m.doneCnt) / float64(m.totalCnt)
				cmds = append(cmds, m.progress.SetPercent(percentage))

			case json.ProvisionStart:
			case json.ProvisionProgress:
			case json.ProvisionComplete:
			case json.ProvisionErrored:
			default:
			}
		default:
			panic(fmt.Sprintf("unknown message type: %T", msg))
		}

		// Update viewState
		var change bool
		oldState := m.state
		m.state, change = m.state.NextState(msg.msg)
		if change {
			m.logger.Info("View State change", "old", oldState.String(), "new", m.state.String())
			m.visitedStates = append(m.visitedStates, m.state)
			m.resetTableEmpty()
		} else {
			m.setTableRows()
		}

		return m, tea.Batch(cmds...)

	default:
		return m, nil
	}
}

func (m *UIModel) resetTableEmpty() {
	// Clean up the rows before changing table columns, mainly to avoid
	// existing rows have more columns than the new columns, i.e. from
	// "apply" (6) to "summary" (5).
	m.table.SetRows(nil)
	m.table.SetCursor(0)
	m.setTableOutlook()
}

func (m *UIModel) resetTableNonEmpty() {
	m.resetTableEmpty()
	m.setTableRows()
}

func (m *UIModel) setTableOutlook() {
	m.table.SetWidth(m.tableSize.Width)
	m.table.SetHeight(m.tableSize.Height)

	switch m.getViewState() {
	case ViewStateRefresh:
		m.table.SetColumns(m.refreshInfos.ToColumns(m.tableSize.Width))
	case ViewStatePlan:
		m.table.SetColumns(m.planInfos.ToColumns(m.tableSize.Width))
	case ViewStateApply:
		m.table.SetColumns(m.applyInfos.ToColumns(m.tableSize.Width))
	case ViewStateSummary:
		m.table.SetColumns(m.outputInfos.ToColumns(m.tableSize.Width))
	}
}

// setTableRows on a one second pace.
func (m *UIModel) setTableRows() {
	switch m.getViewState() {
	case ViewStateRefresh:
		m.table.SetRows(m.refreshInfos.ToRows(0))
	case ViewStatePlan:
		m.table.SetRows(m.planInfos.ToRows())
	case ViewStateApply:
		m.table.SetRows(m.applyInfos.ToRows(m.totalCnt))
	case ViewStateSummary:
		m.table.SetRows(m.outputInfos.ToRows())
	}

	if m.followed {
		m.table.GotoBottom()
	}
}

func (m *UIModel) copyTableRow() {
	if !m.cp.Enabled() {
		return
	}

	switch m.getViewState() {
	case ViewStateRefresh:
		if row := m.table.SelectedRow(); len(row) > 4 {
			m.cp.Write([]byte(row[4]))
		}
	case ViewStateApply:
		if row := m.table.SelectedRow(); len(row) > 4 {
			m.cp.Write([]byte(row[4]))
		}
	case ViewStateSummary:
		if row := m.table.SelectedRow(); len(row) > 4 {
			m.cp.Write([]byte(row[4]))
		}
	}

	m.userOperationInfo = "Copied!"
}

func (m UIModel) ToCsv() []byte {
	return csv.ToCsv(csv.Input{
		RefreshInfos: m.refreshInfos,
		ApplyInfos:   m.applyInfos,
	})
}

func (m *UIModel) getViewState() ViewState {
	if m.viewState != nil {
		return *m.viewState
	}
	return m.state
}

func (m UIModel) logoView() string {
	msg := "pipeform"
	if m.versionMsg != nil {
		msg += fmt.Sprintf(" (%s)", *m.versionMsg)
	}
	return StyleTitle.Render(" " + msg + " ")
}

func (m UIModel) stateView() string {
	prefix := m.spinner.View()
	if m.isEOF {
		if m.diags.HasError() {
			prefix = "❌"
		} else {
			prefix = "✅"
		}
	}

	s := prefix + " " + StyleSubtitle.Render(m.getViewState().String())

	if m.followed {
		s += " [following]"
	}

	if m.lastLog != "" {
		s += "  " + StyleComment.Render(m.lastLog)
	}

	return s
}

func (m UIModel) View() string {
	s := "\n" + m.logoView()

	s += "\n\n" + m.stateView()

	if m.getViewState() != ViewStateIdle {
		s += "\n\n" + StyleTableBase.Render(m.table.View())
	}

	var progressBar string
	if m.getViewState() == ViewStateApply {
		progressBar = m.progress.View()
	}
	s += "\n\n" + progressBar

	if m.viewState != nil {
		s += "\n" + m.paginator.View()
	}

	var bottomLine string
	if m.userOperationInfo != "" {
		bottomLine = StyleComment.Render(m.userOperationInfo)
	} else {
		bottomLine = m.help.View(m.keymap)
	}
	s += "\n\n" + bottomLine

	return indent.String(s, indentLevel)
}
