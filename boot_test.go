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
	booter.Register(AmodId,
		func() any {
			return new(AmodConf)
		},
		func(anyconf any) (booter.Bootable, error) {
			conf := anyconf.(*AmodConf)
			instance := &Amod{
				conf: conf,
			}
			return instance, nil
		})
	booter.Register(BmodId,
		func() any {
			return new(BmodConf)
		},
		func(anyconf any) (booter.Bootable, error) {
			conf := anyconf.(*BmodConf)
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
	b, err := booter.New("./test", []string{"--logging-default-enable-source-location"})
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
	assert.Equal(t, 51, bconf.DefaultPrefixWidth)
	assert.Equal(t, "ERROR", bconf.DefaultLevel)
	assert.Equal(t, true, bconf.DefaultEnableSourceLocation)
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
