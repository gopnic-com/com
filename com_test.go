package com

import (
	"errors"
	"net/http/httptest"
	"testing"
)

func TestNewServer(t *testing.T) {
	s1 := NewServer("a", "b")
	if s1.network != "a" {
		t.Error("wrong network, expected a")
	}
	if s1.address != "b" {
		t.Error("wrong address, expected b")
	}
}

func TestNewClient(t *testing.T) {
	c1 := NewClient("a", "b")
	if c1.network != "a" {
		t.Error("wrong network, expected a")
	}
	if c1.address != "b" {
		t.Error("wrong address, expected b")
	}
	c2 := NewClient(srvOK.network, srvOK.address)
	b2, err := c2.Request([]byte("hello"))
	if err != nil {
		t.Error("failed to make request: " + err.Error())
	}
	if string(b2) != "OK" {
		t.Error("invalid response, expected OK")
	}
	c3 := NewClient(srvNOK.network, srvNOK.address)
	_, err = c3.Request([]byte("hello"))
	if err == nil || err.Error() != "NOK" {
		t.Error("failed to make request: " + err.Error())
	}
}

var (
	srvOK  *Server
	srvNOK *Server
)

func init() {
	srvOK = NewServer("tcp", getAddr())
	srvNOK = NewServer("tcp", getAddr())
	handleOK := func(p *Package) ([]byte, error) {
		return []byte("OK"), nil
	}
	handleNOK := func(p *Package) ([]byte, error) {
		return nil, errors.New("NOK")
	}
	go func() {
		if err := srvOK.Listen(handleOK); err != nil {
			panic(err)
		}
	}()
	go func() {
		if err := srvNOK.Listen(handleNOK); err != nil {
			panic(err)
		}
	}()
}

func getAddr() string {
	s := httptest.NewUnstartedServer(nil)
	return s.Listener.Addr().String()
}
