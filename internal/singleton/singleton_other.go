//go:build !linux

package singleton

// Acquire пока ничего не блокирует на этой платформе (для Windows нужен именованный мьютекс).
func Acquire(string) (bool, error) { return true, nil }
