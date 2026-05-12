package service

import (
	"fmt"
	"net"
)

// IsPortFree returns true if no process is bound to the given TCP port.
func IsPortFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

// NextFreePort walks [start, end] and returns the first free port.
func NextFreePort(start, end int) (int, error) {
	for p := start; p <= end; p++ {
		if IsPortFree(p) {
			return p, nil
		}
	}
	return 0, fmt.Errorf("no free port in range %d–%d", start, end)
}
