package config

import (
	"log"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/iamolegga/enviper"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type GlobalConfig struct {
	Verbose bool
}

/* CodefigReader allows commands to read configuration from config file, env vars or flags.
 Flags have precedence over env vars and evn vars have precedence over config file.
 Flags mapped to config struct filed automatically if their name includes path to field.
 For example:
	type NestedConf struct {
		Foo string
	}

  	type Config struct{
      Nested NestedConf
 	}

in that case flag --nested.foo will be mapped automatically
This flag could be also set by UKAMA_NESTED_FOO env var or by creating  config file .ukama.yaml:
nested:
  foo: bar
*/
type ConfigReader interface {
	// ReadConfig reads config from config file, env vars or flags. In case of error fails with os.Exit(1)
	ReadConfig(flags *pflag.FlagSet, rawVal interface{}) error
	BindFlag(confKey string, flag *pflag.Flag) error
}

type ConfMgr struct {
	viper      *enviper.Enviper
	configName string
	// Set Log if you want to see extra logging from config manager
	Log *log.Logger
	//
	ConfigPaths  []string
	EnvVarPrefix string
}

// NewConfMgr creates new instance of ConfMgr
// configName is name of config file name without extension
func NewConfMgr(configName string) *ConfMgr {
	return &ConfMgr{
		viper:        enviper.New(viper.New()),
		Log:          nil,
		configName:   configName,
		EnvVarPrefix: strings.ToUpper(configName),
	}
}

func (c *ConfMgr) ReadConfig(flags *pflag.FlagSet, configStruct interface{}) error {

	jww.SetLogThreshold(jww.LevelTrace)
	jww.SetStdoutThreshold(jww.LevelTrace)

	if flags != nil {
		c.lateFlagBinding(flags, configStruct)
	}

	c.viper.SetConfigFile(c.configName)

	if c.ConfigPaths == nil || len(c.ConfigPaths) == 0 {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			return err
		}

		// Search config in home directory (without extension).
		c.viper.AddConfigPath(home)
	} else {
		for _, path := range c.ConfigPaths {
			c.viper.AddConfigPath(path)
		}
	}
	c.viper.SetConfigName(c.configName)

	c.viper.SetEnvPrefix(c.EnvVarPrefix)
	c.viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := c.viper.ReadInConfig(); err == nil {
		c.log("Using config file: %v", c.viper.ConfigFileUsed())
	}

	if flags != nil {
		err := c.viper.BindPFlags(flags)
		if err != nil {
			c.log("Error binding flags: %v", err)
			return errors.Wrap(err, "error binding flag")
		}
	}

	err := c.viper.Unmarshal(configStruct)
	if err != nil {
		c.log("Unable to decode into struct, %v", err)
		return errors.Wrap(err, "failed to unmarshal struct")
	}

	err = validator.New().Struct(configStruct)
	if err != nil {
		validationErrors := err.(validator.ValidationErrors)
		if validationErrors != nil && len(validationErrors) > 0 {
			c.log("Config validation errors: '%+v'", validationErrors)
			if err != nil {
				return errors.Wrap(err, "validation error")
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *ConfMgr) lateFlagBinding(flags *pflag.FlagSet, conf interface{}) {
	t := reflect.TypeOf(conf)
	m := map[string]string{}
	c.dumpStruct(t, "", m)

	for k, v := range m {
		c.BindFlag(k, flags.Lookup(v))
	}
}

func (c *ConfMgr) dumpStruct(t reflect.Type, path string, res map[string]string) {
	switch t.Kind() {
	case reflect.Ptr:
		originalValue := t.Elem()
		c.dumpStruct(originalValue, path, res)

	// If it is a struct we translate each field
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			val := f.Tag.Get("flag")
			if val != "" && f.Type.Kind() != reflect.Struct && f.Type.Kind() != reflect.Ptr {
				res[strings.TrimPrefix(strings.ToLower(path+"."+f.Name), ".")] = val
			}

			c.dumpStruct(f.Type, path+"."+f.Name, res)
		}

	case reflect.Interface:
		c.Log.Printf("Skipping interface")
	}

}

func (c *ConfMgr) BindFlag(confKey string, flag *pflag.Flag) error {
	err := c.viper.BindPFlag(confKey, flag)
	if err != nil {
		errors.Wrap(err, "error binding flag")
	}
	return nil
}

func (c *ConfMgr) log(s string, args ...interface{}) {
	if c.Log != nil {
		c.Log.Printf(s, args...)
	}
}
