package booter

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Daemon      bool
	BootlogFile string
	PidFile     string
	Pname       string
	ConfDir     string

	flags         map[BootFlagType]BootFlag
	versionString string
}

type BootFlag struct {
	Long        string
	Short       string
	Placeholder string
	Help        string
	Default     string
}

type BootFlagType int

const (
	noneFlag BootFlagType = iota
	ConfigFlag
	PnameFlag
	PidFlag
	BootlogFlag
	DaemonFlag
	HelpFlag
	numofFlags
)

var defaultBooter Booter
var defaultBuilder = NewBuilder()
var conf *Config
var bootlog *log.Logger

func init() {
	bootlog = log.New(os.Stdout, "booter ", log.LstdFlags|log.Lmsgprefix)
	conf = &Config{
		flags: map[BootFlagType]BootFlag{
			ConfigFlag:  {Long: "config-dir", Short: "c", Placeholder: "<path>", Help: "config directory path"},
			PnameFlag:   {Long: "pname", Placeholder: "<name>", Help: "assign process name"},
			PidFlag:     {Long: "pid", Placeholder: "<path>", Help: "pid file path"},
			BootlogFlag: {Long: "bootlog", Placeholder: "<path>", Help: "boot log path"},
			DaemonFlag:  {Long: "daemon", Short: "d", Help: "run process in background, daemonize"},
			HelpFlag:    {Long: "help", Short: "h", Help: "print this message"},
		},
	}
}

func Startup() {
	parseflags()

	if conf.Daemon {
		// daemon mode일 때는 bootlog와 pidfile을 Damonize()내에서 처리한다.
		Daemonize(conf.BootlogFile, conf.PidFile, func() { serve(conf) })
		return
	}

	// foreground process mode일 때 bootlog와 pidfile을 생성 한다.
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

	if len(conf.PidFile) > 0 && !conf.Daemon {
		pfile, _ := os.OpenFile(conf.PidFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		pfile.WriteString(fmt.Sprintf("%d", os.Getpid()))
		pfile.Close()
	}

	serve(conf)
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

func SetFlag(flagType BootFlagType, longflag, shortflag, defaultValue string) {
	if flag, ok := conf.flags[flagType]; ok {
		flag.Long = longflag
		flag.Short = shortflag
		flag.Default = defaultValue
		conf.flags[flagType] = flag
	} else {
		panic(fmt.Errorf("invalid flag type: %d", flagType))
	}
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

func usage() {
	bin, _ := os.Executable()
	bin = filepath.Base(bin)
	fmt.Println(bin, "flags...")

	var maxlen = 0
	for _, v := range conf.flags {
		l := len(v.Long) + len(v.Short) + len(v.Placeholder)
		if maxlen < l {
			maxlen = l
		}
	}

	var uses = map[BootFlagType]string{}
	for k, v := range conf.flags {
		var format = "  %%s"
		if len(v.Default) > 0 {
			format = fmt.Sprintf("    %%-%ds  %s (default %s)", maxlen+5, v.Help, v.Default)
		} else {
			format = fmt.Sprintf("    %%-%ds  %s", maxlen+5, v.Help)
		}
		use := ""
		if len(v.Short) > 0 {
			use = fmt.Sprintf("-%s", v.Short)
		}
		if len(v.Long) > 0 {
			if len(use) > 0 {
				use = fmt.Sprintf("%s, --%s", use, v.Long)
			} else {
				use = fmt.Sprintf("--%s", v.Long)
			}
		}
		if len(v.Placeholder) > 0 {
			use = fmt.Sprintf("%s %s", use, v.Placeholder)
		}
		line := fmt.Sprintf(format, use)
		uses[k] = line
	}

	for i := 1; i < int(numofFlags); i++ {
		fmt.Println(uses[BootFlagType(i)])
	}
}

func parseflags() {
	flag.Usage = usage

	// init with default values
	for k, v := range conf.flags {
		switch k {
		case ConfigFlag:
			conf.ConfDir = v.Default
		case PnameFlag:
			conf.Pname = v.Default
		case PidFlag:
			conf.PidFile = v.Default
		case BootlogFlag:
			conf.BootlogFile = v.Default
		case DaemonFlag:
			if len(v.Default) > 0 {
				if b, err := strconv.ParseBool(v.Default); err != nil {
					panic(err)
				} else {
					conf.Daemon = b
				}
			}
		}
	}

	// parse args
	for i := 0; i < len(os.Args); i++ {
		argn := os.Args[i]
		argv := ""
		if i < len(os.Args)-1 {
			argv = os.Args[i+1]
		}

		var matched BootFlagType = noneFlag
		for f, v := range conf.flags {
			if strings.HasPrefix(argn, "--") && argn[2:] == v.Long {
				matched = f
				break
			} else if strings.HasPrefix(argn, "-") && argn[1:] == v.Short {
				matched = f
				break
			}
		}

		switch matched {
		case ConfigFlag:
			conf.ConfDir = argv
		case PnameFlag:
			conf.Pname = argv
		case PidFlag:
			conf.PidFile = argv
		case BootlogFlag:
			conf.BootlogFile = argv
		case DaemonFlag:
			conf.Daemon = true
		case HelpFlag:
			flag.Usage()
			os.Exit(0)
		}
	}

	if len(conf.ConfDir) == 0 {
		fmt.Printf("\n  Error: --%s is required\n\n", conf.flags[ConfigFlag].Long)
		flag.Usage()
		os.Exit(1)
	}
}
