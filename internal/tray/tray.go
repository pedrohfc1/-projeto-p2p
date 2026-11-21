package tray

// Tray actions never open a microphone; they operate only on an already joined call.
type Callbacks struct{ Open, Mute, Leave, Quit, StopScreen func() }
type State struct{ Joined, Muted, Sharing bool }

func (s State) Tooltip() string {
	if s.Sharing {
		return "PeeTwoPee — AO VIVO · tela compartilhada; botão direito para parar"
	}
	if !s.Joined {
		return "PeeTwoPee — sem chamada"
	}
	if s.Muted {
		return "PeeTwoPee — microfone silenciado"
	}
	return "PeeTwoPee — em chamada, microfone ativo"
}
