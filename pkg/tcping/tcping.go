package tcping

import (
	"net"
	"time"

	"golang.org/x/net/proxy"
)

const defaultDialTimeout = 3 * time.Second

var defaultDialer = &net.Dialer{Timeout: 3 * time.Second}

func Check(dialer proxy.Dialer, target string) bool {
	if dialer == nil {
		dialer = defaultDialer
	}

	conn, err := dialer.Dial("tcp", target)
	if err != nil {
		return false
	}

	conn.Close()
	return true
}

func Ping(dialer proxy.Dialer, target string) (time.Duration, error) {
	if dialer == nil {
		dialer = defaultDialer
	}

	fromTime := time.Now()
	conn, err := dialer.Dial("tcp", target)
	elapsedTime := time.Since(fromTime)

	if err != nil {
		return 0, err
	}

	conn.Close()
	return elapsedTime, nil
}
