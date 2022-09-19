package booter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	konghcl "github.com/alecthomas/kong-hcl"
)

type Config struct {
	Config        kong.ConfigFlag `short:"c" type:"existingfile" placeholder:"<config_path>" env:"BOOT_CONFIG"`
	Daemon        bool            `short:"d" help:"run in background, daemonize"`
	BootlogFile   string          `default:"./boot.log"`
	PidFile       string          `default:"./boot.pid"`
	Pname         string          `default:"noname"`
	ModuleEnvFile string          `default:""`
	ModuleConfDir string          `default:""`
	Args          []string        `arg:"" optional:"" passthrough:""`
}

var Server Boot
var conf *Config

func Serve(args []string) {
	conf = &Config{}
	parser, err := kong.New(conf, kong.Configuration(konghcl.Loader))
	if err != nil {
		panic(err)
	}
	_, err = parser.Parse(args)
	if err != nil {
		panic(err)
	}
	if conf.Daemon {
		Daemonize(conf.BootlogFile, conf.PidFile, func() { serve(conf) })
	} else {
		serve(conf)
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

func serve(conf *Config) {
	entries, err := os.ReadDir(conf.ModuleConfDir)
	if err != nil {
		panic(err)
	}

	files := make([]string, 0)
	for _, file := range entries {
		if !strings.HasSuffix(file.Name(), ".hcl") {
			continue
		}
		path := filepath.Join(conf.ModuleConfDir, file.Name())
		if path == conf.ModuleEnvFile {
			continue
		}
		files = append(files, path)
	}

	Server, err = NewWithFiles(conf.Args, conf.ModuleEnvFile, files...)
	if err != nil {
		panic(err)
	}

	fmt.Println("boot startup", conf.Pname)
	Server.Startup()

	Server.WaitSignal()

	fmt.Println("boot shutdown", conf.Pname)
	Server.Shutdown()
}
