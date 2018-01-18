// Copyright 2017 HyperHQ Inc.
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"errors"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/moby/moby/pkg/term"
	context "golang.org/x/net/context"

	pb "github.com/kata-containers/agent/protocols/grpc"
)

const sigChanSize = 2048

var sigIgnored = map[syscall.Signal]bool{
	syscall.SIGCHLD:  true,
	syscall.SIGPIPE:  true,
	syscall.SIGWINCH: true,
	syscall.SIGBUS:   true,
	syscall.SIGSEGV:  true,
	syscall.SIGABRT:  true,
}

type shim struct {
	containerID string
	pid         uint32

	ctx   context.Context
	agent *shimAgent
}

func newShim(addr, containerID string, pid uint32) (*shim, error) {
	agent, err := newShimAgent(addr)
	if err != nil {
		return nil, err
	}

	return &shim{containerID: containerID,
		pid:   pid,
		ctx:   context.Background(),
		agent: agent}, nil
}

func (s *shim) proxyStdio(wg *sync.WaitGroup) {
	// don't wait the copying of the stdin, because `io.Copy(inPipe, os.Stdin)`
	// can't terminate when no input. todo: find a better way.
	wg.Add(2)
	inPipe, outPipe, errPipe := shimStdioPipe(s.ctx, s.agent, s.containerID, s.pid)
	go func() {
		_, err1 := io.Copy(inPipe, os.Stdin)
		_, err2 := s.agent.CloseStdin(s.ctx, &pb.CloseStdinRequest{
			ContainerId: s.containerID,
			PID:         s.pid})
		if err1 != nil {
			shimLog.WithError(err1).Warn("copy stdin failed")
		}
		if err2 != nil {
			shimLog.WithError(err2).Warn("close stdin failed")
		}
	}()

	go func() {
		_, err := io.Copy(os.Stdout, outPipe)
		shimLog.WithError(err).Info("copy stdout failed")
		wg.Done()
	}()

	go func() {
		_, err := io.Copy(os.Stderr, errPipe)
		shimLog.WithError(err).Info("copy stderr failed")
		wg.Done()
	}()
}

func (s *shim) forwardAllSignals() chan os.Signal {
	sigc := make(chan os.Signal, sigChanSize)
	// handle all signals for the process.
	signal.Notify(sigc)
	signal.Ignore(syscall.SIGCHLD, syscall.SIGPIPE)

	go func() {
		for sig := range sigc {
			sysSig, ok := sig.(syscall.Signal)
			if !ok {
				err := errors.New("unknown signal")
				shimLog.WithError(err).WithField("signal", sig.String()).Error("")
				continue
			}
			if sigIgnored[sysSig] {
				//ignore these
				continue
			}
			// forward this signal to container
			_, err := s.agent.SignalProcess(s.ctx, &pb.SignalProcessRequest{
				ContainerId: s.containerID,
				PID:         s.pid,
				Signal:      uint32(sysSig)})
			if err != nil {
				shimLog.WithError(err).WithField("signal", sig.String()).Error("forward signal failed")
			}
		}
	}()
	return sigc
}

func (s *shim) resizeTty(fromTty *os.File) error {
	ws, err := term.GetWinsize(fromTty.Fd())
	if err != nil {
		shimLog.WithError(err).Info("Error getting size")
		return nil
	}

	_, err = s.agent.TtyWinResize(s.ctx, &pb.TtyWinResizeRequest{
		ContainerId: s.containerID,
		PID:         s.pid,
		Row:         uint32(ws.Height),
		Column:      uint32(ws.Width)})
	if err != nil {
		shimLog.WithError(err).Error("set winsize failed")
	}

	return err
}

func (s *shim) monitorTtySize(tty *os.File) {
	s.resizeTty(tty)
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGWINCH)
	go func() {
		for range sigchan {
			s.resizeTty(tty)
		}
	}()
}

func (s *shim) wait() (int32, error) {
	resp, err := s.agent.WaitProcess(s.ctx, &pb.WaitProcessRequest{
		ContainerId: s.containerID,
		PID:         s.pid})
	if err != nil {
		return 0, err
	}

	return resp.Status, nil
}
