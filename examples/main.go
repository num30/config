package main

import (
	"fmt"
	"github.com/num30/config"
	"os"
)

type MyConfig struct {
	Debug bool
	Foo   FooConfig
}

type FooConfig struct {
	Name      string
	SomeValue int
}

func main() {
	// Arrange
	os.Setenv("DEBUG", "true")
	defer os.Unsetenv("DEBUG")

	// Run
	cfgReader := config.NewConfMgr("myconf")
	conf := MyConfig{}
	cfgReader.ReadConfig(conf)

	// Output:
	fmt.Printf("%+v", conf)
}
