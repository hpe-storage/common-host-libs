// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

// +build windows

// Package powershell wraps Windows Powershell cmdlets
package powershell

import (
	"strings"

	"github.com/hpe-storage/common-host-libs/util"
)

var (
	psExe        = "powershell.exe" // Executable that runs our PowerShell script
	psExitScript = "; exit -not $?" // Code to append to end of PS script to return pass/fail exit code
	psTrue       = "$True"          // Powershell true variable
	psFalse      = "$False"         // Poweshell false variable
)

// Internal helper function to convert the given Go boolean to a Powershell boolean string (i.e. $False or $True)
func psBoolToText(value bool) string {
	if value {
		return psTrue
	}
	return psFalse
}

// Internal helper routine the wraps the util.ExecCommandOutput routine.  Many PS cmdlets return extra CR/LF at
// the end of the return string.  This routine trims any trailing CR/LF to give a cleaner output string.  We
// also append psExitScript to the script so that the script exits with the correct exit code.
func execCommandOutput(arg string) (string, int, error) {
	arg += psExitScript
	output, rc, err := util.ExecCommandOutput(psExe, []string{"-command", arg})
	output = strings.TrimRight(output, "\r\n")
	return output, rc, err
}

// Internal helper routine the wraps the util.ExecCommandOutputWithTimeout routine.  Many PS cmdlets return extra
// CR/LF at the end of the return string.  This routine trims any trailing CR/LF to give a cleaner output string.
// We also append psExitScript to the script so that the script exits with the correct exit code.
func execCommandOutputWithTimeout(arg string, timeout int) (string, int, error) {
	arg += psExitScript
	output, rc, err := util.ExecCommandOutputWithTimeout(psExe, []string{"-command", arg}, timeout)
	output = strings.TrimRight(output, "\r\n")
	return output, rc, err
}
