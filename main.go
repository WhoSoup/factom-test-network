package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/whosoup/factom-p2p"
)

func startANetwork(ip, port string, id uint64, hook uint64) *p2p.Network {
	config := p2p.DefaultP2PConfiguration()
	config.Network = 1
	config.SeedURL = "http://localhost:81/"
	config.NodeName = fmt.Sprintf("TestNode%d", id)
	config.NodeID = id
	config.BindIP = ip
	config.ListenPort = port
	config.PeerRequestInterval = time.Second * 5
	config.PingInterval = time.Second * 5
	config.ReadDeadline = time.Second * 10
	config.WriteDeadline = time.Second * 10
	config.RedialInterval = time.Second * 15
	config.MinimumQualityScore = -1
	config.Outgoing = 32
	config.Incoming = 150

	network := p2p.NewNetwork(config)

	if id < hook {
		f, _ := os.Create(config.NodeName + ".txt")
		w := bufio.NewWriter(f)
		log.AddHook(&WriterHook{
			Writer: w,
			Node:   config.NodeName,
		})
	}
	go func() {
		time.Sleep(time.Second * time.Duration((rand.Intn(10) + 1)))
		network.Start()
	}()
	return network

}

func main() {
	log.SetLevel(log.DebugLevel)
	log.SetOutput(ioutil.Discard)

	mux := CreateSeedMux([]string{"127.1.0.1:8090\n127.2.0.2:8090\n127.3.0.3:8090"})
	go StartSeedServer("localhost:81", mux)

	var networks []*p2p.Network
	//networks = append(networks, startANetwork("", "8090", 1))
	for i := 1; i <= 10; i++ {
		networks = append(networks, startANetwork(fmt.Sprintf("127.%d.0.%d", i, i), "8090", uint64(i), 6))
	}

	mux.HandleFunc("/debug", func(rw http.ResponseWriter, req *http.Request) {
		hv := ""
		for i, c := range networks {
			if i != 0 {
				rw.Write([]byte(fmt.Sprintf("\n\n==============================\n\tNetwork %d\n==============================\n", i+1)))
			}
			a, b := c.DebugMessage()
			rw.Write([]byte(a))
			hv += b
		}
		rw.Write([]byte("\n" + hv))
	})

	start := time.Now()
	for {
		//fmt.Println(controller.)
		time.Sleep(time.Second)
		if time.Since(start) > 60*time.Second {
			start = time.Now().AddDate(50, 0, 0)
			fmt.Println("Adding network #11")
			networks = append(networks, startANetwork("127.11.0.11", "8090", 11, 0))
		}
	}
	//time.Sleep(time.H)

}
