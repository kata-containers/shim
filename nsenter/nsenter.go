// +build linux

// Copyright 2018 HyperHQ Inc.
//
// SPDX-License-Identifier: Apache-2.0
//

package nsenter

/*
#cgo CFLAGS: -Wall
extern int nsenter(int pid);
*/
import "C"

func NsEnter(nsPid int) bool {
	return C.nsenter(C.int(nsPid)) >= 0
}
