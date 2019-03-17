package p2p

import (
	"errors"
	"fmt"
	"net"
)

var discard = Protocol{
	Name:   "discard",
	Length: 1,
	Run: func(p *Peer, rw MsgReadWriter) error {
		for {
			msg, err := rw.ReadMsg()
			if err != nil {
				return err
			}
			fmt.Printf("discarding %d\n", msg.Code)
			if err = msg.Discard(); err != nil {
				return err
			}
		}
	},
}

func testPeer(protos []Protocol) (func(), *conn, *Peer, <-chan DiscReason) {
	fd1, fd2 := net.Pipe()
	c1 := &conn{fd: fd1, transport: newTestTransport(randomID(), fd1)}
	c2 := &conn{fd: fd2, transport: newTestTransport(randomID(), fd2)}
	for _, p := range protos {
		c1.caps = append(c1.caps, p.cap())
		c2.caps = append(c2.caps, p.cap())
	}

	peer := newPeer(c1, protos)
	errc := make(chan DiscReason, 1)
	go func() { errc <- peer.run() }()

	closer := func() { c2.close(errors.New("close func called")) }
	return closer, c2, peer, errc
}
