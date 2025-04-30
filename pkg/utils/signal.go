package utils

import (
	"os"
	"os/signal"
	"syscall"
)

var (
	terminalCh = make(chan os.Signal, 1)
)

func init() {
	signal.Notify(terminalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
}

func HandleTerminalSignal() chan struct{} {
	ch := make(chan struct{})

	go func() {
		<-terminalCh
		close(ch)
		<-terminalCh
		os.Exit(2)
	}()

	return ch
}

func Shutdown() {
	terminalCh <- syscall.SIGQUIT
}
