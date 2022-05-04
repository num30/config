package config

import (
	"fmt"
	"github.com/creasty/defaults"
	"github.com/go-playground/validator/v10"
	"github.com/iamolegga/enviper"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"io"
	"log"
	"os"
	"reflect"
	"strings"
)

/* ConfReader reads configuration from config file, environment variables or command line flags.
 Flags have precedence over env vars and env vars have precedence over config file.


For example:
	type NestedConf struct {
		Foo string
	}

  	type Config struct{
      Nested NestedConf
 	}

in that case flag --nested.foo will be mapped automatically
This flag could be also set by NESTED_FOO env var or by creating  config file .ukama.yaml:
nested:
  foo: bar
*/
type ConfReader struct {
	viper      *enviper.Enviper
	configName string
	// Set Log if you want to see extra logging from config manager
	Log *log.Logger
	//
	ConfigDirs   []string
	EnvVarPrefix string
}

// NewConfReader creates new instance of ConfReader
// configName is a name of config file name without extension and evn vars prefix
func NewConfReader(configName string) *ConfReader {
	return &ConfReader{
		viper:        enviper.New(viper.New()),
		Log:          nil,
		configName:   configName,
		EnvVarPrefix: strings.ToUpper(configName),
	}
}

// Read reads config from config file, env vars or flags.
func (c *ConfReader) Read(configStruct interface{}) error {
	if configStruct == nil {
		return errors.New("config struct is nil")
	}

	// set default values
	if err := defaults.Set(configStruct); err != nil {
		return errors.Wrap(err, "failed to set default values")
	}

	//jww.SetLogThreshold(jww.LevelTrace)
	//jww.SetStdoutThreshold(jww.LevelTrace)

	c.viper.SetConfigFile(c.configName)

	if c.ConfigDirs == nil || len(c.ConfigDirs) == 0 {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			return err
		}
		// Search config in home directory (without extension).
		c.viper.AddConfigPath(home)
		c.viper.AddConfigPath("./")
	} else {
		for _, path := range c.ConfigDirs {
			c.viper.AddConfigPath(path)
		}
	}
	c.viper.SetConfigName(c.configName)
	c.viper.SetEnvPrefix(c.EnvVarPrefix)

	// Bind flags
	if err := c.flagsBinding(configStruct); err != nil {
		return err
	}

	err := c.viper.Unmarshal(configStruct)
	if err != nil {
		c.log("Unable to decode into struct, %v", err)
		return errors.Wrap(err, "failed to unmarshal struct")
	}

	// validate struct
	err = validator.New().Struct(configStruct)
	if err != nil {
		validationErrors := err.(validator.ValidationErrors)
		if len(validationErrors) > 0 {
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

func (c *ConfReader) flagsBinding(conf interface{}) error {
	t := reflect.TypeOf(conf)
	m := map[string]*flagInfo{}
	c.dumpStruct(t, "", m)

	var flags = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

	for _, v := range m {
		switch v.Type.String() {
		case "string":
			flags.String(v.Name, v.DefaultVal, "")

		case "bool":
			flags.Bool(v.Name, false, "")
		case "float32":
			flags.Float32(v.Name, 0, "")

		case "float64":
			flags.Float64(v.Name, 0, "")

		case "time.Duration":
			flags.Duration(v.Name, 0, "")

		case "int":
		case "int32":
			flags.Int(v.Name, 0, "")

		case "uint":
		case "uint32":
			flags.Uint(v.Name, 0, "")

		case "uint64":
			flags.Uint64P(v.Name, "", 0, "")

		case "uint8":
			flags.Uint8(v.Name, 0, "")

			// TODO: add more types
		}
	}

	err := flags.Parse(os.Args[1:])
	if err != nil {
		return errors.Wrap(err, "failed to parse flags")
	}
	for k, v := range m {
		f := flags.Lookup(v.Name)
		if f != nil && f.Changed {
			c.viper.Set(k, f.Value)
		}

	}

	return nil
}

type flagInfo struct {
	Name       string
	Type       reflect.Type
	DefaultVal string
}

func (c *ConfReader) dumpStruct(t reflect.Type, path string, res map[string]*flagInfo) {
	fmt.Printf("%s: %s", path, t.Name())
	switch t.Kind() {
	case reflect.Ptr:
		originalValue := t.Elem()
		c.dumpStruct(originalValue, path, res)

	// If it is a struct we translate each field
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)

			if f.Type.Kind() != reflect.Struct && f.Type.Kind() != reflect.Ptr && f.Type.Kind() != reflect.Chan &&
				f.Type.Kind() != reflect.Func && f.Type.Kind() != reflect.Interface && f.Type.Kind() != reflect.UnsafePointer {

				// do we have flag name override ?
				val := f.Tag.Get("flag")
				fieldPath := strings.TrimPrefix(strings.ToLower(path+"."+f.Name), ".")
				if val != "" {
					res[fieldPath] = &flagInfo{
						Name:       val,
						Type:       f.Type,
						DefaultVal: f.Tag.Get("default"),
					}
				} else {
					res[fieldPath] = &flagInfo{
						Name:       fieldPath,
						Type:       f.Type,
						DefaultVal: f.Tag.Get("default"),
					}
				}

			} else if f.Type.Kind() == reflect.Struct || f.Type.Kind() == reflect.Ptr {
				val := f.Tag.Get("mapstructure")
				if strings.Contains(val, "squash") {
					c.dumpStruct(f.Type, path, res)
				} else {
					c.dumpStruct(f.Type, path+"."+f.Name, res)
				}
			}
		}

	case reflect.Interface:
		c.Log.Printf("Skipping interface")
	}

}

func (c *ConfReader) BindFlag(confKey string, flag *pflag.Flag) error {
	err := c.viper.BindPFlag(confKey, flag)

	if err != nil {
		errors.Wrap(err, "error binding flag")
	}
	return nil
}

func (c *ConfReader) log(s string, args ...interface{}) {
	if c.Log != nil {
		c.Log.Printf(s, args...)
	}
}

func (c *ConfReader) WithLog(writer io.Writer) *ConfReader {
	c.Log = log.New(writer, "ConfigReader", log.LstdFlags)
	return c
}

func (c *ConfReader) WithSearchDirs(s ...string) *ConfReader {
	c.ConfigDirs = s
	return c
}

func (c *ConfReader) WithoutPrefix() *ConfReader {
	c.EnvVarPrefix = ""
	return c
}
