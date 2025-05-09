package config

import (
	"fmt"
	"strings"

	. "Topicgram/common"
	"Topicgram/pkg/proxy"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Postgres struct {
	Host string
	Port uint16

	User     string
	Password string

	Name string

	TLS       bool
	Cert, Key string
}

func (c *Postgres) Open() (gorm.Dialector, error) {
	configs := make([]string, 0, 9)
	configs = append(configs, fmt.Sprintf("host=%s", c.Host))
	configs = append(configs, fmt.Sprintf("port=%d", c.Port))
	configs = append(configs, fmt.Sprintf("user=%s", c.User))
	configs = append(configs, fmt.Sprintf("password=%s", c.Password))
	configs = append(configs, fmt.Sprintf("database=%s", c.Name))

	if c.TLS {
		if TLSConfig.InsecureSkipVerify {
			configs = append(configs, fmt.Sprintf("sslmode=%s", "require"))
		} else {
			configs = append(configs, fmt.Sprintf("sslmode=%s", "verify-full"))
		}

		if c.Cert != "" || c.Key != "" {
			configs = append(configs, fmt.Sprintf("sslcert=%s", c.Cert))
			configs = append(configs, fmt.Sprintf("sslkey=%s", c.Key))
		}

		configs = append(configs, "sslsni=1")
	} else {
		configs = append(configs, "sslmode=disable")
	}

	config, err := pgx.ParseConfig(strings.Join(configs, " "))
	if err != nil {
		return nil, err
	}

	config.DialFunc = proxy.DialContext

	return postgres.New(postgres.Config{
		DriverName: "pgx",
		DSN:        stdlib.RegisterConnConfig(config),
	}), nil
}
