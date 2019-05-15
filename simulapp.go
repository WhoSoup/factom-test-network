package main

import (
	"github.com/whosoup/factom-p2p"
)

type SimulApp struct {
	seen []int
	net  *p2p.Network
	id   byte
}

func NewSimulApp(id byte, n *p2p.Network) *SimulApp {
	sa := new(SimulApp)
	sa.net = n
	sa.id = id
	sa.seen = make([]int, 51)

	go sa.read()

	return sa
}

func (sa *SimulApp) read() {
	for {
		select {
		case p := <-sa.net.FromNetwork.Reader():
			s := sa.seen[p.Payload[0]-1]
			sa.seen[p.Payload[0]-1]++

			if p.Payload[1] > 0 && s == 0 {
				p.Address = p2p.Broadcast
				p.Payload[1]--
				sa.net.ToNetwork.Send(p)
			}
		}
	}
}

func (sa *SimulApp) send() {
	p := p2p.NewMessage(p2p.Broadcast, []byte{sa.id, 5})
	sa.net.ToNetwork.Send(p)
}
