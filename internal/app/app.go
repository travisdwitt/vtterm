package app

import (
	tea "charm.land/bubbletea/v2"
	"github.com/traviswitt/vtterm/internal/msg"
	"github.com/traviswitt/vtterm/internal/table"
	"github.com/traviswitt/vtterm/internal/views"
)

type screen int

const (
	screenMainMenu screen = iota
	screenWizard
	screenTableView
	screenLoad
	screenTokens
)

type Model struct {
	active      screen
	mainMenu    tea.Model
	wizard      tea.Model
	tableView   tea.Model
	loadList    tea.Model
	tokenScreen tea.Model
	tokenLib    *table.TokenLibrary
	width       int
	height      int
}

func New() Model {
	lib, err := table.LoadTokenLibrary()
	if err != nil {
		lib = &table.TokenLibrary{}
	}
	return Model{
		active:   screenMainMenu,
		mainMenu: views.NewMainMenu(),
		tokenLib: lib,
	}
}

func (m Model) Init() tea.Cmd {
	return m.mainMenu.Init()
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
	case msg.GoToWizard:
		m.wizard = views.NewWizard()
		m.active = screenWizard
		return m, m.wizard.Init()
	case msg.GoToTableView:
		m.tableView = views.NewTableView(message.Table, m.tokenLib, m.width, m.height)
		m.active = screenTableView
		return m, m.tableView.Init()
	case msg.GoToMainMenu:
		m.mainMenu = views.NewMainMenu()
		m.active = screenMainMenu
		return m, nil
	case msg.GoToTokens:
		m.tokenScreen = views.NewTokenScreen(m.tokenLib, m.width)
		m.active = screenTokens
		return m, m.tokenScreen.Init()
	case msg.GoToLoad:
		m.loadList = newLoadScreen()
		m.active = screenLoad
		return m, m.loadList.Init()
	}

	var cmd tea.Cmd
	switch m.active {
	case screenMainMenu:
		m.mainMenu, cmd = m.mainMenu.Update(message)
	case screenWizard:
		m.wizard, cmd = m.wizard.Update(message)
	case screenTableView:
		m.tableView, cmd = m.tableView.Update(message)
	case screenLoad:
		m.loadList, cmd = m.loadList.Update(message)
	case screenTokens:
		m.tokenScreen, cmd = m.tokenScreen.Update(message)
	}
	return m, cmd
}

func (m Model) View() tea.View {
	var v tea.View
	switch m.active {
	case screenMainMenu:
		v = m.mainMenu.View()
	case screenWizard:
		v = m.wizard.View()
	case screenTableView:
		v = m.tableView.View()
	case screenLoad:
		v = m.loadList.View()
	case screenTokens:
		v = m.tokenScreen.View()
	default:
		v = tea.NewView("")
	}
	v.AltScreen = true
	return v
}
