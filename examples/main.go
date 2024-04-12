package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/tiechui1994/proxy/adapter"
	"github.com/tiechui1994/proxy/constant"
	"gopkg.in/yaml.v3"
)

func urlToMetadata(rawURL string) (addr constant.Metadata, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return
	}

	port := u.Port()
	if port == "" {
		switch u.Scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		default:
			err = fmt.Errorf("%s scheme not Support", rawURL)
			return
		}
	}

	p, _ := strconv.ParseUint(port, 10, 16)

	addr = constant.Metadata{
		Host:    u.Hostname(),
		DstIP:   nil,
		DstPort: constant.Port(p),
	}
	return
}

type RawConfig struct {
	Proxy []map[string]interface{} `yaml:"proxies"`
}

type Proxies map[string]constant.ProxyAdapter

func Parse(buf []byte) (Proxies, error) {
	rawCfg := &RawConfig{
		Proxy: make([]map[string]interface{}, 0),
	}
	if err := yaml.Unmarshal(buf, rawCfg); err != nil {
		return nil, err
	}

	proxies := make(Proxies)
	proxiesConfig := rawCfg.Proxy
	for idx, mapping := range proxiesConfig {
		proxy, err := adapter.ParseProxy(mapping)
		if err != nil {
			return nil, fmt.Errorf("proxy %d: %w", idx, err)
		}
		if _, exist := proxies[proxy.Name()]; exist {
			return nil, fmt.Errorf("proxy %s is the duplicate name", proxy.Name())
		}

		proxies[proxy.Name()] = proxy
	}

	return proxies, nil
}

func main() {
	raw, err := ioutil.ReadFile("/home/user/workspace/proxy/examples/hello.yaml")
	if err != nil {
		log.Printf("ReadFile:%v", err)
		return
	}
	proxies, err := Parse(raw)
	if err != nil {
		log.Printf("Parse:%v", err)
		return
	}

	test := func(proxy constant.ProxyAdapter) {
		reqURL := "https://api6.ipify.org?format=json"
		addr, err := urlToMetadata(reqURL)
		if err != nil {
			log.Printf("urlToMetadata:%v", err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(7000))
		defer cancel()

		start := time.Now()
		instance, err := proxy.DialContext(ctx, &addr)
		if err != nil {
			log.Printf("DialContext:%v", err)
			return
		}
		defer instance.Close()

		req, err := http.NewRequest(http.MethodGet, reqURL, nil)
		if err != nil {
			return
		}
		req = req.WithContext(ctx)

		transport := &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return instance, nil
			},
			// from http.DefaultTransport
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}

		client := http.Client{
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		defer client.CloseIdleConnections()

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("%v client.Do:%v", proxy.Name(), err)
			return
		}
		raw, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		log.Printf("%v total: %v, data: %v", proxy.Name(), time.Since(start), string(raw))
	}

	for _, proxy := range proxies {
		test(proxy)
	}
}
