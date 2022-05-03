package config

import (
	"flag"
	"github.com/spf13/pflag"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const bindedFlag = "id"

type FullConfig struct {
	GlobalConfig `mapstructure:",squash"`
	App          LocalConfig
}

type LocalConfig struct {
	Id                 string `flag:"id"`
	FromEnvVar         string
	FromConfig         string
	OverriddenByEvnVar string
	OverriddenByArg    string
}

func Test_ConfigReader(t *testing.T) {
	// arrange
	valFromVar := "valFromVar"
	overridenVar := "overridenVar"
	fromArgVal := "fromArgValue"

	nc := &FullConfig{}
	confReader := NewConfReader("myapp").WithLog(os.Stdout)
	confReader.ConfigDirs = []string{"testdata"}

	os.Args = []string{"get", "--" + bindedFlag, "10", "--verbose", "--app.overriddenbyarg", fromArgVal}

	os.Setenv("MYAPP_APP_FROMENVVAR", valFromVar)
	defer os.Unsetenv("MYAPP_APP_FROMENVVAR")

	os.Setenv("MYAPP_APP_OVERRIDDENBYEVNVAR", overridenVar)
	defer os.Unsetenv("MYAPP_APP_OVERRIDDENBYEVNVAR")

	flag.Set("fromArg", "ValFromArg")

	// act
	err := confReader.Read(nc)

	// assert
	if assert.NoError(t, err) {
		assert.Equal(t, "10", nc.App.Id)
		assert.Equal(t, true, nc.GlobalConfig.Verbose)
		assert.Equal(t, "valFromConf", nc.App.FromConfig)
		assert.Equal(t, valFromVar, nc.App.FromEnvVar)
		assert.Equal(t, fromArgVal, nc.App.OverriddenByArg)
	}
}

func Test_ReadFromFile(t *testing.T) {
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	nc := &FullConfig{}
	confReader := NewConfReader("myapp").WithSearchDirs("testdata")

	err := confReader.Read(nc)
	if assert.NoError(t, err) {
		assert.Equal(t, "valFromConf", nc.App.FromConfig)
		assert.Equal(t, true, nc.Verbose)
	}
}

func Test_EnvVarsNoPrefix(t *testing.T) {
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	nc := &FullConfig{}
	confReader := NewConfReader("myapp").WithoutPrefix()
	os.Setenv("APP_FROMENVVAR", "valFromEnvVar")

	err := confReader.Read(nc)
	if assert.NoError(t, err) {
		assert.Equal(t, "valFromEnvVar", nc.App.FromEnvVar)
	}
}

func Test_ReadFromJsonFile(t *testing.T) {
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	nc := &FullConfig{}
	confReader := NewConfReader("myappjson")
	confReader.ConfigDirs = []string{"testdata"}
	err := confReader.Read(nc)
	if assert.NoError(t, err) {
		assert.Equal(t, "valFromConf", nc.App.FromConfig)
		assert.Equal(t, true, nc.Verbose)
	}
}

type dmParent struct {
	GlobalConfig `mapstructure:",squash"`
	Conf         dmSibling `flag:"notAllowed"`
	PtrConf      *dmSibling
	Par          float64 `flag:"par"`
	Duration     time.Duration
}

type dmSibling struct {
	Id         string `flag:"id"`
	FromEnvVar string
}

func Test_DumpStruct(t *testing.T) {
	m := map[string]*flagInfo{}
	c := &ConfReader{}
	c.dumpStruct(reflect.TypeOf(dmParent{}), "", m)

	if assert.NotNil(t, m["verbose"]) {
		assert.Equal(t, "verbose", m["verbose"].Name)
		assert.Equal(t, "bool", m["verbose"].Type.String())
	}

	if assert.NotNil(t, m["conf.id"]) {
		assert.Equal(t, "id", m["conf.id"].Name)
		assert.Equal(t, "string", m["conf.id"].Type.String())
	}

	if assert.NotNil(t, m["ptrconf.id"]) {
		assert.Equal(t, "id", m["ptrconf.id"].Name)
		assert.Equal(t, "string", m["ptrconf.id"].Type.String())
	}

	if assert.NotNil(t, m["par"]) {
		assert.Equal(t, "par", m["par"].Name)
		assert.Equal(t, "float64", m["par"].Type.String())
	}

	if assert.NotNil(t, m["duration"]) {
		assert.Equal(t, "duration", m["duration"].Name)
		assert.Equal(t, "time.Duration", m["duration"].Type.String())
	}

}

type ValidationConfig struct {
	Host string `validate:"required"`
}

func Test_Validation(t *testing.T) {
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	cf := &ValidationConfig{}
	reader := NewConfReader("myapp")

	t.Run("fails", func(t *testing.T) {
		err := reader.Read(cf)
		if assert.Error(t, err) {
			assert.Equal(t, "validation error: Key: 'ValidationConfig.Host' Error:Field validation for 'Host' failed on the 'required' tag", err.Error())
		}
	})

	t.Run("passes", func(t *testing.T) {
		os.Setenv("MYAPP_HOST", "localhost")
		defer os.Unsetenv("MYAPP_HOST")
		err := reader.Read(cf)
		if assert.NoError(t, err) {
			assert.Equal(t, "localhost", cf.Host)
		}
	})

}
