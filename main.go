package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/whosoup/factom-p2p"
)

func startANetwork(ip, port string, id uint32, hook uint32) *p2p.Network {
	config := p2p.DefaultP2PConfiguration()
	config.Network = 1
	config.SeedURL = "http://localhost:81/seed"
	config.NodeName = fmt.Sprintf("TestNode%d", id)
	//config.NodeID = id
	config.BindIP = ip
	config.ListenPort = port
	config.PeerRequestInterval = time.Second * 15
	config.PingInterval = time.Second * 10
	config.ReadDeadline = time.Second * 60
	config.WriteDeadline = time.Second * 60
	config.RedialInterval = time.Second * 10
	config.RoundTime = time.Second * 10
	config.Target = 8
	config.Max = 16
	config.Drop = 6
	config.MinReseed = 3
	config.MinimumQualityScore = -1
	config.Outgoing = 4
	config.Incoming = 150
	config.PeerIPLimitIncoming = 50
	config.PeerIPLimitOutgoing = 50
	config.ListenLimit = time.Millisecond * 50
	config.PersistFile = fmt.Sprintf("C:\\work\\debug\\peers-%s-%s-%d.json", ip, port, id)
	config.PersistInterval = time.Minute
	config.Special = "127.1.0.1:8110,127.41.0.41:8110,127.42.0.42:8110"
	if id%2 == 0 {
		config.ProtocolVersion = 9
	} else {
		config.ProtocolVersion = 10
	}

	if id != 1 {
		config.EnablePrometheus = false
	}

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
	mux := CreateSeedMux([]string{"127.1.0.1:8110\n127.2.0.2:8110\n127.3.0.3:8110"})
	go StartSeedServer("localhost:81", mux)

	var networks []*p2p.Network
	var apps []*SimulApp
	//networks = append(networks, startANetwork("", "8110", 1))
	for i := 1; i <= 50; i++ {
		n := startANetwork(fmt.Sprintf("127.%d.0.%d", i, i), "8110", uint32(i), 6)
		networks = append(networks, n)
		apps = append(apps, NewSimulApp(byte(i), n))
		time.Sleep(time.Millisecond)
	}

	promux := http.NewServeMux()
	promux.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(fmt.Sprintf(":%d", 82), promux)

	count := 0

	mux.HandleFunc("/debug", func(rw http.ResponseWriter, req *http.Request) {
		count = 0
		for i, c := range networks {
			if i != 0 {
				rw.Write([]byte(fmt.Sprintf("\n\n==============================\n\tNetwork %d\n==============================\n", i+1)))
			}
			a, _, cc := c.DebugMessage()
			count += cc
			rw.Write([]byte(fmt.Sprintf("%v", apps[i].seen)))
			rw.Write([]byte(a))
		}
	})

	mux.HandleFunc("/halfviz", func(rw http.ResponseWriter, req *http.Request) {
		count = 0
		for _, c := range networks {
			_, b, _ := c.DebugMessage()
			rw.Write([]byte(b))
		}

		rw.Write([]byte("\n127.1.0.1:8110 {color: red}\n127.2.0.2:8110 {color: green}\n127.3.0.3:8110 {color: blue}"))
	})

	mux.HandleFunc("/ana", func(rw http.ResponseWriter, req *http.Request) {
		var min, max, connections, total, rounds int
		min = math.MaxInt32
		for _, c := range networks {
			cons := c.Total()
			r := c.Rounds()
			if r > rounds {
				rounds = r
			}
			total++
			connections += cons
			if cons < min {
				min = cons
			}
			if cons > max {
				max = cons
			}
		}

		mean := float64(connections) / float64(total)
		var deviation float64

		for _, c := range networks {
			cons := c.Total()
			dev := float64(cons) - mean
			if dev < 0 {
				dev = -dev
			}
			deviation += dev
		}

		msg := fmt.Sprintf("Total Connections: %d\n", total)
		msg += fmt.Sprintf("Rounds: %d", rounds)
		msg += fmt.Sprintf("Min: %d\n", min)
		msg += fmt.Sprintf("Max: %d\n", max)
		msg += fmt.Sprintf("Average Connections: %f\n", mean)
		msg += fmt.Sprintf("Deviation: %f\n", deviation)

		rw.Write([]byte(msg))
	})

	mux.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`<!doctype html><html><body><ul>
		<li><a href="/debug">global network</a></li>
		<li><a href="/halfviz">halfviz</a></li>
		<li><a href="/ana">network imbalance analysis</a></li>
		<li><a href="http://localhost:82/metrics">prometheus</a></li>
		<li><a href="http://localhost:8070/debug">network 0 debug</a></li>
		<li><a href="http://localhost:8070/stats">network 0 stats</a></li>
		</ul></body></html>`))
	})

	time.AfterFunc(10*time.Second, func() {
		newnet := uint32(len(networks))
		fmt.Println("Adding network ", newnet)
		n := startANetwork(fmt.Sprintf("127.%d.0.%d", newnet, newnet), "8110", newnet, 0)
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
