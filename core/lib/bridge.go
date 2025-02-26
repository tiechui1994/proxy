package main

/*
#include <stdlib.h>
typedef struct request {
	const char* url;
	const char* method;
	const char* header;
	const char* body;
} request;
*/
import "C"
import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	_ "unsafe"

	"github.com/metacubex/mihomo/adapter"
	"github.com/metacubex/mihomo/constant"
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
		DstPort: uint16(p),
	}

	return
}

func ParseQuery(str string) (url.Values, error) {
	return url.ParseQuery(str)
}

//export HttpTest
func HttpTest(node *C.char, request *C.struct_request, msTimeout C.int, delay *C.int, response **C.char, responseLen *C.int, errMessage **C.char, errLen *C.int) {
	val := C.GoString(node)

	var err error
	defer func() {
		if err != nil {
			msg := err.Error()
			*errMessage = C.CString(msg)
			*errLen = C.int(len(msg))
		}
	}()

	var mapping map[string]any
	err = json.Unmarshal([]byte(val), &mapping)
	if err != nil {
		return
	}

	proxy, err := adapter.ParseProxy(mapping)
	if err != nil {
		return
	}

	u := C.GoString(request.url)
	meta, err := urlToMetadata(u)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(msTimeout))
	defer cancel()

	start := time.Now()
	instance, err := proxy.DialContext(ctx, &meta)
	if err != nil {
		return
	}
	defer instance.Close()

	method := C.GoString(request.method)
	var body io.Reader
	if request.body != nil {
		body = strings.NewReader(C.GoString(request.body))
	}
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return
	}

	if request.header != nil {
		var pairs url.Values
		pairs, err = ParseQuery(C.GoString(request.header))
		if err != nil {
			return
		}
		for key, value := range pairs {
			req.Header.Add(key, value[0])
		}
	}

	req = req.WithContext(ctx)
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return instance, nil
		},
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
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
		return
	}
	defer resp.Body.Close()
	*delay = C.int(time.Since(start).Milliseconds())

	if response != nil {
		var raw []byte
		raw, err = io.ReadAll(resp.Body)
		if err != nil {
			return
		}
		*response = C.CString(string(raw))
		*responseLen = C.int(len(raw))
	}
}

func main() {}
