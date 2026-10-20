//go:build !windows

package main

import "os/exec"

func notify(title, msg string) { exec.Command("notify-send", "-a", "Scrawl", title, msg).Run() }
