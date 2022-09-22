package main

import "github.com/machbase/booter"

func main() {
	booter.SetFlag(booter.ConfigFlag, "conf", "", "")
	booter.SetFlag(booter.PidFlag, "pid", "", "./boot.pid")
	booter.SetFlag(booter.DaemonFlag, "background", "d", "false")

	booter.Startup()
	booter.WaitSignal()
	booter.Shutdown()
}
