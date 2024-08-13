package netutil

import (
	"fmt"
	"net"
)

// GetFreeLocalhostTCPPort returns a free TCP port on localhost.
func GetFreeLocalhostTCPPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("failed to find free port to listen on: %w", err)
	}
	defer listener.Close()

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("failed to get port from listener")
	}

	return tcpAddr.Port, nil
}
