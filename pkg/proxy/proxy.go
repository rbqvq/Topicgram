package proxy

import (
	"context"
	"errors"
	"net"
	"net/url"

	_ "Topicgram/pkg/proxy/http"

	"golang.org/x/net/proxy"
)

var dialer proxy.Dialer = proxy.Direct

func Register(u *url.URL) error {
	if u == nil {
		return errors.New("no proxy can be used")
	}

	d, err := proxy.FromURL(u, proxy.Direct)
	if err != nil {
		return err
	}

	dialer = d
	return nil
}

func Dial(network, address string) (net.Conn, error) {
	switch network {
	case "unix", "unixgram", "unixpacket":
		return net.Dial(network, address)
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
		var dialer net.Dialer
		return dialer.DialContext(ctx, network, address)
	default:
		if dialer, ok := dialer.(proxy.ContextDialer); ok {
			return dialer.DialContext(ctx, network, address)
		}

		return Dial(network, address)
	}
}
