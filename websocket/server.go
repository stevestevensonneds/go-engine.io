package websocket

import (
	"io"
	"net/http"

	"github.com/googollee/go-engine.io/message"
	"github.com/googollee/go-engine.io/parser"
	"github.com/googollee/go-engine.io/transport"
	"github.com/gorilla/websocket"
)

type Server struct {
	callback transport.Callback
	conn     *websocket.Conn
}

func NewServer(w http.ResponseWriter, r *http.Request, callback transport.Callback) (transport.Server, error) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:    10240,
		WriteBufferSize:   10240,
		EnableCompression: callback.GetEnableCompression(),
		CheckOrigin:       func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	if callback.GetEnableCompression() {
		conn.EnableWriteCompression(true)
		conn.SetCompressionLevel(1)
	}

	ret := &Server{
		callback: callback,
		conn:     conn,
	}

	go ret.serveHTTP(w, r)

	return ret, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
}

func (s *Server) NextWriter(msgType message.MessageType, packetType parser.PacketType) (io.WriteCloser, error) {
	wsType, newEncoder := websocket.TextMessage, parser.NewStringEncoder
	if msgType == message.MessageBinary {
		wsType, newEncoder = websocket.BinaryMessage, parser.NewBinaryEncoder
	}

	w, err := s.conn.NextWriter(wsType)
	if err != nil {
		return nil, err
	}
	ret, err := newEncoder(w, packetType)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *Server) Close() error {
	return s.conn.Close()
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	defer s.callback.OnClose(s)

	for {
		t, r, err := s.conn.NextReader()
		if err != nil {
			s.conn.Close()
			return
		}

		switch t {
		case websocket.TextMessage:
			fallthrough
		case websocket.BinaryMessage:
			decoder, err := parser.NewDecoder(r)
			if err != nil {
				return
			}
			s.callback.OnPacket(decoder)
			decoder.Close()
		}
	}
}
