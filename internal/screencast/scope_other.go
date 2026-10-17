//go:build !linux

package screencast

// EnterAppScope нужна только Linux с portal старше 1.21.
func (p *Portal) EnterAppScope() error { return nil }
