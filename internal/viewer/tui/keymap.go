package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Quit        key.Binding
	Tab         key.Binding
	NextComment key.Binding
	PrevComment key.Binding
	Up          key.Binding
	Down        key.Binding
	PageUp      key.Binding
	PageDown    key.Binding
	HalfPageUp  key.Binding
	HalfPageDown key.Binding
	Home        key.Binding
	End         key.Binding
	ToggleLayout key.Binding
	Help        key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch pane"),
		),
		NextComment: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next comment"),
		),
		PrevComment: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "prev comment"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "scroll up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "scroll down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "page down"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("C-u", "half page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("C-d", "half page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "bottom"),
		),
		ToggleLayout: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "toggle layout"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

func (k keyMap) helpKeys() []key.Binding {
	return []key.Binding{
		k.Quit, k.Tab, k.NextComment, k.PrevComment,
		k.Up, k.Down, k.ToggleLayout, k.Help,
	}
}
