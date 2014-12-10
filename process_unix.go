// +build linux

package main

import (
	"os"
	"syscall"
	"time"
)

const (
	gracefulTimeout = 5000 * time.Millisecond
)

func (p *Process) Terminate() error {
	osProc, err := os.FindProcess(int(p.Pid))
	if err != nil {
		return err
	}
	osProc.Signal(syscall.SIGTERM)
	osProc.Signal(syscall.SIGQUIT)
	osProc.Signal(syscall.SIGINT)

	ch := make(chan bool, 2)
	go func() {
		time.Sleep(gracefulTimeout)
		ch <- false
	}()
	go func() {
		osProc.Wait()
		ch <- true
	}()
	if !<-ch {
		osProc.Kill()
	}
	return nil
}
