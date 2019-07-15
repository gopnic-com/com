package com

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
)

// NewServer is returning a new instance of Server.
func NewServer(network, address string) *Server {
	return &Server{networking: newNetworking(network, address)}
}

// NewClient returns a new instance of Client.
func NewClient(network, address string) *Client {
	return &Client{newNetworking(network, address)}
}

// Server is the com server.
type Server struct {
	*networking
	handler Handler
}

// Listen starts listening and handle connections.
func (s *Server) Listen(handler Handler) error {
	s.handler = handler
	listener, err := net.Listen(s.network, s.address)
	if err != nil {
		return err
	}
	defer func() {
		if err := listener.Close(); err != nil {
			s.ErrHandler(err)
		}
	}()
	for {
		conn, err := listener.Accept()
		if err != nil {
			s.ErrHandler(err)
			continue
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			s.ErrHandler(err)
		}
	}()
	pReq, err := parsePackage(conn)
	if err != nil {
		s.ErrHandler(err)
		return
	}
	for _, m := range s.middleware {
		if err := m(pReq); err != nil {
			s.ErrHandler(err)
			return
		}
	}
	var (
		pRes *Package
	)
	b, err := s.handler(pReq)
	if err != nil {
		pRes = newPackage(TypeErr, []byte(err.Error()))
	} else {
		pRes = newPackage(TypeDat, b)
	}
	if _, err := conn.Write(pRes.Bytes()); err != nil {
		s.ErrHandler(err)
	}
}

func newNetworking(network, address string) *networking {
	return &networking{
		network:    network,
		address:    address,
		middleware: make([]Middleware, 0),
		ErrHandler: defaultErrorHandler,
	}
}

type networking struct {
	network    string
	address    string
	middleware []Middleware
	ErrHandler ErrorHandler
}

// RegisterMiddleware registers new middleware for the request/response.
func (n *networking) RegisterMiddleware(handlers ...Middleware) {
	for _, m := range handlers {
		n.middleware = append(n.middleware, m)
	}
}

func newPackage(t Type, d []byte) *Package {
	return &Package{t, d}
}

func parsePackage(conn net.Conn) (*Package, error) {
	l := make([]byte, 20)
	if _, err := conn.Read(l); err != nil {
		return nil, err
	}
	lu, err := strconv.ParseUint(string(l), 10, 0)
	if err != nil {
		return nil, err
	}
	dat := make([]byte, lu)
	if _, err := conn.Read(dat); err != nil {
		return nil, err
	}
	t := dat[:1]
	tp, err := strconv.ParseUint(string(t), 10, 0)
	if err != nil {
		return nil, err
	}
	return newPackage(Type(tp), dat[1:]), nil
}

// Client is the com client.
type Client struct {
	*networking
}

// Request makes a network request to server.
func (c *Client) Request(data []byte) ([]byte, error) {
	conn, err := net.Dial(c.network, c.address)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			c.ErrHandler(err)
		}
	}()
	pReq := newPackage(TypeDat, data)
	if _, err := conn.Write(pReq.Bytes()); err != nil {
		return nil, err
	}
	pRes, err := parsePackage(conn)
	if err != nil {
		return nil, err
	}
	if err := pRes.Error(); err != nil {
		return nil, err
	}
	return pRes.Data, nil
}

// Handler is the request handler.
type Handler func(*Package) ([]byte, error)

// Middleware is request/response middleware.
type Middleware func(*Package) error

var (
	sep = make([]byte, 0)
)

// Package is a network package.
type Package struct {
	Type Type
	Data []byte
}

// Error returns package error if package type is error.
func (p *Package) Error() error {
	if p.Type.IsErr() {
		return errors.New(string(p.Data))
	}
	return nil
}

// Size returns package size.
func (p *Package) Size() uint64 {
	return uint64(len(p.Data) + 1)
}

// SizeBytes returns formatted package size as byte slice.
func (p *Package) SizeBytes() []byte {
	return []byte(fmt.Sprintf("%020d", p.Size()))
}

// TypeBytes returns the package type as byte slice.
func (p *Package) TypeBytes() []byte {
	return []byte(strconv.FormatUint(uint64(p.Type), 10))
}

// Bytes returns the whole package a byte slice.
func (p *Package) Bytes() []byte {
	return bytes.Join([][]byte{
		p.SizeBytes(),
		p.TypeBytes(),
		p.Data,
	}, sep)
}

const (
	// TypeDat is the constant for data type.
	TypeDat = Type(1)

	// TypeErr is the constant for error type.
	TypeErr = Type(2)
)

// Type is a package type.
type Type uint

// IsDat returns true if the type is data.
func (t Type) IsDat() bool {
	return t == TypeDat
}

// IsErr returns true if the type is an error.
func (t Type) IsErr() bool {
	return t == TypeErr
}

// ErrorHandler is a default type for handling background errors.
type ErrorHandler func(error)

func defaultErrorHandler(err error) {
	log.Println(err)
}
