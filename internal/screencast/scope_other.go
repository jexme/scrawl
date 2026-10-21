//go:build !linux

package screencast

// EnterAppScope is needed only on Linux with a portal older than 1.21.
func (p *Portal) EnterAppScope() error { return nil }
