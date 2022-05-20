# Declarative configuration for Go  :rocket:
[![test-and-lint](https://github.com/num30/config/actions/workflows/test-and-lint.yaml/badge.svg)](https://github.com/num30/config/actions/workflows/test-and-lint.yaml)
[![codecov](https://codecov.io/gh/num30/config/branch/main/graph/badge.svg?token=YBOM7T2YUK)](https://codecov.io/gh/num30/config)
[![Go Report Card](https://goreportcard.com/badge/github.com/num30/config)](https://goreportcard.com/report/github.com/num30/config)
[![Go Reference](https://pkg.go.dev/badge/github.com/num30/config.svg)](https://pkg.go.dev/github.com/num30/config)

## Features
- declarative way of defining configuration
- reading configuration from file, environment variables or command line arguments in one lines of code
- validation 

## Example 
`config` is a package that supports reading configuration into a struct from files, environment variable and command line arguments.
All you need is to declare a config structure and call `Read` method.

``` go
type Config struct {	
	DB                Database	
	Debug             Debug
}

type Database struct {
	Host       string
	Password   string
	DbName     string
	Username   string
	Port       int
}

func main() {
    var config Config 
    err := config.NewConfReader("myconf").Read(&config) 
    if err != nil {
        panic(err)
    }
}
```
When you want to change, for example, DB Host of your applications you can do any of the following:
1. create config `myconf.yaml` file in home directory 
``` 
db:
   host: "localhost"
```
2. set environment variable. Like `DB_HOST=localhost`
3. set command line argument. Like `--db.host=localhost`

:information_source: Refer to the [example](/examples/main.go) that illustrates how to use `ConfReader`. 

Execute  `go run examples/main.go` to run the example. 



### Install :package:
``` go
go get github.com/num30/config  
```

## Setting Configuration Values :construction_worker:

`ConfReader` merges values from all three sources in the following order:
1. File
2. Environment variables
3. Command line arguments

Setting same key in file will be overridden by environment variable and command line argument has the highest priority. 
However, you can set one key in file and other in env vars or command line args. Those will be merged. 

### Config File :memo:
#### Name
`ConfReader` will use config name property to search for a config file with that name.

#### Location
By default, config reader will search for a config file in home or in current directory. 
You can override this behavior by using `NewConfReader("myconf").WithSearchDirs("/etc/conf")` of config builder

#### Referring fields
Field names are converted from camel case starting with lower case letter. For example if it code you refer to value as `DB.DbName` then it will be converted to 
``` yaml
db:
   dbName: "mydb"
```

#### Format

Config file type could be any type supported by  [viper](https://github.com/spf13/viper#reading-config-files): JSON, TOML, YAML, HCL, INI, envfile and Java Properties files.

### Environment Variables :package:

To set a flag via environment variable, make all letters uppercase and replace '.' with '_' in path. For example: app.Server.Port -> APP_SERVER_PORT

Environment variables are prefixed with `config name` by default. For example `NewConfReader("myconf")` will search for environment variables like `MYCONF_DB_HOST` 
This behavior could be overridden by setting `NewConfReader("myconf").WithoutPrefix()`

### Command Line Arguments :computer: 

To set a configuration field via command line argument you need to pass and argument prefixes wiht `--` and lowercase field name with path. Like `--db.host=localhost`
Boolean value could be set by passing only flag name like `--verbose`

You can override name for a flag by using tag `flag:"name"` on a field. For example:

``` go
type Config struct {		
	DebugMode             bool `flag:"debug"`
}
```
You can set the flag by calling `myapp --debug`


## Validations :underage:
You can validate fields of you configuration struct by using `validate` tag. For example:

``` go
type Config struct {		
    Host           string `validate:"required"`
}
```

For full list of validation tag refer to [validator](https://github.com/go-playground/validator#baked-in-validations) documentation.

## FAQ

- How to set values for slice? 
    If we have struct like
    ```
    type SliceConf struct {
	    Slice []string
    }
    ```
    then we can set values for slice like
    -- environment variable
        `export SLICE_SLICE="a,b"`
    -- command line argument
        `myapp --slice", "a", "--slice", "b"`
    -- config file
        `slice: [ "a", "b"]`

    

##  Contributing :clap:
We love help! Contribute by forking the repo and opening a pull requests or by creating an issue.

## Credits :star:
This package is based [Viper](https://github.com/spf13/viper)
Special thanks:
- [enviper](https://github.com/iamolegga/enviper) for making environment variables work with viper
- [defaults](https://github.com/creasty/defaults) for making default values work with structures
