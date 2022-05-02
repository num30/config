package config

import (
	"flag"
	"os"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

const bindedFlag = "id"

type fullConfig struct {
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

	nc := &fullConfig{}
	confReader := NewConfMgr("myapp.yaml")
	confReader.ConfigPaths = []string{"testdata"}

	os.Args = []string{"get", "--" + bindedFlag, "10", "--verbose", "true", "--fromArg", fromArgVal}

	os.Setenv("MYAPP_CONF_FROMENVVAR", valFromVar)
	defer os.Unsetenv("MYAPP_CONF_FROMENVVAR")

	os.Setenv("MYAPP_CONF_OVERRIDDENBYEVNVAR", overridenVar)
	defer os.Unsetenv("MYAPP_CONF_OVERRIDDENBYEVNVAR")

	flag.Set("fromArg", "ValFromArg")

	// act
	err := confReader.ReadConfig(nc)

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
	nc := &fullConfig{}
	confReader := NewConfMgr("myapp.yaml")
	confReader.ConfigPaths = []string{"testdata"}
	err := confReader.ReadConfig(nc)
	if assert.NoError(t, err) {
		assert.Equal(t, "valFromConf", nc.App.FromConfig)
		assert.Equal(t, true, nc.Verbose)
	}
}

func Test_ReadFromJsonFile(t *testing.T) {
	nc := &fullConfig{}
	confReader := NewConfMgr("myappjson")
	confReader.ConfigPaths = []string{"testdata"}
	err := confReader.ReadConfig(nc)
	if assert.NoError(t, err) {
		assert.Equal(t, "valFromConf", nc.App.FromConfig)
		assert.Equal(t, true, nc.Verbose)
	}
}

func newTestRootCommand(confReader ConfigReader, actualConf *fullConfig) *cobra.Command {
	nodeCmd := &cobra.Command{
		Use:   "node",
		Short: "Access node",
	}
	nodeCmd.PersistentFlags().Bool("verbose", false, "verbose")

	nodeCmd.AddCommand(subCommand(confReader, actualConf))
	return nodeCmd
}

// getCmd represents the get command
func subCommand(confReader ConfigReader, actualConf *fullConfig) *cobra.Command {
	getCmd := cobra.Command{
		Use: "get",
		Run: func(cmd *cobra.Command, args []string) {
			confReader.ReadConfig(actualConf)
		},
	}

	getCmd.Flags().StringP(bindedFlag, "i", "", "id")
	//confReader.BindFlag("conf.id", getCmd.Flags().Lookup(bindedFlag))

	getCmd.Flags().StringP("node.cert", "c", "", "")
	getCmd.Flags().String("conf.overriddenByArg", "", "")

	return &getCmd
}

type dmParent struct {
	GlobalConfig `mapstructure:",squash"`
	Conf         dmSibling `flag:"notAllowed"`
	PtrConf      *dmSibling
	Par          float64 `flag:"par"`
}

type dmSibling struct {
	Id         string `flag:"id"`
	FromEnvVar string
}

func TestDumpStrunct(t *testing.T) {
	m := map[string]*flagInfo{}
	c := &ConfMgr{}
	c.dumpStruct(reflect.TypeOf(dmParent{}), "", m)

	assert.Equal(t, "id", m["conf.id"].Name)
	assert.Equal(t, "string", m["conf.id"].Type.String())

	assert.Equal(t, "id", m["ptrconf.id"].Name)
	assert.Equal(t, "string", m["ptrconf.id"].Type.String())

	assert.Equal(t, "par", m["par"].Name)
	assert.Equal(t, "float64", m["par"].Type.String())
}
