package service

import (
	"net"
	"testing"
)

func TestIsPortFree(t *testing.T) {
	// Bind a port, then verify IsPortFree returns false for it.
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	if IsPortFree(port) {
		t.Errorf("IsPortFree(%d) = true, want false (port is bound)", port)
	}
}

func TestNextFreePort(t *testing.T) {
	port, err := NextFreePort(19000, 19100)
	if err != nil {
		t.Fatalf("NextFreePort(19000, 19100) returned error: %v", err)
	}
	if port < 19000 || port > 19100 {
		t.Errorf("NextFreePort returned %d, outside range [19000, 19100]", port)
	}
}

func TestNextFreePortNoRange(t *testing.T) {
	// Bind a single port and ask NextFreePort to find within that one port.
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	_, err = NextFreePort(port, port)
	if err == nil {
		t.Errorf("NextFreePort(%d, %d) expected error when port is occupied", port, port)
	}
}
