// Set of commonly used configuration structs

package lib

import "fmt"

// PostgresDb default configuration for Postgres database connection
type PostgresqlDb struct {
	Host       string
	Password   string
	DbName     string
	Username   string
	Port       int
	SslEnabled bool
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
