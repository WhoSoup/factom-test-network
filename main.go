package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	p2p "github.com/whosoup/factom-p2p"
)

const LogMax = 5
const Port = "8888"

var ig *IPGenerator

func versionFromID(id uint32) uint16 {
	if id%3 == 0 {
		return 9
	} else if id%3 == 1 {
		return 10
	} else {
		return 11
	}
}

func startANetwork() *p2p.Network {
	id, ip := ig.Next()

	port := Port
	/*	if id == 1 {
		ip = "localhost"
		port = "7999"
	}*/

	config := p2p.DefaultP2PConfiguration()
	config.Network = 1
	config.SeedURL = "http://localhost:81/seed"
	config.NodeName = fmt.Sprintf("TestNode%d", id)
	//config.NodeID = id
	config.BindIP = ip
	config.ListenPort = port
	config.PeerRequestInterval = time.Second * 5
	config.PingInterval = time.Second * 10
	config.ReadDeadline = time.Second * 60
	config.WriteDeadline = time.Second * 60
	config.RedialInterval = time.Second * 10
	config.RoundTime = time.Minute
	config.PeerShareAmount = 3
	config.TargetPeers = 32
	config.MaxPeers = 36
	config.DropTo = 28
	config.MinReseed = 3
	config.MaxIncoming = 36
	config.PeerIPLimitIncoming = 50
	config.PeerIPLimitOutgoing = 50
	config.ListenLimit = time.Millisecond * 50
	//config.PeerCacheFile = fmt.Sprintf("C:\\work\\debug\\peers-%s-%s-%d.json", ip, Port, id)
	config.Fanout = 8
	//config.PersistInterval = time.Minute
	//config.Special = "127.1.0.1:8888,127.41.0.41:8888,127.42.0.42:8888"
	config.ProtocolVersion = versionFromID(id)

	if id == 1 {
		config.Special = "127.0.0.23:8888"
	}

	if id != 1 {
		config.EnablePrometheus = false
	}

	network, err := p2p.NewNetwork(config)
	if err != nil {
		panic(err)
	}

	if id <= LogMax || id == 50 {
		f, _ := os.Create(config.NodeName + ".txt")
		w := bufio.NewWriter(f)
		log.AddHook(&WriterHook{
			Writer: w,
			Node:   config.NodeName,
		})
	}
	go func() {
		time.Sleep(time.Millisecond * time.Duration(id*200))
		network.Run()
	}()
	return network

}

func main() {

	var networkCount = flag.Int("n", 50, "number of networks to start")
	flag.Parse()

	log.SetLevel(log.DebugLevel)
	log.SetOutput(ioutil.Discard)
	ig = NewIPGenerator()
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	mux := CreateSeedMux([]string{"127.0.0.1:8888\n127.0.0.2:8888\n127.0.0.3:8888"})
	go StartSeedServer("localhost:81", mux)

	var networks []*p2p.Network
	var apps []*SimulApp
	//networks = append(networks, startANetwork("", "8888", 1))
	for i := 1; i <= *networkCount; i++ {
		n := startANetwork()
		if i == 1 {
			p2p.DebugServer(n)
		}
		networks = append(networks, n)
		apps = append(apps, NewSimulApp(i, n))
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
				id := i + 1
				version := versionFromID(uint32(id))
				rw.Write([]byte(fmt.Sprintf("\n\n==============================\n\tNetwork %d (Prot %d)\n==============================\n", i+1, version)))
			}
			a, cc := c.DebugMessage()
			count += cc
			rw.Write([]byte(fmt.Sprintf("%v", apps[i].seen)))
			rw.Write([]byte(a))
		}
	})

	mux.HandleFunc("/halfviz", func(rw http.ResponseWriter, req *http.Request) {
		count = 0
		for _, c := range networks {
			b := c.DebugHalfviz()
			rw.Write([]byte(b))
		}

		rw.Write([]byte("\n127.0.0.1:8888 {color: red}\n127.0.0.2:8888 {color: green}\n127.0.0.3:8888 {color: blue}"))
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
		n := startANetwork()
		networks = append(networks, n)
		apps = append(apps, NewSimulApp(int(newnet), n))
	})

	c := make(chan os.Signal, 3)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	shutdown := func() {
		for i, n := range networks {
			n.Stop()
			apps[i].Stop()
		}
		fmt.Println("All networks and apps shut down")
	}

	go func() {
		<-c
		fmt.Println("Received Ctrl+C, initiating shutdown")
		go func() {
			<-c
			os.Exit(1)
		}()
		shutdown()
		<-time.After(time.Second * 10)
		pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		os.Exit(1)
	}()

	select {}
}
