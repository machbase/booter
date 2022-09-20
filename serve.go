package booter

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type Config struct {
	Daemon      bool
	BootlogFile string
	PidFile     string
	Pname       string
	ConfDir     string
}

var Server Boot
var conf *Config
var bootlog *log.Logger

func init() {
	bootlog = log.New(os.Stdout, "booter", log.LstdFlags|log.Lmsgprefix)
}

func Main() {
	conf = &Config{
		Daemon:      false,
		BootlogFile: "./boot.log",
		PidFile:     "./boot.pid",
		Pname:       "noname",
	}
	for i := 0; i < len(os.Args); i++ {
		if len(os.Args[i]) <= 2 || os.Args[i][0] != '-' || os.Args[i][1] != '-' {
			continue
		}
		switch os.Args[i] {
		case "--daemon", "-d":
			conf.Daemon = true
		case "--bootlog":
			conf.BootlogFile = os.Args[i+1]
		case "--pid":
			conf.PidFile = os.Args[i+1]
		case "--pname":
			conf.Pname = os.Args[i+1]
		case "--config-dir", "-c":
			conf.ConfDir = os.Args[i+1]
		}
	}

	if len(conf.ConfDir) == 0 {
		panic(errors.New("--config-dir required"))
	}

	var writer io.Writer
	if len(conf.BootlogFile) > 0 {
		logfile, _ := os.OpenFile(conf.BootlogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		defer logfile.Close()
		writer = io.MultiWriter(os.Stdout, logfile)
	} else {
		writer = os.Stdout
	}
	bootlog = log.New(writer, fmt.Sprintf("boot-%s", conf.Pname), log.LstdFlags|log.Lmsgprefix)
	bootlog.Println("pid:", os.Getpid())

	if len(conf.PidFile) > 0 {
		pfile, _ := os.OpenFile(conf.PidFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		pfile.WriteString(fmt.Sprintf("%d", os.Getpid()))
		pfile.Close()
	}

	if conf.Daemon {
		Daemonize(conf.BootlogFile, conf.PidFile, func() { Serve(conf) })
	} else {
		Serve(conf)
	}
}

func Pname() string {
	return conf.Pname
}

func Shutdown() {
	Server.Shutdown()
}

func WaitSignal() {
	Server.WaitSignal()
}

func NotifySignal() {
	Server.NotifySignal()
}

func Serve(conf *Config) {
	entries, err := os.ReadDir(conf.ConfDir)
	if err != nil {
		panic(err)
	}

	files := make([]string, 0)
	for _, file := range entries {
		if !strings.HasSuffix(file.Name(), ".hcl") {
			continue
		}
		path := filepath.Join(conf.ConfDir, file.Name())
		files = append(files, path)
	}

	Server, err = NewWithFiles(files)
	if err != nil {
		panic(err)
	}

	bootlog.Println("startup", conf.Pname)
	err = Server.Startup()
	if err != nil {
		panic(err)
	}

	Server.WaitSignal()

	bootlog.Println("shutdown", conf.Pname)
	Server.Shutdown()
}
