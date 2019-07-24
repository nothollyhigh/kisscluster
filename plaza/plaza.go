package main

import (
	"github.com/nothollyhigh/kiss/util"
	"kisscluster/plaza/app"
	"os"
	"syscall"
)

var Version = ""

func main() {
	app.Run(Version)

	util.HandleSignal(func(sig os.Signal) {
		if sig == syscall.SIGTERM || sig == syscall.SIGINT {
			app.Stop()
			os.Exit(0)
		}
	})
}
