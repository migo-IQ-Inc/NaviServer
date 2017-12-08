package main

import (
	"fmt"
	"net"
	"time"
)

// type addressRoom map[string]*net.UDPAddr
// var addresses = make(map[string]addressRoom, 10)

var config Config

func startServer(packetChan chan int, byteChan chan int64) {
	// Construct UDP address
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", config.Host, config.Port))
	if didError(err, "Could not resolve UDP address "+fmt.Sprintf("%s:%d", config.Host, config.Port)) {
		return
	}

	// Listen on UDP port
	Debug("[INFO]  Listening on %s\n", addr.String())
	conn, err := net.ListenUDP("udp", addr)
	if didError(err, "Could not listen on port") {
		return
	}
	defer conn.Close()

	packetCount, byteCount := 0, int64(0)

	var n int
	var remoteAddr *net.UDPAddr
	buffer := make([]byte, 2048)
	for {
		// Read packet
		n, remoteAddr, err = conn.ReadFromUDP(buffer)
		// fmt.Printf("Read %d bytes\n", n)
		if didError(err, "Error reading packet") {
			continue
		}
		// Debug(" -> Received: '%x'\n", string(buffer[:32]))

		/*
			addrRoom, ok := addresses[string(buffer[:32])]
			if !ok {
				addrRoom = make(map[string]*net.UDPAddr, 10)
				addresses[string(buffer[:32])] = addrRoom
			}

			_, ok = addrRoom[remoteAddr.String()]
			if !ok {
				addrRoom[remoteAddr.String()] = remoteAddr
			}

			// Broadcast packet
			///*
				for k, v := range addrRoom {
					if k == remoteAddr.String() {
						continue
					}
					go sendPacket(string(buffer[32:n]), conn, v)
				}
				/**/
		payload := []byte{}
		go Update(conn, string(buffer[:32]), append(payload, buffer[32:n]...), remoteAddr)

		packetCount++
		byteCount += int64(n)

		packetChan <- packetCount
		byteChan <- byteCount
	}
}

func sendPacket(msg string, conn *net.UDPConn, addr *net.UDPAddr) {
	n, err := conn.WriteToUDP([]byte(msg), addr)
	if !didError(err, "Could not send packet") {
		Debug(" <- Response: '%s' ... Wrote %d bytes to %v\n", "binary" /*msg*/, n, addr)
	}
}

func didError(err error, msg string) bool {
	if err != nil {
		Debug("[ERROR] %s\n%v\n", msg, err)
		return true
	}
	return false
}

func bytesPerConn(bytes int64, conns int) int64 {
	if conns == 0 {
		return 0
	}
	return int64(float64(bytes)/float64(conns) + 0.5)
}

func reportConnections(pch chan int, bch chan int64) {
	ready, refresh := make(chan bool), make(chan bool)
	go func() {
		rounds := 0
		for range time.Tick(5 * time.Second) {
			refresh <- true
			rounds++
			if rounds >= 12 {
				rounds = 0
				ready <- true
			}
		}
	}()
	reportedZero := false
	pnewest, platest, bnewest, blatest := 0, 0, int64(0), int64(0)
	for {
		select {
		case <-ready:
			if pnewest == platest && bnewest == blatest {
				if reportedZero {
					continue
				}
				reportedZero = true
			}
			Debug("[LOG]   %7d conns/min., %12d total conns; %11d bytes/min., %16d total bytes; %4d bytes/conn\n",
				pnewest-platest, pnewest, bnewest-blatest, bnewest, bytesPerConn(bnewest-blatest, pnewest-platest))
			platest, blatest = pnewest, bnewest
		case <-refresh:
			go Refresh()
		case pnewest = <-pch:
		case bnewest = <-bch:
		}
	}
}

// Debug ...
// printf but with time tag
func Debug(format string, a ...interface{}) {
	a = append([]interface{}{time.Now().Format(time.RFC3339)}, a...)
	fmt.Printf("[%s] | "+format, a...)
}

func main() {
	config = LoadConfig()
	Debug("[INFO]  Using configuration: %+v\n", config)
	pch, bch := make(chan int), make(chan int64)
	go reportConnections(pch, bch)
	startServer(pch, bch)
}
