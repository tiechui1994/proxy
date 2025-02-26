package main

import (
	"fmt"

	"github.com/tiechui1994/proxy/core"
)

func main() {
	node := map[string]interface{}{"name": "sss", "type": "vless", "server": "41.149.12.183", "network": "ws", "port": 8052, "region": "us",
		"servername": "ss.sss.ir", "skip-cert-verify": false, "tls": false, "udp": true, "use": true, "uuid": "53fa8faf-ba4b-4322-9c69-a3e5b1555049",
		"ws-opts": map[string]interface{}{"headers": map[string]interface{}{"Host": "redw.ssss.ir"}, "path": "/s.NET@@s@ss?ed=2560"}}

	fmt.Println(core.NodeTest(node, "https://api.quinn.eu.org", 5000))
}
