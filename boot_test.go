package booter_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/machbase/booter"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	"github.com/stretchr/testify/require"
)

var AmodId = "github.com/booter/amod"
var BmodId = "github.com/booter/bmod"

func TestMain(m *testing.M) {
	os.Args = []string{
		"--logging-default-level", "WARN",
		"--logging-default-enable-source-location", "true",
		"--logging-default-prefix-width", "30",
	}

	booter.Register(AmodId,
		func() *AmodConf {
			return new(AmodConf)
		},
		func(conf *AmodConf) (booter.Boot, error) {
			instance := &Amod{
				conf: conf,
			}
			return instance, nil
		})
	booter.Register(BmodId,
		func() *BmodConf {
			return new(BmodConf)
		},
		func(conf *BmodConf) (booter.Boot, error) {
			instance := &Bmod{
				conf: *conf,
			}
			return instance, nil
		})

	var GetVersionFunc = function.New(&function.Spec{
		Params: []function.Parameter{},
		Type:   function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			return cty.StringVal("1.2.3"), nil
		},
	})

	booter.SetFunction("version", GetVersionFunc)
	m.Run()
}

func TestParser(t *testing.T) {
	defs, err := booter.LoadDefinitionFiles([]string{
		"./test/env.hcl",
		"./test/mod_amod.hcl",
		"./test/mod_bmod.hcl",
		"./test/mod_others.hcl",
	})
	require.Nil(t, err)
	require.Equal(t, 3, len(defs))
}

func TestBoot(t *testing.T) {
	builder := booter.NewBuilder()
	b, err := builder.BuildWithDir("./test")
	require.Nil(t, err)

	err = b.Startup()
	require.Nil(t, err)

	def := b.GetDefinition(AmodId)
	require.NotNil(t, def)
	require.Equal(t, 201, def.Priority)
	require.Equal(t, false, def.Disabled)
	aconf := b.GetConfig(AmodId).(*AmodConf)
	require.Equal(t, true, aconf.TcpConfig.Tls.LoadPrivateCAs)
	require.Equal(t, "./test/test_server_cert.pem", aconf.TcpConfig.Tls.CertFile)
	require.Equal(t, "./test/test_server_key.pem", aconf.TcpConfig.Tls.KeyFile)
	require.Equal(t, 5*time.Second, aconf.TcpConfig.Tls.HandshakeTimeout)
	require.Equal(t, "1.2.3", aconf.Version)
	// check if injection works
	amod := b.GetInstance(AmodId).(*Amod)
	require.NotNil(t, amod)
	require.NotNil(t, amod.Bmod)

	def = b.GetDefinition(BmodId)
	require.NotNil(t, def)
	require.Equal(t, 202, def.Priority)
	require.Equal(t, false, def.Disabled)
	bconf := b.GetConfig(BmodId).(*BmodConf)
	require.Equal(t, fmt.Sprintf("%s/./tmp/cmqd00.log", os.Getenv("HOME")), bconf.Filename)
	require.Equal(t, true, bconf.Append)
	require.Equal(t, "@midnight", bconf.RotateSchedule)
	require.Equal(t, 3, bconf.MaxBackups)
	require.Equal(t, 3, len(bconf.Levels))
	require.Equal(t, 30, bconf.DefaultPrefixWidth)
	require.Equal(t, "WARN", bconf.DefaultLevel)
	require.Equal(t, true, bconf.DefaultEnableSourceLocation)

	b.Shutdown()
}

type AmodConf struct {
	Version   string
	TcpConfig TcpConfig
}

type TcpConfig struct {
	ListenAddress    string
	AdvertiseAddress string
	SoLinger         int
	KeepAlive        int
	NoDelay          bool
	Tls              TlsConfig
}

type TlsConfig struct {
	LoadSystemCAs    bool
	LoadPrivateCAs   bool
	CertFile         string
	KeyFile          string
	HandshakeTimeout time.Duration
}

type Amod struct {
	conf             *AmodConf
	Bmod             *Bmod
	OtherNameForBmod *Bmod
}

func (this *Amod) Start() error {
	fmt.Println("amod start")
	fmt.Printf("    with Amod.Bmod             = %p\n", this.Bmod)
	fmt.Printf("    with Amod.OtherNameForBmod = %p\n", this.OtherNameForBmod)
	if this.Bmod != this.OtherNameForBmod {
		return errors.New("amod.Bmod and amod.OtherNameForBmod has different references")
	}
	return nil
}

func (this *Amod) Stop() {
	fmt.Println("amod stop")
}

type BmodConf struct {
	Filename                    string
	Append                      bool
	MaxBackups                  int
	RotateSchedule              string
	DefaultLevel                string
	DefaultPrefixWidth          int
	DefaultEnableSourceLocation bool
	Levels                      []LevelConf
}

type LevelConf struct {
	Pattern string
	Level   string
}

type Bmod struct {
	conf BmodConf
}

func (this *Bmod) Start() error {
	fmt.Println("bmod start")
	return nil
}

func (this *Bmod) Stop() {
	fmt.Println("bmod stop")
}
