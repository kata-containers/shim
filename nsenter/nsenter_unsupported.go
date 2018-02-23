// Copyright 2018 HyperHQ Inc.
//
// SPDX-License-Identifier: Apache-2.0
//

// +build !linux !cgo

package nsenter

func NsEnter(nsPid int) bool {
	return false
}
