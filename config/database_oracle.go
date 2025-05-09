package config

import (
	. "Topicgram/common"
	"Topicgram/pkg/proxy"
	"context"
	"net"

	"database/sql"
	"database/sql/driver"

	go_ora "github.com/sijms/go-ora/v2"
	oracle "gitlab.com/CoiaPrant/gorm-oracle"
	"gorm.io/gorm"
)

func init() {
	sql.Register("oracle-proxy", &oracleDriver{})
}

type oracleDriver struct{}

func (*oracleDriver) Open(name string) (driver.Conn, error) {
	connecter, _ := (&go_ora.OracleDriver{}).OpenConnector(name)
	connecter.(*go_ora.OracleConnector).Dialer(&oracleProxy{})
	return connecter.Connect(context.Background())
}

type oracleProxy struct{}

func (*oracleProxy) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return proxy.DialContext(ctx, network, address)
}

type Oracle struct {
	Host string
	Port uint16

	User     string
	Password string

	Service string

	AuthType                   string
	TLS                        bool
	WalletPath, WalletPassword string
}

func (c *Oracle) Open() (gorm.Dialector, error) {
	options := make(map[string]string)

	if c.AuthType != "" {
		options["AUTH TYPE"] = c.AuthType
	}

	if c.WalletPath != "" {
		options["WALLET"] = c.WalletPath
	}

	if c.WalletPassword != "" {
		options["WALLET PASSWORD"] = c.WalletPassword
	}

	if c.TLS {
		options["SSL"] = "TRUE"

		if !TLSConfig.InsecureSkipVerify {
			options["SSL VERIFY"] = "TRUE"
		}
	}

	return oracle.New(oracle.Config{
		DriverName:        "oracle-proxy",
		DSN:               oracle.BuildUrl(c.Host, int(c.Port), c.Service, c.User, c.Password, options),
		DefaultStringSize: 4000,
	}), nil
}
