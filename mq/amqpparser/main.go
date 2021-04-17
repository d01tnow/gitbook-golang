package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"context"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"

	"amqp.d01t.now/amqp"
)

func handleError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
	}
}

var traceLogEnabled bool
var currentPacketTimestamp time.Time
var currentPcap chan string

func handleMsg(ctx context.Context, chMsg chan *amqp.WrappedMessage) {
	pcap := ""
	for {
		select {
		case cpap := <-currentPcap:
			pcap = cpap
		case msg := <-chMsg:
			var mymsg MyMessage
			err := mymsg.Unmarshal(msg.Body)
			if err != nil {
				fmt.Println("unable to unmarsh, err:", err)
				continue
			}
			fmt.Printf(`pcap: %s
      serverIP: %s, serverPort: %s, clientIP: %s, clientPort: %s
      raw message: %s,
      message.id: %d, message.start: %s
      packet.Timestamp: %s
      `,
				pcap,
				msg.ServerEP.Host,
				msg.ServerEP.Port,
				msg.ClientEP.Host,
				msg.ClientEP.Port,
				hex.EncodeToString(msg.Body),
				mymsg.id,
				mymsg.start.Format(time.RFC3339Nano),
				currentPacketTimestamp.Format(time.RFC3339Nano),
			)
		case <-ctx.Done():
			return
		}
	}
}

func printPacketInfo(packet gopacket.Packet, parser amqp.AppPayloadParser) {
	var (
		client amqp.ClientEP
		server amqp.ServerEP
	)
	setHost := func(ip *layers.IPv4) {
		if isServerHost(ip.SrcIP.String()) {
			server.Host = ip.SrcIP.String()
			client.Host = ip.DstIP.String()
		} else {
			server.Host = ip.DstIP.String()
			client.Host = ip.SrcIP.String()
		}
		// fmt.Printf("ip.SrcIP: %s, ip.DstIP: %s, client: %v, server: %v\n", ip.SrcIP, ip.DstIP, client, server)
	}
	setIP := func(tcp *layers.TCP) {
		// tcp.SrcPort.String() 会返回 port(常见协议名), 比如: 5672(amqp)
		srcPort := fmt.Sprintf("%d", tcp.SrcPort)
		dstPort := fmt.Sprintf("%d", tcp.DstPort)
		if isServerPort(srcPort) {
			server.Port = srcPort
			client.Port = dstPort
		} else {
			server.Port = dstPort
			client.Port = srcPort
		}

		// fmt.Printf("tcp.srcPort: %s, tcp.dstPort: %s, client: %v, server: %v\n", tcp.SrcPort, tcp.DstPort, client, server)
	}
	currentPacketTimestamp = packet.Metadata().Timestamp
	var traceLog bool
	if packet.ApplicationLayer() != nil && traceLogEnabled {
		traceLog = true
		fmt.Println("---------", packet.Metadata().Timestamp)
	}
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethLayer != nil {
		if traceLog {
			fmt.Println("ethernet layer detected")
			fmt.Println()
		}
	}
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		setHost(ip)
		if traceLog {
			fmt.Println("ipv4 layer detected")
			fmt.Printf("from %s to %s\n", ip.SrcIP, ip.DstIP)
			fmt.Println("protocol:", ip.Protocol)
			fmt.Println()
		}
	}
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		setIP(tcp)
		if traceLog {
			fmt.Println("TCP layer detected")
			fmt.Printf("from port: %d, to port: %d\n", tcp.SrcPort, tcp.DstPort)
			fmt.Println()
		}
	}
	appLayer := packet.ApplicationLayer()
	if appLayer != nil {
		r := bytes.NewReader(appLayer.Payload())
		parser.Parse(r, &server, &client, packet.Metadata().Timestamp)
		if traceLog {
			fmt.Println("+++++++++++++++++++++++++", packet.Metadata().Timestamp)
			fmt.Println("application layer/Payload found")
			fmt.Printf("server: %v, client: %v\n", server, client)
			fmt.Println("+_+_+_+_+_+_+_")
			fmt.Println()
		}
	}
	if err := packet.ErrorLayer(); err != nil {
		fmt.Println("error decoding some part of packet:", err)
		fmt.Println()
	}
	if traceLog {
		fmt.Println("================================")
		fmt.Println()
	}
}

func isServerHost(host string) bool {
	return host == "192.168.50.38"
}
func isServerPort(port string) bool {
	return port == "5672"
}

func parsePcap(ctx context.Context, file string, chMsg chan<- *amqp.WrappedMessage) {
	if file == "" {
		return
	}
	currentPcap <- file
	handle, err := pcap.OpenOffline(file)
	handleError(err, "open offline failed")
	packageSource := gopacket.NewPacketSource(handle, handle.LinkType())
	ap := amqp.NewParser(chMsg)
	for packet := range packageSource.Packets() {
		printPacketInfo(packet, ap)
	}
}

var (
	clientPcap string
	serverPcap string
)

func init() {
	flag.StringVar(&clientPcap, "c", "", "client pcap file")
	flag.StringVar(&serverPcap, "s", "", "server pcap file")
}

func main() {
	flag.Parse()
	fmt.Println(clientPcap, serverPcap)
	ctx, cancel := context.WithCancel(context.Background())
	chMsg := make(chan *amqp.WrappedMessage, 1)
	go handleMsg(ctx, chMsg)
	if clientPcap != "" {
		clientPcap, err := filepath.Abs(clientPcap)
		if err != nil {
			log.Fatal(err)
		}
		parsePcap(ctx, clientPcap, chMsg)

	}
	if serverPcap != "" {
		serverPcap, err := filepath.Abs(serverPcap)
		if err != nil {
			log.Fatal(err)
		}
		parsePcap(ctx, serverPcap, chMsg)

	}
	sg := make(chan os.Signal, 1)
	signal.Notify(sg, os.Interrupt)
	<-sg
	cancel()
}
