package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/whosoup/factom-p2p"
)

func startANetwork(ip, port string, id uint64, hook uint64) *p2p.Network {
	config := p2p.DefaultP2PConfiguration()
	config.Network = 1
	config.SeedURL = "http://localhost:81/seed"
	config.NodeName = fmt.Sprintf("TestNode%d", id)
	config.NodeID = id
	config.BindIP = ip
	config.ListenPort = port
	config.PeerRequestInterval = time.Second * 15
	config.PingInterval = time.Second * 10
	config.ReadDeadline = time.Second * 60
	config.WriteDeadline = time.Second * 60
	config.RedialInterval = time.Second * 10
	config.MinimumQualityScore = -1
	config.Outgoing = 6
	config.Incoming = 150
	config.PeerIPLimitIncoming = 50
	config.PeerIPLimitOutgoing = 50
	config.ListenLimit = time.Millisecond * 50
	config.PeerFile = fmt.Sprintf("C:\\work\\debug\\peers-%s-%s-%d.json", ip, port, id)
	config.PersistInterval = time.Minute

	network := p2p.NewNetwork(config)

	if id < hook || id == 50 {
		f, _ := os.Create(config.NodeName + ".txt")
		w := bufio.NewWriter(f)
		log.AddHook(&WriterHook{
			Writer: w,
			Node:   config.NodeName,
		})
	}
	go func() {
		time.Sleep(time.Millisecond * time.Duration(id*200))
		network.Start()
	}()
	return network

}

func main() {
	log.SetLevel(log.DebugLevel)
	log.SetOutput(ioutil.Discard)
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	mux := CreateSeedMux([]string{"127.1.0.1:8090\n127.2.0.2:8090\n127.3.0.3:8090"})
	go StartSeedServer("localhost:81", mux)

	var networks []*p2p.Network
	var apps []*SimulApp
	//networks = append(networks, startANetwork("", "8090", 1))
	for i := 1; i <= 50; i++ {
		n := startANetwork(fmt.Sprintf("127.%d.0.%d", i, i), "8090", uint64(i), 6)
		networks = append(networks, n)
		apps = append(apps, NewSimulApp(byte(i), n))
	}

	count := 0

	mux.HandleFunc("/debug", func(rw http.ResponseWriter, req *http.Request) {
		hv := ""
		count = 0
		for i, c := range networks {
			if i != 0 {
				rw.Write([]byte(fmt.Sprintf("\n\n==============================\n\tNetwork %d\n==============================\n", i+1)))
			}
			a, b, cc := c.DebugMessage()
			count += cc
			rw.Write([]byte(fmt.Sprintf("%v", apps[i].seen)))
			rw.Write([]byte(a))
			hv += b
		}
		rw.Write([]byte("\n" + hv))
		rw.Write([]byte("\n127.1.0.1:8090 {color: red}\n127.2.0.2:8090 {color: green}\n127.3.0.3:8090 {color: blue}"))
	})

	time.AfterFunc(10*time.Second, func() {
		newnet := uint64(len(networks))
		fmt.Println("Adding network ", newnet)
		n := startANetwork(fmt.Sprintf("127.%d.0.%d", newnet, newnet), "8090", newnet, 0)
		networks = append(networks, n)
		apps = append(apps, NewSimulApp(byte(newnet), n))
	})

	time.AfterFunc(40*time.Second, func() {
		fmt.Println("Sending")
		for _, a := range apps {
			a.send()
		}
	})

	/*	time.AfterFunc(13*time.Second, func() {
			p := p2p.NewParcel(p2p.TypeMessage, []byte("Test"))
			p.Header.TargetPeer = p2p.FullBroadcastFlag
			networks[0].ToNetwork.Send(p)
			fmt.Println("Sent")
		})
		time.AfterFunc(15*time.Second, func() {
			for i, n := range networks {
				select {
				case p := <-n.FromNetwork.Reader():
					fmt.Printf("Network %d received parcel with message %s\n", i+1, p.Payload)
				default:
				}
			}
		})*/

	/*	time.AfterFunc(time.Second*30, func() {
		fmt.Println("Stopping networks")
		for _, n := range networks {
			n.Stop()
		}
	})*/

	/*time.AfterFunc(time.Second*18, func() {
		fmt.Println("Restarting network 0")
		networks[0].Start()
	})*/

	for {
		//fmt.Println(controller.)
		time.Sleep(time.Second * 5)
		//fmt.Println("Goroutines", runtime.NumGoroutine(), "count", count)
	}
	//time.Sleep(time.H)

}
