package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/miekg/dns"
)

const (
	endpoint  = "hello"
	hostname  = "hello.service.consul"
	consulDNS = "127.0.0.1:8600"
	interval  = 2 * time.Second
)

func main() {
	var (
		loop = flag.Bool("loop", true, "Make continuous requests to hello service.")
	)
	flag.Parse()

	ticker := time.NewTicker(interval)

	for {
		if err := requestHello(); err != nil {
			log.Printf("[ERR] failed to dial hello service: %v", err)
		}
		if !*loop {
			// Only run once if not looping
			break
		}
		<-ticker.C
	}
}

func requestHello() error {
	// Resolve address with Consul's DNS
	addr, err := resolveAddr()
	if err != nil {
		return fmt.Errorf("failed to resolve addr: %v", err)
	}

	// Use result to query Hello service
	// TODO: Update port
	target := fmt.Sprintf("http://%s:8080/%s", addr, endpoint)
	resp, err := http.Get(target)
	if err != nil {
		return fmt.Errorf("failed to get '%s': %v", target, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %v", err)
	}

	log.Println(fmt.Sprintf("%s says: %s", target, body))
	return nil
}

func resolveAddr() (string, error) {
	var c dns.Client
	var m dns.Msg

	m.SetQuestion(hostname+".", dns.TypeA)
	r, _, err := c.Exchange(&m, consulDNS)
	if err != nil {
		log.Fatal(err)
	}
	if len(r.Answer) == 0 {
		return "", fmt.Errorf("no results")
	}

	var Arecord *dns.A
	for _, ans := range r.Answer {
		Arecord = ans.(*dns.A)
		break
	}
	return Arecord.A.String(), nil
}
