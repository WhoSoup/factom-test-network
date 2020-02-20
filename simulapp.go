package main

import (
	"encoding/json"
	"fmt"
	"time"

	p2p "github.com/whosoup/factom-p2p"
)

type SimulApp struct {
	seen    []int
	net     *p2p.Network
	id      int
	counter int
	stop    chan interface{}
}
type Msg struct {
	Id          int
	Count       int
	Rebroadcast int
}

func NewSimulApp(id int, n *p2p.Network) *SimulApp {
	sa := new(SimulApp)
	sa.net = n
	sa.id = id
	sa.seen = make([]int, 1)
	sa.stop = make(chan interface{})

	go sa.read()
	go sa.emit()

	return sa
}

func (sa *SimulApp) Stop() { close(sa.stop) }

func (sa *SimulApp) update(i, count int) {
	for len(sa.seen) <= i {
		sa.seen = append(sa.seen, 0)
	}
	if count > sa.seen[i] {
		sa.seen[i] = count
	}
}

func (sa *SimulApp) get(i int) int {
	for len(sa.seen) <= i {
		sa.seen = append(sa.seen, 0)
	}
	return sa.seen[i]
}

func (sa *SimulApp) read() {
	for {
		select {
		case <-sa.stop:
			return
		case p := <-sa.net.FromNetwork:

			msg := new(Msg)
			if err := json.Unmarshal(p.Payload, msg); err != nil {
				fmt.Println(err)
				continue
			}

			if sa.id == 0 {
				fmt.Printf("1 recv [%d %d %d] %s\n", msg.Id, msg.Count, msg.Rebroadcast, p.Payload)
			}

			if msg.Count > sa.get(msg.Id) {
				sa.update(msg.Id, msg.Count)
				if msg.Rebroadcast > 0 {
					msg.Rebroadcast--
					js, _ := json.Marshal(msg)
					p = p2p.NewParcel(p2p.Broadcast, js)
					sa.net.ToNetwork <- p
					if sa.id == 0 {
						fmt.Printf("1 sent [%d %d %d]\n", msg.Id, msg.Count, msg.Rebroadcast)
					}
				}
			}
		}
	}
}

func (sa *SimulApp) emit() {
	t := time.NewTicker(time.Second)
	for range t.C {
		select {
		case <-sa.stop:
			return
		default:
		}
		sa.counter++
		msg := Msg{Id: sa.id, Count: sa.counter, Rebroadcast: 5}
		js, _ := json.Marshal(msg)
		p := p2p.NewParcel(p2p.Broadcast, js)
		sa.net.ToNetwork <- p
	}
}
