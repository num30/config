package config

import (
	"encoding/base64"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/num30/config/lib"
	"github.com/spf13/pflag"

	"github.com/stretchr/testify/assert"
)

type GlobalConfig struct {
	Verbose bool
}

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
	EnvVarName         string `envvar:"CUSTOM_ENV_VAR"`
}

func Test_ConfigReader(t *testing.T) {
	// arrange
	resetFlags()
	valFromVar := "valFromVar"
	overriddenVar := "overriddenVar"
	fromArgVal := "fromArgValue"

	nc := &FullConfig{}
	confReader := NewConfReader("myapp").WithSearchDirs("testdata")

	os.Args = []string{"get", "--id", "10", "--verbose", "--app.overriddenbyarg", fromArgVal}

	os.Setenv("APP_FROMENVVAR", valFromVar)
	defer os.Unsetenv("APP_FROMENVVAR")

	os.Setenv("APP_OVERRIDDENBYEVNVAR", overriddenVar)
	defer os.Unsetenv("APP_OVERRIDDENBYEVNVAR")

	os.Setenv("CUSTOM_ENV_VAR", "valFromEnvVar")
	defer os.Unsetenv("CUSTOM_ENV_VAR")

	// act
	err := confReader.Read(nc)

	// assert
	if assert.NoError(t, err) {
		assert.Equal(t, "10", nc.App.Id)
		assert.Equal(t, true, nc.GlobalConfig.Verbose)
		assert.Equal(t, "valFromConf", nc.App.FromConfig)
		assert.Equal(t, valFromVar, nc.App.FromEnvVar)
		assert.Equal(t, fromArgVal, nc.App.OverriddenByArg)
		assert.Equal(t, "valFromEnvVar", nc.App.EnvVarName)
	}
}

// flags are static so we have to reset them before each test
func resetFlags() {
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	os.Args = []string{"app"}
}

func Test_ReadFromFile(t *testing.T) {
	resetFlags()
	nc := &FullConfig{}
	confReader := NewConfReader("myapp").WithSearchDirs("testdata")

	err := confReader.Read(nc)
	if assert.NoError(t, err) {
		assert.Equal(t, "valFromConf", nc.App.FromConfig)
		assert.Equal(t, true, nc.Verbose)
	}
}

func Test_EnvVarsNoPrefix(t *testing.T) {
	resetFlags()
	nc := &FullConfig{}
	confReader := NewConfReader("myapp")
	os.Setenv("APP_FROMENVVAR", "valFromEnvVar")

	err := confReader.Read(nc)
	if assert.NoError(t, err) {
		assert.Equal(t, "valFromEnvVar", nc.App.FromEnvVar)
	}
}

func Test_EnvVarsWithPrefix(t *testing.T) {
	resetFlags()
	nc := &FullConfig{}
	confReader := NewConfReader("myapp").WithPrefix("MYAPP")
	os.Setenv("MYAPP_APP_FROMENVVAR", "valFromEnvVar")
	defer os.Unsetenv("APP_FROMENVVAR")

	err := confReader.Read(nc)
	if assert.NoError(t, err) {
		assert.Equal(t, "valFromEnvVar", nc.App.FromEnvVar)
	}
}

func Test_ReadFromJsonFile(t *testing.T) {
	resetFlags()
	nc := &FullConfig{}
	confReader := NewConfReader("myappjson")
	confReader.configDirs = []string{"testdata"}
	err := confReader.Read(nc)
	if assert.NoError(t, err) {
		assert.Equal(t, "valFromConf", nc.App.FromConfig)
		assert.Equal(t, true, nc.Verbose)
	}
}

func Test_WatchWithFile(t *testing.T) {
	resetFlags()
	nc := &FullConfig{}

	if err := os.Mkdir("testdata/tmp", 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("testdata/tmp")

	if err := os.WriteFile("testdata/tmp/changing_file.json", []byte(`{"verbose":"true"}`), 0644); err != nil {
		t.Fatal(err)
	}

	confReader := NewConfReader("changing_file")
	confReader.configDirs = []string{"testdata/tmp"}

	t.Run("callWatchBeforeRead", func(t *testing.T) {
		assert.Panics(t, func() {
			confReader.Watch()
		})
	})

	err := confReader.Read(nc)
	if assert.NoError(t, err) {
		assert.Equal(t, true, nc.Verbose)
	}
	mutex := confReader.Watch()

	t.Run("configChanges", func(t *testing.T) {
		err = os.WriteFile("testdata/tmp/changing_file.json", []byte(`{"verbose":"false"}`), 0644)
		if assert.NoError(t, err) {
			time.Sleep(10 * time.Millisecond)
			assert.Equal(t, false, nc.Verbose)
		}
	})

	t.Run("configStructLock", func(t *testing.T) {
		mutex.RLock()
		err = os.WriteFile("testdata/tmp/changing_file.json", []byte(`{"verbose":"true"}`), 0644)
		if err != nil {
			t.Fatal(err)
		}

		//time.Sleep(time.Second)
		assert.Equal(t, false, nc.Verbose)

		mutex.RUnlock()
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, true, nc.Verbose)
	})

	t.Run("doesNotFilaIfFileDeleted", func(t *testing.T) {
		err = os.Remove("testdata/tmp/changing_file.json")
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, true, nc.Verbose)
	})

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

func Test_dumpStruct(t *testing.T) {
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
	cf := &ValidationConfig{}
	reader := NewConfReader("myapp")

	t.Run("fails", func(t *testing.T) {
		resetFlags()
		err := reader.Read(cf)
		if assert.Error(t, err) {
			assert.Equal(t, "validation error: Key: 'ValidationConfig.Host' Error:Field validation for 'Host' failed on the 'required' tag", err.Error())
		}
	})

	t.Run("passes", func(t *testing.T) {
		resetFlags()
		os.Setenv("HOST", "localhost")
		defer os.Unsetenv("HOST")
		err := reader.Read(cf)
		if assert.NoError(t, err) {
			assert.Equal(t, "localhost", cf.Host)
		}
	})
}

type DefaultVals struct {
	DB   lib.PostgresqlDb
	Test string `default:"test"`
}

func Test_DefaultFalse(t *testing.T) {
	resetFlags()
	cf := &DefaultVals{}
	reader := NewConfReader("def-vals")
	err := reader.Read(cf)
	if assert.NoError(t, err) {
		assert.Equal(t, "localhost", cf.DB.Host)
		assert.Equal(t, "test", cf.Test)
	}
}

type AllTypes struct {
	Bool     bool
	Int      int
	Int8     int8
	Int16    int16
	Int32    int32
	Int64    int64
	Uint     uint
	Uint8    uint8
	Uint16   uint16
	Uint32   uint32
	Uint64   uint64
	Float32  float32
	Float64  float64
	String   string
	Bytes    []byte
	Duration time.Duration
	Slice    []string
}

func Test_AllTypesFlags(t *testing.T) {
	resetFlags()
	cf := &AllTypes{}
	reader := NewConfReader("all-types")

	// encode to base64
	encoded := base64.StdEncoding.EncodeToString([]byte("test"))

	os.Args = []string{"myapp", "--bool", "true", "--int", "1", "--int8", "1", "--int16", "1", "--int32", "1",
		"--int64", "1", "--uint", "1", "--uint8", "1", "--uint16", "1", "--uint32", "1", "--uint64", "1",
		"--float32", "1.1", "--float64", "1.1", "--string", "test", "--bytes", encoded, "--duration", "1m",
		"--slice", "a", "--slice", "b"}
	err := reader.Read(cf)

	if assert.NoError(t, err) {
		assert.Equal(t, true, cf.Bool)
		assert.Equal(t, 1, cf.Int)
		assert.Equal(t, int8(1), cf.Int8)
		assert.Equal(t, int16(1), cf.Int16)
		assert.Equal(t, int32(1), cf.Int32)
		assert.Equal(t, int64(1), cf.Int64)
		assert.Equal(t, uint(1), cf.Uint)
		assert.Equal(t, uint8(1), cf.Uint8)
		assert.Equal(t, uint16(1), cf.Uint16)
		assert.Equal(t, uint32(1), cf.Uint32)
		assert.Equal(t, uint64(1), cf.Uint64)
		assert.Equal(t, float32(1.1), cf.Float32)
		assert.Equal(t, float64(1.1), cf.Float64)
		assert.Equal(t, "test", cf.String)
		assert.Equal(t, []byte("test"), cf.Bytes)
		assert.Equal(t, 1*time.Minute, cf.Duration)
		assert.Equal(t, []string{"a", "b"}, cf.Slice)
	}
}

type SliceConf struct {
	Slice []string
}

func Test_Slice(t *testing.T) {
	resetFlags()

	t.Run("envVar", func(t *testing.T) {
		cfg := SliceConf{}
		os.Setenv("SLICE", "a,b,c")
		defer os.Unsetenv("SLICE")
		reader := NewConfReader("slice")
		err := reader.Read(&cfg)
		if assert.NoError(t, err) {
			assert.Equal(t, []string{"a", "b", "c"}, cfg.Slice)
		}
	})

	t.Run("file", func(t *testing.T) {
		cfg := SliceConf{}
		reader := NewConfReader("slice").WithSearchDirs("testdata")
		err := reader.Read(&cfg)
		if assert.NoError(t, err) {
			assert.Equal(t, []string{"a", "b", "c"}, cfg.Slice)
		}
	})

	t.Run("cmdArgs", func(t *testing.T) {
		cfg := SliceConf{}
		os.Args = []string{"myapp", "--slice", "a", "--slice", "b"}
		reader := NewConfReader("slice-args")
		err := reader.Read(&cfg)
		if assert.NoError(t, err) {
			assert.Equal(t, []string{"a", "b"}, cfg.Slice)
		}
	})
}

func Test_ByteArray(t *testing.T) {
	resetFlags()
	cfg := struct {
		Bytes []byte
	}{}

	t.Run("envVar", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("test"))
		os.Args = []string{"myapp", "--bytes", encoded}

		reader := NewConfReader("slice")
		err := reader.Read(&cfg)
		if assert.NoError(t, err) {
			assert.Equal(t, "test", string(cfg.Bytes))
		}
	})
}

func Test_InputStructErrors(t *testing.T) {
	resetFlags()
	confReader := NewConfReader("myapp").WithSearchDirs("testdata")

	t.Run("notAPointer", func(t *testing.T) {
		var nc FullConfig
		err := confReader.Read(nc)
		if assert.Error(t, err, "failed to read") {
			assert.Contains(t, err.Error(), "config struct must be pointer")
		}
	})

	t.Run("nil", func(t *testing.T) {
		err := confReader.Read(nil)
		if assert.Error(t, err, "failed to read") {
			assert.Contains(t, err.Error(), "config struct is nil")
		}
	})

	t.Run("zero", func(t *testing.T) {
		var nc *FullConfig
		err := confReader.Read(nc)
		if assert.Error(t, err, "failed to read") {
			assert.Contains(t, err.Error(), "config struct is nil")
		}
	})
}
