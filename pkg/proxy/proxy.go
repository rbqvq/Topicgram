package proxy

import (
	"context"
	"errors"
	"net"
	"net/url"
	"time"

	"gitlab.com/CoiaPrant/lib/net/diagnostics/tcping"
	"gitlab.com/CoiaPrant/lib/net/sockopt"
	"golang.org/x/net/proxy"
)

var (
	defaultDialer = &net.Dialer{
		Timeout: 10 * time.Second,
		KeepAliveConfig: net.KeepAliveConfig{
			Enable:   true,
			Idle:     -1,
			Interval: -1,
			Count:    -1,
		},
		Control: sockopt.DialControl,
	}

	current *url.URL
	dialer  proxy.Dialer = defaultDialer
)

func Register(u *url.URL) error {
	if u == nil {
		return errors.New("no proxy can be proxied")
	}

	d, err := proxy.FromURL(u, defaultDialer)
	if err != nil {
		return err
	}

	current = u
	dialer = d
	return nil
}

func Replace(d proxy.Dialer) {
	dialer = d
}

func Dialer() proxy.Dialer {
	return dialer
}

func Dial(network, address string) (net.Conn, error) {
	switch network {
	case "unix", "unixgram", "unixpacket":
		return defaultDialer.Dial(network, address)
	default:
		return dialer.Dial(network, address)
	}
}

func DialRPC(ctx context.Context, address string) (net.Conn, error) {
	return DialContext(ctx, "tcp", address)
}

func DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	switch network {
	case "unix", "unixgram", "unixpacket":
		return defaultDialer.DialContext(ctx, network, address)
	default:
		if dialer, ok := dialer.(proxy.ContextDialer); ok {
			return dialer.DialContext(ctx, network, address)
		}

		return Dial(network, address)
	}
}

func Ping() error {
	if current == nil {
		return errors.New("no proxy registered")
	}

	_, err := tcping.Ping(current.Host)
	return err
}
