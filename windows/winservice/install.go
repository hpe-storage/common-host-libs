// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package winservice

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

func exePath() (string, error) {
	prog := os.Args[0]
	p, err := filepath.Abs(prog)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(p)
	if err == nil {
		if !fi.Mode().IsDir() {
			return p, nil
		}
		err = fmt.Errorf("%s is directory", p)
	}
	if filepath.Ext(p) == "" {
		p += ".exe"
		fi, err := os.Stat(p)
		if err == nil {
			if !fi.Mode().IsDir() {
				return p, nil
			}
			err = fmt.Errorf("%s is directory", p)
		}
	}
	return "", err
}

// InstallService is used to install the service
func (winService WinService) InstallService(name, displayName string, description string) error {
	return winService.InstallServiceWithOptions(name, mgr.Config{DisplayName: displayName, Description: description}, nil, 0)
}

// InstallServiceWithOptions is used to install the service with the provided configuration and
// recovery options.
func (winService WinService) InstallServiceWithOptions(name string, config mgr.Config, recoveryActions []mgr.RecoveryAction, resetPeriod uint32) error {
	exepath, err := exePath()
	if err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", name)
	}
	s, err = m.CreateService(name, exepath, config)
	if err != nil {
		return err
	}
	defer s.Close()
	if winService.UseEventLog {
		err = eventlog.InstallAsEventCreate(name, eventlog.Error|eventlog.Warning|eventlog.Info)
		if err != nil {
			s.Delete()
			return fmt.Errorf("SetupEventLogSource() failed: %s", err)
		}
	}

	// If no recovery options provided, service has been successfully installed
	if len(recoveryActions) == 0 {
		return nil
	}

	// Set the service recovery options (e.g. automatically restart service on a crash)
	return s.SetRecoveryActions(recoveryActions, resetPeriod)
}

// RemoveService is used to uninstall the service
func (winService WinService) RemoveService(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", name)
	}
	defer s.Close()
	err = s.Delete()
	if err != nil {
		return err
	}

	if winService.UseEventLog {
		err = eventlog.Remove(name)
		if err != nil {
			return fmt.Errorf("RemoveEventLogSource() failed: %s", err)
		}
	}
	return nil
}
