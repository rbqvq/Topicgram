package http

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"golang.org/x/net/proxy"
)

func init() {
	proxy.RegisterDialerType("http", New)
	proxy.RegisterDialerType("https", New)
}

type httpProxy struct {
	u *url.URL

	forward proxy.Dialer
}

func New(u *url.URL, forward proxy.Dialer) (proxy.Dialer, error) {
	if u == nil {
		return nil, fmt.Errorf("uri is empty")
	}

	if forward == nil {
		forward = proxy.Direct
	}

	switch u.Scheme {
	case "http", "https":
	default:
		return nil, fmt.Errorf("unsupported scheme")
	}

	s := &httpProxy{
		u:       u,
		forward: forward,
	}

	return s, nil
}

func (s *httpProxy) Dial(network, address string) (net.Conn, error) {
	switch network {
	case "tcp", "tcp4", "tcp6":
	default:
		return nil, fmt.Errorf("unsupport network: %v", network)
	}

	c, err := s.forward.Dial("tcp", s.u.Host)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("CONNECT", address, nil)
	if err != nil {
		c.Close()
		return nil, err
	}

	req.Close = false
	if s.u.User != nil {
		password, _ := s.u.User.Password()
		req.SetBasicAuth(s.u.User.Username(), password)
	}

	err = req.Write(c)
	if err != nil {
		c.Close()
		return nil, err
	}

	response, err := http.ReadResponse(bufio.NewReader(c), req)

	if response != nil && response.Body != nil {
		response.Body.Close()
	}

	if err != nil {
		c.Close()
		return nil, err
	}

	if response.StatusCode != 200 {
		c.Close()
		err = fmt.Errorf("proxy response code %d", response.StatusCode)
		return nil, err
	}

	return c, nil
}

func (s *httpProxy) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	dialer, ok := s.forward.(proxy.ContextDialer)
	if !ok {
		return s.Dial(network, address)
	}

	c, err := dialer.DialContext(ctx, "tcp", s.u.Host)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("CONNECT", address, nil)
	if err != nil {
		c.Close()
		return nil, err
	}
	req.Close = false

	if s.u.User != nil {
		password, _ := s.u.User.Password()
		req.SetBasicAuth(s.u.User.Username(), password)
	}

	err = req.Write(c)
	if err != nil {
		c.Close()
		return nil, err
	}

	response, err := http.ReadResponse(bufio.NewReader(c), req)
	if response != nil && response.Body != nil {
		response.Body.Close()
	}

	if err != nil {
		c.Close()
		return nil, err
	}

	if response.StatusCode != 200 {
		c.Close()
		err = fmt.Errorf("proxy response code %d", response.StatusCode)
		return nil, err
	}

	return c, nil
}
