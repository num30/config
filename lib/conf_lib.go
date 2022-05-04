// Set of commonly used configuration structs

package lib

import "fmt"

// PostgresqlDb is a default configuration for Postgres database connection
type PostgresqlDb struct {
	Host       string `default:"localhost"`
	Password   string `default:"pass"`
	DbName     string `default:"postgres"`
	Username   string `default:"postgres"`
	Port       int    `default:"5432"`
	SslEnabled bool   `default:"false"`
}

func (p *PostgresqlDb) GetConnString() string {
	sslMode := "disable"
	if p.SslEnabled {
		sslMode = "enable"
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s database=%s port=%d sslmode=%s",
		p.Host, p.Username, p.Password, p.DbName, p.Port, sslMode)
	return dsn
}
