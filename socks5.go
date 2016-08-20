package socks

import (
	"fmt"
	"io"
	"log"
	"net"
	//"strconv"
	"encoding/binary"
)

type socks5Conn struct {
	//addr        string
	clientConn net.Conn
	serverConn net.Conn
	dial       dialFunc
}

func (s5 *socks5Conn) Serve() {
	defer s5.Close()
	if err := s5.handshake(); err != nil {
		log.Println(err)
		return
	}

	if err := s5.processRequest(); err != nil {
		log.Println(err)
		return
	}
}

func (s5 *socks5Conn) handshake() error {
	// version has already readed by socksConn.Serve()
	// only process auth methods here

	buf := make([]byte, 258)
	n, err := io.ReadAtLeast(s5.clientConn, buf, 1)
	if err != nil {
		return err
	}

	l := int(buf[0]) + 1
	if n < l {
		_, err := io.ReadFull(s5.clientConn, buf[n:l])
		if err != nil {
			return err
		}
	}

	// no auth required
	s5.clientConn.Write([]byte{0x05, 0x00})

	return nil
}

func (s5 *socks5Conn) processRequest() error {
	buf := make([]byte, 258)
	n, err := io.ReadAtLeast(s5.clientConn, buf, 10)
	if err != nil {
		return err
	}
	if buf[0] != socks5Version {
		return fmt.Errorf("error version %s", buf[0])
	}

	// only support connect
	if buf[1] != cmdConnect {
		return fmt.Errorf("unsupported command %s", buf[1])
	}

	hlen := 0
	host := ""
	msglen := 0
	switch buf[3] {
	case addrTypeIPv4:
		hlen = 4
	case addrTypeDomain:
		hlen = int(buf[4]) + 1
	case addrTypeIPv6:
		hlen = 16
	}

	msglen = 6 + hlen

	if n < msglen {
		_, err := io.ReadFull(s5.clientConn, buf[n:msglen])
		if err != nil {
			return err
		}
	}

	// get target address
	addr := buf[4 : 4+hlen]
	if buf[3] == addrTypeDomain {
		host = string(addr[1:])
	} else {
		host = net.IP(addr).String()
	}

	port := binary.BigEndian.Uint16(buf[msglen-2 : msglen])

	target := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	// reply user with connect success
	// if dial to target failed, user will receive connection reset
	s5.clientConn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01})

	log.Printf("connecing to %s", target)

	// connect to the target
	s5.serverConn, err = s5.dial("tcp", target)
	if err != nil {
		return err
	}

	// enter data exchange
	s5.forward()

	return nil
}

func (s5 *socks5Conn) forward() {
	go io.Copy(s5.clientConn, s5.serverConn)
	io.Copy(s5.serverConn, s5.clientConn)
}

func (s5 *socks5Conn) Close() {
	if s5.serverConn != nil {
		s5.serverConn.Close()
	}
	if s5.clientConn != nil {
		s5.clientConn.Close()
	}
}
