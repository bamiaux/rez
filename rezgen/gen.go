// Copyright 2014 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	. "github.com/bamiaux/rez/asm"
	"io"
	"log"
	"os"
)

const (
	header = `// Copyright 2014 Benoît Amiaux. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// This file is auto-generated - do not modify

`
)

func main() {
	mode := flag.String("gen", "horizontal", "set which generator to use")
	flag.Parse()
	io.WriteString(os.Stdout, header)
	a := NewAsm(os.Stdout)
	switch *mode {
	case "horizontal":
		hgen(a)
	case "vertical":
		vgen(a)
	}
	err := a.Flush()
	if err != nil {
		log.Fatalln(err)
	}
}
