// Copyright 2014 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// +build !amd64

package rez

func hasAsm() bool { return false }

var (
	h8scale2  = h8scale2Go
	h8scale4  = h8scale4Go
	h8scale6  = h8scale6Go
	h8scale8  = h8scale8Go
	h8scale10 = h8scale10Go
	h8scale12 = h8scale12Go
	h8scaleN  = h8scaleNGo
	v8scale2  = v8scale2Go
	v8scale4  = v8scale4Go
	v8scale6  = v8scale6Go
	v8scale8  = v8scale8Go
	v8scale10 = v8scale10Go
	v8scale12 = v8scale12Go
	v8scaleN  = v8scaleNGo
)
