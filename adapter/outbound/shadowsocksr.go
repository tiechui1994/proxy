package outbound

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/tiechui1994/proxy/component/dialer"
	"github.com/tiechui1994/proxy/constant"
	"github.com/tiechui1994/proxy/transport/shadowsocks/core"
	"github.com/tiechui1994/proxy/transport/shadowsocks/shadowaead"
	"github.com/tiechui1994/proxy/transport/shadowsocks/shadowstream"
	"github.com/tiechui1994/proxy/transport/ssr/obfs"
	"github.com/tiechui1994/proxy/transport/ssr/protocol"
)

type ShadowSocksR struct {
	*Base
	cipher   core.Cipher
	obfs     obfs.Obfs
	protocol protocol.Protocol
}

type ShadowSocksROption struct {
	BasicOption
	Name          string `proxy:"name"`
	Server        string `proxy:"server"`
	Port          int    `proxy:"port"`
	Password      string `proxy:"password"`
	Cipher        string `proxy:"cipher"`
	Obfs          string `proxy:"obfs"`
	ObfsParam     string `proxy:"obfs-param,omitempty"`
	Protocol      string `proxy:"protocol"`
	ProtocolParam string `proxy:"protocol-param,omitempty"`
	UDP           bool   `proxy:"udp,omitempty"`
}

// StreamConn implements constant.ProxyAdapter
func (ssr *ShadowSocksR) StreamConn(c net.Conn, metadata *constant.Metadata) (net.Conn, error) {
	c = ssr.obfs.StreamConn(c)
	c = ssr.cipher.StreamConn(c)
	var (
		iv  []byte
		err error
	)
	switch conn := c.(type) {
	case *shadowstream.Conn:
		iv, err = conn.ObtainWriteIV()
		if err != nil {
			return nil, err
		}
	case *shadowaead.Conn:
		return nil, fmt.Errorf("invalid connection type")
	}
	c = ssr.protocol.StreamConn(c, iv)
	_, err = c.Write(serializesSocksAddr(metadata))
	return c, err
}

// DialContext implements constant.ProxyAdapter
func (ssr *ShadowSocksR) DialContext(ctx context.Context, metadata *constant.Metadata, opts ...dialer.Option) (_ constant.Conn, err error) {
	c, err := dialer.DialContext(ctx, "tcp", ssr.addr, ssr.Base.DialOptions(opts...)...)
	if err != nil {
		return nil, fmt.Errorf("%s connect error: %w", ssr.addr, err)
	}
	tcpKeepAlive(c)

	defer func(c net.Conn) {
		safeConnClose(c, err)
	}(c)

	c, err = ssr.StreamConn(c, metadata)
	return NewConn(c, ssr), err
}

// ListenPacketContext implements constant.ProxyAdapter
func (ssr *ShadowSocksR) ListenPacketContext(ctx context.Context, metadata *constant.Metadata, opts ...dialer.Option) (constant.PacketConn, error) {
	pc, err := dialer.ListenPacket(ctx, "udp", "", ssr.Base.DialOptions(opts...)...)
	if err != nil {
		return nil, err
	}

	addr, err := resolveUDPAddr("udp", ssr.addr)
	if err != nil {
		pc.Close()
		return nil, err
	}

	pc = ssr.cipher.PacketConn(pc)
	pc = ssr.protocol.PacketConn(pc)
	return newPacketConn(&ssPacketConn{PacketConn: pc, rAddr: addr}, ssr), nil
}

func NewShadowSocksR(option ShadowSocksROption) (*ShadowSocksR, error) {
	// SSR protocol compatibility
	// https://github.com/Dreamacro/clash/pull/2056
	if option.Cipher == "none" {
		option.Cipher = "dummy"
	}

	addr := net.JoinHostPort(option.Server, strconv.Itoa(option.Port))
	cipher := option.Cipher
	password := option.Password
	coreCiph, err := core.PickCipher(cipher, nil, password)
	if err != nil {
		return nil, fmt.Errorf("ssr %s initialize error: %w", addr, err)
	}
	var (
		ivSize int
		key    []byte
	)

	if option.Cipher == "dummy" {
		ivSize = 0
		key = core.Kdf(option.Password, 16)
	} else {
		ciph, ok := coreCiph.(*core.StreamCipher)
		if !ok {
			return nil, fmt.Errorf("%s is not none or a supported stream cipher in ssr", cipher)
		}
		ivSize = ciph.IVSize()
		key = ciph.Key
	}

	obfs, obfsOverhead, err := obfs.PickObfs(option.Obfs, &obfs.Base{
		Host:   option.Server,
		Port:   option.Port,
		Key:    key,
		IVSize: ivSize,
		Param:  option.ObfsParam,
	})
	if err != nil {
		return nil, fmt.Errorf("ssr %s initialize obfs error: %w", addr, err)
	}

	protocol, err := protocol.PickProtocol(option.Protocol, &protocol.Base{
		Key:      key,
		Overhead: obfsOverhead,
		Param:    option.ProtocolParam,
	})
	if err != nil {
		return nil, fmt.Errorf("ssr %s initialize protocol error: %w", addr, err)
	}

	return &ShadowSocksR{
		Base: &Base{
			name:  option.Name,
			addr:  addr,
			tp:    constant.ShadowsocksR,
			udp:   option.UDP,
			iface: option.Interface,
			rmark: option.RoutingMark,
		},
		cipher:   coreCiph,
		obfs:     obfs,
		protocol: protocol,
	}, nil
}
