package main

import (
	"fmt"
	"github.com/num30/config"
	"os"
	"time"
)

type MyConfig struct {
	// we need `mapstructure:",squash"` here to refer to the inside properties of the struct without refering to the struct itself
	// In our case we want to use `verbose` flag instead of  `globalConfig.verbose`
	GlobalConfig `mapstructure:",squash"`
	Debug        bool
	Foo          FooConfig
}

type FooConfig struct {
	Name               string
	ValueFromFile      int
	DurationFromEnvVar time.Duration
	NestedFlag         string `flag:"nested"`
}

type GlobalConfig struct {
	Verbose bool
}

func main() {
	// Arrange

	// Use env vars to set config keys
	os.Setenv("MYCONF_FOO_DURATIONFROMENVVAR", "10m")

	// Use command args to set config keys
	os.Args = append(os.Args, "--nested", "ThisCameFromAnArg")

	// bool values does not need "value itself
	os.Args = append(os.Args, "--debug", "")

	// Run
	cfgReader := config.NewConfReader("myconf").WithLog(os.Stdout)

	conf := MyConfig{}
	err := cfgReader.Read(&conf)
	if err != nil {
		panic(err)
	}

	// Output:
	fmt.Printf("%+v", conf)
}
