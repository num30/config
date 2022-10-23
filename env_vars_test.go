package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/iamolegga/enviper"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var env map[string]string

func init() {

}

func TestThrowsErrorWhenBrokenConfig(t *testing.T) {
	p := path.Join("testdata", "test_config.yaml")
	ioutil.WriteFile(p, []byte("qwerty"), 0777)
	defer os.Remove(p)

	conf := NewConfReader("test_config").WithSearchDirs("testdata")
	c := Config{}
	assert.Error(t, conf.Read(&c))
}

func TestOnlyEnvs(t *testing.T) {
	loadEnvVars()
	setupEnvConfig(t)
	defer tearDownEnvConfig(t)

	var c Config
	conf := NewConfReader("fixture").WithPrefix("PREF")
	err := conf.Read(&c)

	getEnv := os.Getenv("PREF_FOO")
	assert.Equal(t, "fooooo", getEnv, "env var should be set")

	if assert.NoError(t, err) {
		assert.Equal(t, "fooooo", c.Foo)
		assert.Equal(t, 2, c.Bar.BAZ)
		assert.Equal(t, false, c.QUX.Quuux)
	}

	// TODO: known bug, that maps can not be set when there is no config file. Uncomment and fix
	//s.Equal(true, c.QuxMap["key1"].Quuux)
}

func TestOnlyConfig(t *testing.T) {
	loadEnvVars()
	setupEnvConfig(t)

	var c Config
	confReader := NewConfReader("fixture").WithSearchDirs("testdata")
	if assert.NoError(t, confReader.Read(&c)) {
		assert.Equal(t, "foo", c.Foo)
		assert.Equal(t, 1, c.Bar.BAZ)
		assert.Equal(t, true, c.QUX.Quuux)
		assert.Equal(t, "testptr1", c.FooPtr.Value)
		assert.Equal(t, "testptr2", c.QUX.QuuuxPtr.Value)
	}
}

func TestConfigWithEnvs(t *testing.T) {
	loadEnvVars()
	setupEnvConfig(t)
	defer tearDownEnvConfig(t)

	var c Config
	conf := NewConfReader("fixture")
	conf.WithPrefix("PREF")
	err := conf.Read(&c)
	if assert.NoError(t, err) {
		assert.Equal(t, "fooooo", c.Foo)
		assert.Equal(t, 2, c.Bar.BAZ)
		assert.Equal(t, false, c.QUX.Quuux)
		assert.Equal(t, "testptr1", c.FooPtr.Value)
		assert.Equal(t, "testptr2", c.QUX.QuuuxPtr.Value)
	}
}

func setupEnvConfig(t *testing.T) {
	for k, v := range env {
		if err := os.Setenv(k, v); err != nil {
			t.Error(err)
		}
	}
}

func tearDownEnvConfig(t *testing.T) {
	for k := range env {
		if err := os.Unsetenv(k); err != nil {
			t.Error(err)
		}
	}
}

func loadEnvVars() {
	p := path.Join("testdata", "fixture_env")
	bytes, err := ioutil.ReadFile(p)
	if err != nil {
		panic("Can't load env vars. Err: " + err.Error())
	}
	content := string(bytes)
	raws := strings.Split(content, "\n")
	env = make(map[string]string, len(raws))
	for _, raw := range raws {
		if len(raw) == 0 {
			continue
		}
		pair := strings.Split(raw, "=")
		if len(pair) != 2 {
			panic("invalid env fixtures")
		}
		k := pair[0]
		v := pair[1]
		env[k] = v
	}
}

type Config struct {
	Foo    string
	FooPtr *PtrTest
	Bar    struct {
		BAZ int
	}
	QUX
	TagTest string `custom_tag:"TAG_TEST"`
}

type QUX struct {
	Quuux    bool
	QuuuxPtr *PtrTest
}

type PtrTest struct {
	Value string
}

func TestNew(t *testing.T) {
	v := viper.New()
	assert.Exactly(t, &enviper.Enviper{Viper: v}, enviper.New(v))
}

func ExampleEnviper_Unmarshal() {
	// describe config structure

	type barry struct {
		Bar int `envvar:"bar"`
	}
	type config struct {
		Foo   string
		Barry barry
	}

	// write config file

	dir := os.TempDir()
	defer os.RemoveAll(dir)
	p := path.Join(dir, "config.yaml")
	ioutil.WriteFile(p, []byte(`
Foo: foo
Barry:
  bar: 1
`), 0777)

	// write env vars that could override values from config file

	os.Setenv("MYAPP_BARRY_BAR", "2") // override value from file
	defer os.Unsetenv("MYAPP_BARRY_BAR")

	// setup viper and enviper

	var c config
	e := enviper.New(viper.New())
	e.SetEnvPrefix("MYAPP")
	e.AddConfigPath(dir)
	e.SetConfigName("config")
	if err := e.Unmarshal(&c); err != nil {
		fmt.Printf("%+v\n", err)
	}

	fmt.Println(c.Foo)       // file only
	fmt.Println(c.Barry.Bar) // file & env, take env
	// Output:
	// foo
	// 2
}
