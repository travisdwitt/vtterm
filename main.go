package main

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/traviswitt/vtterm/internal/app"
)

func main() {
	var last time.Time
	drop := func(_ tea.Model, m tea.Msg) tea.Msg {
		if _, ok := m.(tea.MouseMotionMsg); ok {
			if now := time.Now(); now.Sub(last) < 16*time.Millisecond {
				return nil
			} else {
				last = now
			}
		}
		return m
	}
	p := tea.NewProgram(app.New(), tea.WithFilter(drop))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
