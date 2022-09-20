package booter_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/machbase/booter"

	"github.com/stretchr/testify/assert"
)

var AmodId = "github.com/amod"
var BmodId = "github.com/bmod"

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
		func(conf *AmodConf) (booter.Bootable, error) {
			instance := &Amod{
				conf: conf,
			}
			return instance, nil
		})
	booter.Register(BmodId,
		func() *BmodConf {
			return new(BmodConf)
		},
		func(conf *BmodConf) (booter.Bootable, error) {
			instance := &Bmod{
				conf: *conf,
			}
			return instance, nil
		})
	m.Run()
}

func TestParser(t *testing.T) {
	defs, err := booter.LoadDefinitions([]string{
		"./test/env.hcl",
		"./test/mod_amod.hcl",
		"./test/mod_bmod.hcl",
		"./test/mod_others.hcl",
	})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(defs))
}

func TestBoot(t *testing.T) {
	b, err := booter.New("./test")
	assert.Nil(t, err)

	err = b.Startup()
	assert.Nil(t, err)

	def := b.GetDefinition(AmodId)
	assert.NotNil(t, def)
	assert.Equal(t, 201, def.Priority)
	assert.Equal(t, false, def.Disabled)
	aconf := b.GetConfig(AmodId).(*AmodConf)
	assert.Equal(t, true, aconf.TcpConfig.Tls.LoadPrivateCAs)
	assert.Equal(t, "./test/test_server_cert.pem", aconf.TcpConfig.Tls.CertFile)
	assert.Equal(t, "./test/test_server_key.pem", aconf.TcpConfig.Tls.KeyFile)
	assert.Equal(t, 5*time.Second, aconf.TcpConfig.Tls.HandshakeTimeout)

	def = b.GetDefinition(BmodId)
	assert.NotNil(t, def)
	assert.Equal(t, 202, def.Priority)
	assert.Equal(t, false, def.Disabled)
	bconf := b.GetConfig(BmodId).(*BmodConf)
	assert.Equal(t, fmt.Sprintf("%s/./tmp/cmqd00.log", os.Getenv("HOME")), bconf.Filename)
	assert.Equal(t, true, bconf.Append)
	assert.Equal(t, "@midnight", bconf.RotateSchedule)
	assert.Equal(t, 3, bconf.MaxBackups)
	assert.Equal(t, 3, len(bconf.Levels))
	assert.Equal(t, 30, bconf.DefaultPrefixWidth)
	assert.Equal(t, "WARN", bconf.DefaultLevel)
	assert.Equal(t, true, bconf.DefaultEnableSourceLocation)

	b.Shutdown()
}

type AmodConf struct {
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
	conf *AmodConf
}

func (this *Amod) Start() error {
	fmt.Println("amod start")
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
