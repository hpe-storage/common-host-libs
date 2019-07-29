// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package winservice

import (
	"fmt"
	"strings"
	"time"

	log "github.com/hpe-storage/common-host-libs/logger"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

var elog debug.Log

// Execute is the thread executing the service and receiving control events
func (winService *WinService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	// Start the service
	winService.Start()
loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				// golang.org/x/sys/windows/svc.TestExample is verifying this output.
				testOutput := strings.Join(args, "-")
				testOutput += fmt.Sprintf("-%d", c.Context)
				if winService.UseEventLog {
					elog.Info(1, testOutput)
				}
				log.Infof("Stop/shutdown signal received, testOutput=%v, c.Cmd=%v", testOutput, c.Cmd)
				winService.Stop()
				break loop
			default:
				msg := fmt.Sprintf("unexpected control request #%d", c)
				if winService.UseEventLog {
					elog.Error(1, msg)
				}
				log.Error(msg)
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

// RunService is called to run the Windows service
func (winService WinService) RunService(name string, isDebug bool) {

	// If either the start or stop functions were not provided, log the error and abort.  This
	// would only occur if the winservice client was not programmed correctly.
	if (winService.Start == nil) || (winService.Stop == nil) {
		log.Errorf("WinService struct not initialized properly, StartProvided=%v, StopProvided=%v", (winService.Start != nil), (winService.Stop != nil))
		return
	}

	var err error
	msg := fmt.Sprintf("starting %s service", name)
	if winService.UseEventLog {
		if isDebug {
			elog = debug.New(name)
		} else {
			elog, err = eventlog.Open(name)
			if err != nil {
				return
			}
		}
		defer elog.Close()

		elog.Info(1, msg)
	}
	log.Info(msg)
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(name, &winService)
	if err != nil {
		msg = fmt.Sprintf("%s service failed: %v", name, err)
		if winService.UseEventLog {
			elog.Error(1, msg)
		}
		log.Error(msg)
		return
	}
	msg = fmt.Sprintf("%s service stopped", name)
	if winService.UseEventLog {
		elog.Info(1, msg)
	}
	log.Info(msg)
}
