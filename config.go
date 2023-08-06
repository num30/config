package config

import (
	"encoding/base64"
	"log"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/creasty/defaults"
	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/validator/v10"
	"github.com/iamolegga/enviper"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

/*
	ConfReader reads configuration from config file, environment variables or command line flags.
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
	viper        *enviper.Enviper
	configName   string
	configDirs   []string
	envVarPrefix string
	Verbose      bool
	configStruct any
}

// NewConfReader creates new instance of ConfReader
// configName is a name of config file name without extension and evn vars prefix
func NewConfReader(configName string) *ConfReader {
	return &ConfReader{
		viper:        enviper.New(viper.New()),
		configName:   configName,
		envVarPrefix: "",
	}
}

// Read reads config from config file, env vars or flags.
func (c *ConfReader) Read(configStruct interface{}) error {
	// validate the input struct
	rval := reflect.ValueOf(configStruct)
	if configStruct == nil || rval == reflect.Zero(rval.Type()) {
		return errors.New("config struct is nil")
	}

	if rval.Kind() != reflect.Ptr {
		return errors.New("config struct must be pointer")
	}

	// set default values
	if err := defaults.Set(configStruct); err != nil {
		return errors.Wrap(err, "failed to set default values")
	}

	jww.SetLogThreshold(jww.LevelTrace)
	jww.SetStdoutThreshold(jww.LevelTrace)

	c.viper.SetConfigFile(c.configName)

	if c.configDirs == nil || len(c.configDirs) == 0 {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		// Search config in home directory (without extension).
		c.viper.AddConfigPath(home)
		c.viper.AddConfigPath("./")
	} else {
		for _, path := range c.configDirs {
			c.viper.AddConfigPath(path)
		}
	}
	c.viper.SetConfigName(c.configName)
	c.viper.SetEnvPrefix(c.envVarPrefix)

	// Bind flags
	if err := c.flagsBinding(configStruct); err != nil {
		return err
	}

	err := c.viper.Unmarshal(configStruct)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal struct")
	}

	// validate struct
	err = validator.New().Struct(configStruct)
	if err != nil {
		validationErrors := err.(validator.ValidationErrors)
		if len(validationErrors) > 0 {
			if err != nil {
				return errors.Wrap(err, "validation error")
			}
		}
		return err
	}

	c.configStruct = configStruct
	return nil
}

func (c *ConfReader) flagsBinding(conf interface{}) error {
	t := reflect.TypeOf(conf)
	tagsInfo := map[string]*flagInfo{}
	tagsInfo = c.dumpStruct(t, "", tagsInfo)

	var flags = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

	for _, v := range tagsInfo {
		switch v.Type.Kind() {
		case reflect.String:
			flags.String(v.Name, v.DefaultVal, "")

		case reflect.Bool:
			flags.Bool(v.Name, false, "")

		case reflect.Float32:
			flags.Float32(v.Name, 0, "")

		case reflect.Float64:
			flags.Float64(v.Name, 0, "")

		case reflect.Int:
			flags.Int(v.Name, 0, "")

		case reflect.Int16:
			flags.Int16(v.Name, 0, "")

		case reflect.Int32:
			flags.Int(v.Name, 0, "")

		case reflect.Int64:
			if v.Type.String() == "time.Duration" {
				flags.Duration(v.Name, 0, "")
			} else {
				flags.Int64(v.Name, 0, "")
			}

		case reflect.Int8:
			flags.Int8(v.Name, 0, "")

		case reflect.Uint:
			flags.Uint(v.Name, 0, "")
		case reflect.Uint32:
			flags.Uint(v.Name, 0, "")

		case reflect.Uint64:
			flags.Uint64P(v.Name, "", 0, "")

		case reflect.Uint8:
			flags.Uint8(v.Name, 0, "")

		case reflect.Uint16:
			flags.Uint16(v.Name, 0, "")

		case reflect.Slice:
			switch v.Type.String() {
			case "[]string":
				flags.StringSlice(v.Name, []string{}, "")
			case "[]uint8":
				flags.BytesBase64(v.Name, []byte{}, "byte array in base64")
			}
		}

		if v.EnvVar != "" {
			err := c.viper.BindEnv(v.Name, v.EnvVar)
			if err != nil {
				return err
			}
		}
	}

	err := flags.Parse(os.Args[1:])
	if err != nil {
		return errors.Wrap(err, "failed to parse flags")
	}
	for k, v := range tagsInfo {
		f := flags.Lookup(v.Name)
		if f != nil && f.Changed {
			if v.Type.Kind() == reflect.Slice {
				// byte array should be in base64
				if v.Type.String() == "[]uint8" {
					b, err := base64.StdEncoding.DecodeString(f.Value.String())
					if err != nil {
						return errors.Wrap(err, "failed to decode base64 value for flag: "+v.Name)
					}
					c.viper.Set(k, b)
				} else {
					c.viper.Set(k, f.Value.(pflag.SliceValue).GetSlice())
				}

			} else {
				c.viper.Set(k, f.Value)
			}
		}

	}

	return nil
}

type flagInfo struct {
	Name       string
	Type       reflect.Type
	DefaultVal string
	EnvVar     string
}

func (c *ConfReader) dumpStruct(t reflect.Type, path string, res map[string]*flagInfo) map[string]*flagInfo {
	if c.Verbose {
		log.Printf("%s: %s", path, t.Name())
	}
	switch t.Kind() {
	case reflect.Ptr:
		originalValue := t.Elem()
		res = c.dumpStruct(originalValue, path, res)

	// If it is a struct we translate each field
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)

			if f.Type.Kind() != reflect.Struct && f.Type.Kind() != reflect.Ptr && f.Type.Kind() != reflect.Chan &&
				f.Type.Kind() != reflect.Func && f.Type.Kind() != reflect.Interface && f.Type.Kind() != reflect.UnsafePointer {

				// do we have flag name override ?
				flagVal := f.Tag.Get("flag")
				envVar := f.Tag.Get("envvar")

				fieldPath := strings.TrimPrefix(strings.ToLower(path+"."+f.Name), ".")
				if flagVal != "" {
					res[fieldPath] = &flagInfo{
						Name:       flagVal,
						Type:       f.Type,
						DefaultVal: f.Tag.Get("default"),
						EnvVar:     envVar,
					}
				} else {
					res[fieldPath] = &flagInfo{
						Name:       fieldPath,
						Type:       f.Type,
						DefaultVal: f.Tag.Get("default"),
						EnvVar:     envVar,
					}
				}

			} else if f.Type.Kind() == reflect.Struct || f.Type.Kind() == reflect.Ptr {
				val := f.Tag.Get("mapstructure")
				if strings.Contains(val, "squash") {
					res = c.dumpStruct(f.Type, path, res)
				} else {
					res = c.dumpStruct(f.Type, path+"."+f.Name, res)
				}
			}
		}

	case reflect.Interface:
		// Skipping interface
	}
	return res
}

func (c *ConfReader) WithSearchDirs(s ...string) *ConfReader {
	c.configDirs = s
	return c
}

// WithPrefix sets the prefix for environment variables. It adds '_' to the end of the prefix.
// For example, if prefix is "MYAPP", then environment variable for field "Name" will be "MYAPP_NAME".
func (c *ConfReader) WithPrefix(prefix string) *ConfReader {
	c.envVarPrefix = prefix
	return c
}

// Watch watches for config changes and reloads config. This method should be called after Read() to make sure that ConfReader konws which struct to reload.
// Returns a mutex that can be used to synchronize access to the config.
// If you care about thread safety, call RLock() on the mutex while accessing the config and the RUnlock().
// This will ensure that the config is not reloaded while you are accessing it.
func (c *ConfReader) Watch() *sync.RWMutex {
	if c.configStruct == nil {
		panic("ConfReader: config struct is not set. Call Read before Watch")
	}
	rwmutex := &sync.RWMutex{}

	c.viper.WatchConfig()
	c.viper.OnConfigChange(func(e fsnotify.Event) {
		rwmutex.Lock()
		defer rwmutex.Unlock()
		err := c.Read(c.configStruct)
		if err != nil {
			log.Printf("failed to reload config: %s\n", err)
		}

	})
	return rwmutex
}
