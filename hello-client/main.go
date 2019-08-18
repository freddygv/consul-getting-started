package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/miekg/dns"
)

const (
	endpoint  = "hello"
	hostname  = "hello.service.consul"
	interval  = 2 * time.Second
)

func main() {
	var (
		loop = flag.Bool("loop", true, "Make continuous requests to hello service.")
		consulDNS = flag.String("consul-dns", os.Getenv("NODE_IP") + ":8600", "Consul DNS server addr")
	)
	flag.Parse()

	ticker := time.NewTicker(interval)
	for {
		if err := requestHello(*consulDNS); err != nil {
			log.Printf("[ERR] failed to dial hello service: %v", err)
		}
		if !*loop {
			// Only run once if not looping
			break
		}
		<-ticker.C
	}
}

func requestHello(consulDNS string) error {
	// Resolve address with Consul's DNS
	addr, err := resolveAddr(consulDNS)
	if err != nil {
		return fmt.Errorf("failed to resolve addr: %v", err)
	}

	// Use result to query Hello service
	target := fmt.Sprintf("http://%s/%s", addr, endpoint)
	resp, err := http.Get(target)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %v", err)
	}

	log.Println(fmt.Sprintf("%s says: %s", target, body))
	return nil
}

func resolveAddr(srvAddr string) (string, error) {
	var c dns.Client
	var m dns.Msg

	m.SetQuestion(hostname+".", dns.TypeSRV)
	r, _, err := c.Exchange(&m, srvAddr)
	if err != nil {
		log.Fatal(err)
	}
	if len(r.Answer) == 0 {
		return "", fmt.Errorf("no results")
	}

	// Get port from SRV record in Answer
	var srv *dns.SRV
	var ok bool
	for _, ans := range r.Answer {
		srv, ok = ans.(*dns.SRV)
		if !ok {
			return "", fmt.Errorf("answer was not of type dns.SRV, got: %v", reflect.TypeOf(ans))
		}
		break
	}
	port := strconv.Itoa(int(srv.Port))

	// Get IP from A record in the Additional Section
	var a *dns.A
	for _, ans := range r.Extra {
		a, ok = ans.(*dns.A)
		if !ok {
			return "", fmt.Errorf("additional record was not of type dns.A, got: %v", reflect.TypeOf(ans))
		}
		break
	}
	addr := a.A.String()

	return net.JoinHostPort(addr, port), nil
}
