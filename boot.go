package booter

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

type Boot interface {
	Startup() error
	Shutdown()

	WaitSignal()
	NotifySignal()

	GetDefinition(id string) *Definition
	GetInstance(id string) Bootable
	GetEnvContext() *EnvContext
}

type boot struct {
	moduleDefs []*Definition
	modules    []wrapper
	quitChan   chan os.Signal
	envCtx     *EnvContext
}

type wrapper struct {
	id    string
	real  Bootable
	state State
}

type State int

const (
	None State = iota
	PreStart
	PostStart
	Running
	Stopping
	Stop
)

func New(configDir string, args []string) (Boot, error) {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, errors.Wrap(err, "invalid config directory")
	}

	envfile := ""
	files := make([]string, 0)
	if _, err := os.Stat(filepath.Join(configDir, "env.hcl")); err == nil {
		envfile = filepath.Join(configDir, "env.hcl")
	}

	for _, file := range entries {
		if !strings.HasSuffix(file.Name(), ".hcl") || file.Name() == "env.hcl" {
			continue
		}
		files = append(files, filepath.Join(configDir, file.Name()))
	}
	return NewWithFiles(args, envfile, files...)
}

func NewWithFiles(args []string, envfile string, files ...string) (Boot, error) {
	b := &boot{}
	if len(envfile) > 0 {
		if _, err := os.Stat(envfile); err == nil {
			b.envCtx, err = LoadEnvContext(envfile)
			if err != nil {
				return nil, errors.Wrap(err, "env file error")
			}
		}
	}

	cfgs := make([]*Definition, 0)
	for _, file := range files {
		cs, err := loadModuleConfig(b.envCtx, file, args)
		if err != nil {
			return nil, err
		}
		cfgs = append(cfgs, cs...)
	}
	sort.Slice(cfgs, func(i, j int) bool {
		return cfgs[i].Priority < cfgs[j].Priority
	})
	b.moduleDefs = cfgs
	return b, nil
}

type Hook map[string]func()

var PostStartHook map[string]func()
var PreStartHook map[string]func()
var PreStopHook map[string]func()

func (this *boot) Startup() error {
	for _, def := range this.moduleDefs {
		fact := getBootFactory(def.Id)
		mod, err := fact.NewInstance(def.Config)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("mod NewInstance %s", def.Id))
		}
		wrap := wrapper{id: def.Id, real: mod, state: None}
		this.modules = append(this.modules, wrap)

		wrap.state = PreStart
		if hook, ok := PreStartHook[def.Id]; ok {
			hook()
		}

		err = mod.Start()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("mod start %s", def.Id))
		}
		wrap.state = PostStart
		if hook, ok := PostStartHook[def.Id]; ok {
			hook()
		}
		wrap.state = Running
	}
	return nil
}

func (this *boot) Shutdown() {
	for i := len(this.modules); i >= 0; i-- {
		mod := this.modules[i]
		mod.state = Stopping
		if hook, ok := PreStopHook[mod.id]; ok {
			hook()
		}
		instance := mod.real
		instance.Stop()
		mod.state = Stop
	}
}

func (this *boot) WaitSignal() {
	// signal handler
	this.quitChan = make(chan os.Signal)
	signal.Notify(this.quitChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// wait signal
	<-this.quitChan
}

func (this *boot) NotifySignal() {
	if this.quitChan != nil {
		this.quitChan <- syscall.SIGINT
	}
}

func (this *boot) GetDefinition(id string) *Definition {
	for _, def := range this.moduleDefs {
		if def.Id == id {
			return def
		}
	}
	return nil
}

func (this *boot) GetInstance(id string) Bootable {
	for _, mod := range this.modules {
		if mod.id == id {
			return mod.real
		}
	}
	return nil
}

func (this *boot) GetEnvContext() *EnvContext {
	return this.envCtx
}
