package booter

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

type Config struct {
	Daemon      bool
	BootlogFile string
	PidFile     string
	Pname       string
	ConfDir     string

	versionString string
}

var defaultBooter Booter
var defaultBuilder = NewBuilder()
var conf *Config
var bootlog *log.Logger

func init() {
	bootlog = log.New(os.Stdout, "booter ", log.LstdFlags|log.Lmsgprefix)
}

func Startup() {
	conf = &Config{
		Daemon:      false,
		BootlogFile: "./boot.log",
		PidFile:     "./boot.pid",
		Pname:       "noname",
	}

	flag.Usage = func() {
		bin, _ := os.Executable()
		bin = filepath.Base(bin)
		fmt.Println(bin, "options...")
		fmt.Println("   --config-dir, -c <path> ", "config directory path")
		fmt.Println("   --pname <name>          ", "assign process name (default noname)")
		fmt.Println("   --pid <path>            ", "write pid (default ./boot.pid)")
		fmt.Println("   --bootlog <path>        ", "boot log path (default ./boot.log)")
		fmt.Println("   --daemon, -d            ", "run process in background, daemonize")
		fmt.Println("   --help, -h              ", "print this message")
		fmt.Println("")
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
		case "--help", "-h":
			flag.Usage()
			os.Exit(0)
		}
	}

	if len(conf.ConfDir) == 0 {
		panic(errors.New("--config-dir required"))
	}

	var writer io.Writer
	if len(conf.BootlogFile) > 0 {
		logfile, _ := os.OpenFile(conf.BootlogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		defer logfile.Close()
		if conf.Daemon {
			writer = logfile
		} else {
			writer = io.MultiWriter(os.Stdout, logfile)
		}
	} else {
		if conf.Daemon {
			writer = io.Discard
		} else {
			writer = os.Stdout
		}
	}
	bootlog = log.New(writer, fmt.Sprintf("boot-%s ", conf.Pname), log.LstdFlags|log.Lmsgprefix)
	bootlog.Println("pid:", os.Getpid())

	if len(conf.PidFile) > 0 {
		pfile, _ := os.OpenFile(conf.PidFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		pfile.WriteString(fmt.Sprintf("%d", os.Getpid()))
		pfile.Close()
	}

	if conf.Daemon {
		Daemonize(conf.BootlogFile, conf.PidFile, func() { serve(conf) })
	} else {
		serve(conf)
	}
}

func Shutdown() {
	defaultBooter.Shutdown()
}

func WaitSignal() {
	defaultBooter.WaitSignal()
}

func NotifySignal() {
	defaultBooter.NotifySignal()
}

func AddStartupHook(hooks ...func()) {
	defaultBuilder.AddStartupHook(hooks...)
}

func AddShutdownHook(hooks ...func()) {
	// booter가 시작되고 나면 builder에 hook을 추가하는 것은 의미가 없다.
	if defaultBooter == nil {
		defaultBuilder.AddShutdownHook(hooks...)
	} else {
		defaultBooter.AddShutdownHook(hooks...)
	}
}

func GetDefinition(id string) *Definition {
	return defaultBooter.GetDefinition(id)
}

func GetInstance(id string) Boot {
	return defaultBooter.GetInstance(id)
}

func GetConfig(id string) any {
	return defaultBooter.GetInstance(id)
}

func Pname() string {
	return conf.Pname
}

func VersionString() string {
	return conf.versionString
}

func SetVersionString(str string) {
	conf.versionString = str
}

func serve(conf *Config) {
	var err error
	defaultBooter, err = defaultBuilder.BuildWithDir(conf.ConfDir)
	if err != nil {
		panic(err)
	}

	bootlog.Println("startup", conf.Pname)
	err = defaultBooter.Startup()
	if err != nil {
		panic(err)
	}

	defaultBooter.WaitSignal()

	bootlog.Println("shutdown", conf.Pname)
	defaultBooter.Shutdown()
}
