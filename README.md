# config
Configuration management for golang

## Convention
- config file could be in home or in current directory
- config file type could be any type supported by [viper](https://github.com/spf13/viper#reading-config-files): JSON, TOML, YAML, HCL, INI, envfile and Java Properties files.
- Environment variables are not-prefixed (by default)
- To set a flag via environment variable, make all letters uppercase and replace ',' with '_' in path. For example: app.Server.Port -> APP_SERVER_PORT- 