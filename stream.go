package ttrpc

import (
	"context"
	"errors"
	"net"
)

type StreamInf interface {
	SendMsg([]byte)
	RecvMsg() ([]byte, error)
}

type ClientStream struct {
	streamID uint32
	service  string
	method   string
	calls    chan *callRequest
	recvChan chan []byte
}

type ServerStream struct {
	streamID uint32
	service  string
	method   string
	sendChan chan []byte
	recvChan chan []byte
}

func NewServerStream(conn net.Conn, service string, method string) *ServerStream {
	streamID := GetStreamID()
	//streamMap.Store(streamID, recvChan)
	recvChan := make(chan []byte, 100)
	sendChan := make(chan []byte, 100)

	return &ServerStream{
		streamID: streamID,
		service:  service,
		method:   method,
		sendChan:sendChan,
		recvChan: recvChan,
	}
}

func (s *ServerStream) SendMsg(data []byte) {
	select {
	case s.sendChan <- data:
		return
	}
}

func (s *ServerStream) RecvMsg() ([]byte, error) {
	// Todo: timeout
	select {
	case data := <-s.recvChan:
		return data, nil
	}
}

func (s *ServerStream) send(data []byte) {
	s.recvChan <- data
}

func (s *ServerStream) recv() []byte {
	return <-s.sendChan
}

func NewClientStream(conn net.Conn, calls chan *callRequest, service string, method string) *ClientStream {
	streamID := GetStreamID()
	recvChan := make(chan []byte, 100)
	streamMap.Store(streamID, recvChan)

	return &ClientStream{
		streamID: streamID,
		calls:    calls,
		service:  service,
		method:   method,
		recvChan: recvChan,
	}
}

func (s *ClientStream) SendMsg(data []byte) {
	call := &callRequest{
		ctx:      context.Background(),
		streamID: s.streamID,
		msgType:  MessageTypeStream,
		req: &Request{
			Service: s.service,
			Method:  s.method,
			Payload: data,
		},
		resp: nil,
		errs: make(chan error, 1),
	}

	s.calls <- call
}

func (s *ClientStream) RecvMsg() ([]byte, error) {
	// Todo: timeout
	select {
	case data, ok := <-s.recvChan:
		if ok {
			return data, nil
		} else {
			return []byte{}, errors.New("stream end")
		}
	}
}
