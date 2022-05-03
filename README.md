# config
ConfReader is a library that supports reading configuration into struct from files, environment variable and command line arguments.
All you need is delare your config structure and call `Read` method.

```
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
If you want to change the DB Host of your applications the you can do any of the following:
1. creating config file in json, yaml, toml.For example myconf.yaml
``` 
db:
   host: "localhost"
```
2. setting environment variables. Like `DB_HOST=localhost`
3. setting command line arguments. Like `--db_host=localhost`

`ConfReader` merges values from all three sources in the following order:
1. File
2. Environment variables
3. Command line arguments

Refer to the [example](/examples/main.go) that illustrates how to use `ConfReader`.


## File config
#### Name
`ConfReader` will use config name property to search for a config file with that name.

#### Location
By default config reader will search for a config file in home or in current directory. 
You can override this behavior by useing `NewConfReader("myconf").WithSearchDirs("/etc/conf")` of config builder

#### Refering fields
Fieled names are converted from camel case starting with lower case letter. For example if it code you refer to value as `DB.DbName` then it will be converted to 
``` yaml
db:
   dbName: "mydb"
```

#### Format

Config file type could be any type supported by  [viper](https://github.com/spf13/viper#reading-config-files): JSON, TOML, YAML, HCL, INI, envfile and Java Properties files.

## Environment variables

To set a flag via environment variable, make all letters uppercase and replace '.' with '_' in path. For example: app.Server.Port -> APP_SERVER_PORT

Environment variables are prefixed with `config name` by default. For example `NewConfReader("myconf")` will search for environment variables like `MYCONF_DB_HOST` 
This behavior could be overridden by setting `NewConfReader("myconf").WithoutPrefix()`

## Command line arguments

To set a configuration field via command line argument you need to pass and argument prefixes wiht `--` and lowercase field name with path. Like `--db.host=localhost`

Boolean value could be set by passing only flag name like `--verbose`
 