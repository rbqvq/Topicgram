package config

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"

	. "Topicgram/common"
	"Topicgram/pkg/proxy"

	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func init() {
	driver.SetLogger(log.New(io.Discard, "", 0))
	driver.RegisterDialContext("tcp", func(ctx context.Context, addr string) (net.Conn, error) {
		return proxy.DialContext(ctx, "tcp", addr)
	})
}

type MySQL struct {
	Host string
	Port uint16

	User     string
	Password string

	Name string

	TLS        bool
	Cert, Key  string
	ServerName string
}

func (c *MySQL) Open() (gorm.Dialector, error) {
	configs := make(url.Values)

	configs.Add("charset", "utf8mb4")
	configs.Add("collation", "utf8mb4_general_ci")
	configs.Add("parseTime", "true")
	configs.Add("loc", "Local")

	if c.TLS {
		tlsConfig := TLSConfig.Clone()

		tlsConfig.ServerName = c.Host
		if c.ServerName != "" {
			tlsConfig.ServerName = c.ServerName
		}

		if c.Cert != "" || c.Key != "" {
			certificate, err := tls.LoadX509KeyPair(c.Cert, c.Key)
			if err != nil {
				return nil, err
			}

			tlsConfig.Certificates = []tls.Certificate{certificate}
		}

		tlsConfigName := "custom"
		driver.RegisterTLSConfig(tlsConfigName, tlsConfig)

		configs.Add("allowFallbackToPlaintext", "false")
		configs.Add("tls", tlsConfigName)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s", c.User, c.Password, net.JoinHostPort(c.Host, fmt.Sprint(c.Port)), c.Name)
	if args := configs.Encode(); args != "" {
		dsn += "?" + args
	}

	return mysql.Open(dsn), nil
}
