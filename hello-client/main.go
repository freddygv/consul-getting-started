package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/miekg/dns"
)

const (
	endpoint  = "hello"
	hostname  = "hello.service.consul"
	consulDNS = "127.0.0.1:8600"
)

func main() {
	flag.Parse()

	if err := requestHello(); err != nil {
		log.Fatalf("[ERR] failed to dial hello service: %v", err)
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

	log.Println(string(body))
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
