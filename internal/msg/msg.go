package msg

import "github.com/traviswitt/vtterm/internal/table"

type GoToWizard struct{}

type GoToTableView struct {
	Table table.Table
}

type GoToMainMenu struct{}

type GoToLoad struct{}

type GoToTokens struct{}
