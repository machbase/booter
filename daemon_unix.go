//go:build !windows
// +build !windows

package booter

import (
	"fmt"

	"github.com/sevlyar/go-daemon"
)

func Daemonize(bootlog string, pidfile string, proc func()) {
	context := daemon.Context{LogFileName: bootlog, PidFileName: pidfile}

	child, err := context.Reborn()
	if err != nil {
		context := daemon.Context{}
		child, err = context.Reborn()
		if err != nil {
			panic(fmt.Sprintf("Unable to run %s", err.Error()))
		}
	}
	if child != nil {
		return
	}
	defer context.Release()
	proc()
}
