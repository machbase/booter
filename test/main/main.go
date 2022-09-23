package main

import "github.com/machbase/booter"

func main() {
	booter.SetFlag(booter.ConfigFileFlag, "conf", "", "")
	booter.SetFlag(booter.PidFlag, "pid", "", "./boot.pid")
	booter.SetFlag(booter.DaemonFlag, "background", "d", "false")

	booter.Startup()
	booter.WaitSignal()
	booter.ShutdownAndExit(0)
}
