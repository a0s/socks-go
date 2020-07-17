package main

import (
	socks "github.com/a0s/socks-go"
	"log"
	"net"
	"time"
)

func main() {
	conn, err := net.Listen("tcp", ":1080")
	if err != nil {
		log.Fatal(err)
	}

	for {
		c, err := conn.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		log.Printf("connected from %s", c.RemoteAddr())

		d := net.Dialer{Timeout: 10 * time.Second}
		s := socks.Conn{Conn: c, Dial: d.Dial, Socks4Enabled: true, Socks5Enabled: true}
		go s.Serve()
	}
}
