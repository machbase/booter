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

func TestBoot(t *testing.T) {
	b, err := booter.New("./test", []string{"--logging-default-enable-source-location"})
	assert.Nil(t, err)
	b.Startup()

	def := b.GetDefinition(AmodId)
	assert.NotNil(t, def)
	assert.Equal(t, 100, def.Priority)
	assert.Equal(t, false, def.Disabled)
	// conf := def.Config.(*logging.Config)
	// assert.Equal(t, fmt.Sprintf("%s/./tmp/cmqd00.log", os.Getenv("HOME")), conf.Filename)
	// assert.Equal(t, true, conf.Append)
	// assert.Equal(t, "@midnight", conf.RotateSchedule)
	// assert.Equal(t, 3, conf.MaxBackups)
	// assert.Equal(t, 3, len(conf.Levels))
	// assert.Equal(t, 51, conf.DefaultPrefixWidth)
	// assert.Equal(t, "ERROR", conf.DefaultLevel)
	// assert.Equal(t, true, conf.DefaultEnableSourceLocation)

	def = b.GetDefinition(BmodId)
	assert.Nil(t, def)
	// assert.NotNil(t, def)
	// assert.Equal(t, 100, def.Priority)
	// assert.Equal(t, false, def.Disabled)
	// mqttConf := def.Config.(*mqtt.MqttConfig)
	// assert.Equal(t, true, mqttConf.TcpConfig.Tls.LoadPrivateCAs)
	// assert.Equal(t, "./test/test_server_cert.pem", mqttConf.TcpConfig.Tls.CertFile)
	// assert.Equal(t, "./test/test_server_key.pem", mqttConf.TcpConfig.Tls.KeyFile)
	// assert.Equal(t, 5*time.Second, mqttConf.TcpConfig.Tls.HandshakeTimeout)

	var content = `
		name          = "hong gil ${upper("dong")}"
		age           = 20
		fiction-story = true
		home-path     = env("HOME")
		user-path     = "${env("HOME")}/data"
	`

	type Target struct {
		Name         string
		Age          int
		FictionStory bool
		HomePath     string
		UserPath     string
	}

	var obj = &Target{}
	var envctx = b.GetEnvContext()

	err = booter.ParseWithContext(envctx, []byte(content), obj)
	assert.Nil(t, err)
	assert.Equal(t, "hong gil DONG", obj.Name)
	assert.Equal(t, 20, obj.Age)
	assert.Equal(t, true, obj.FictionStory)
	assert.Equal(t, os.Getenv("HOME"), obj.HomePath)
	assert.Equal(t, fmt.Sprintf("%s/data", os.Getenv("HOME")), obj.UserPath)
	// fmt.Printf("\n%#v\n", obj)
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
