package main

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	networkMotionDna  = 0
	networkRawPayload = 1
	networkQueryRooms = 2
	networkRoomFull   = 3
	networkServerFull = 4
)

type packet struct {
	addr    *net.UDPAddr
	payload *[]byte
}

type connection struct {
	keepAlive int64
	addr      *net.UDPAddr
	room      *serverRoom
}

type serverRoom struct {
	lock        sync.RWMutex
	capacity    int
	connections map[string]*connection
}

type serverChan struct {
	ch   chan *packet
	done chan bool
}

var numberOfServersAlreadyWarned = false
var serverCapacityAlreadyWarned = false // change to save state per server

var connections = make(map[string]*connection, int(0.5*float64(config.NumberOfServers*config.ServerCapacity)))
var connectionsRWLock sync.RWMutex

var servers = make(map[string]*serverChan, config.NumberOfServers)
var serverRooms = make(map[string]*serverRoom, config.NumberOfServers)
var serverRWLock sync.RWMutex

// Refresh ...
// Release any servers or connections that have timed out
func Refresh() {
	currentTime := time.Now().Unix()
	for servRm, server := range serverRooms {
		if len(server.connections) == 0 {
			Debug("[INFO]  <DEL> Server (%d/%d) %x is empty and will be released\n", len(servers), config.NumberOfServers, servRm)
			numberOfServersAlreadyWarned = false
			servers[servRm].done <- true
			serverRWLock.Lock()
			delete(servers, servRm)
			delete(serverRooms, servRm)
			serverRWLock.Unlock()
		} else {
			for key, conn := range server.connections {
				if conn.keepAlive+int64(config.Timeout) < currentTime {
					Debug("[INFO]  <DEL> Connection (%d/%d) from %v timed out\n", len(server.connections), config.ServerCapacity, conn.addr)
					serverCapacityAlreadyWarned = false
					connectionsRWLock.Lock()
					delete(connections, key)
					connectionsRWLock.Unlock()
					server.lock.Lock()
					delete(server.connections, key)
					server.lock.Unlock()
				}
			}
		}
	}
}

// Update ...
// Update and distribute a new connection across the network
func Update(conn *net.UDPConn, room string, payload []byte, addr *net.UDPAddr) {
	// Debug("Updating %x from %v\n", room, addr)
	serverRWLock.RLock()
	serv, ok := servers[room]
	servCount := len(servers)
	serverRWLock.RUnlock()

	if !ok {
		serverRWLock.Lock()
		serv, ok = servers[room] // securely create lock
		if !ok {                 // check if change occured between RUnlock and Lock calls
			if servCount >= config.NumberOfServers {
				if !numberOfServersAlreadyWarned {
					numberOfServersAlreadyWarned = true
					Debug("[WARN]  Server limit reached at %d\n", config.NumberOfServers)
				}
				conn.WriteToUDP([]byte{networkServerFull}, addr)
				return
			}
			Debug("[INFO]  <NEW> Making new server (%d/%d) with key %x\n", servCount+1, config.NumberOfServers, room)
			servRm, servCh := makeServer(conn, config.ServerCapacity, room)
			serv = &servCh

			servers[room] = serv
			serverRooms[room] = servRm
		}
		serverRWLock.Unlock()
	}
	serv.ch <- &packet{addr, &payload}
}

func makeServer(conn *net.UDPConn, cap int, roomName string) (*serverRoom, serverChan) {
	server := &serverRoom{
		capacity:    cap,
		connections: make(map[string]*connection, cap),
	}
	ch := make(chan *packet, cap)
	done := make(chan bool)
	go server.broadcast(conn, ch, done, roomName)
	return server, serverChan{ch, done}
}

func getRoomCapacity(rooms []byte) []byte {
	index, length := 0, len(rooms)
	var output bytes.Buffer
	for index+32 <= length {
		key := rooms[index : index+32]
		output.Write(key)
		count := 0
		if server, ok := serverRooms[string(key)]; ok {
			server.lock.RLock()
			count = len(server.connections)
			server.lock.RUnlock()
		}
		output.WriteString(fmt.Sprintf("%08d", count)[:8])
		index += 32
	}
	return output.Bytes()
}

func (s *serverRoom) broadcast(server *net.UDPConn, ch chan *packet, done chan bool, roomName string) {
	for {
		select {
		case p := <-ch:
			source := p.addr.String()
			// Debug("Received packet from %s\n", source)

			s.lock.RLock()
			pconn, ok := s.connections[source] // packet's connection
			s.lock.RUnlock()

			if !ok {
				if len(s.connections) >= config.ServerCapacity {
					if !serverCapacityAlreadyWarned {
						serverCapacityAlreadyWarned = true
						Debug("[WARN]  Server %x capacity reached at %d\n", roomName, config.ServerCapacity)
					}
					server.WriteToUDP([]byte{networkRoomFull}, p.addr)
					continue
				}
				Debug("[INFO]  <NEW> Creating new connection (%d/%d) %v on server %x\n", len(s.connections)+1, config.ServerCapacity, source, roomName)
				s.lock.Lock()
				pconn = &connection{
					addr: p.addr,
					room: s,
				}
				s.connections[source] = pconn
				s.lock.Unlock()

				connectionsRWLock.RLock()
				lconn, lok := connections[source] // last connection
				if lok {
					serverCapacityAlreadyWarned = false
					lconn.room.lock.Lock()
					delete(lconn.room.connections, source)
					lconn.room.lock.Unlock()
				}
				connectionsRWLock.RUnlock()
			}
			connectionsRWLock.Lock()
			connections[source] = pconn
			connectionsRWLock.Unlock()

			pconn.keepAlive = time.Now().Unix()

			// fmt.Println("Broadcasting packets")
			opcode := (*p.payload)[0]
			payload := (*p.payload)[1:]
			switch {
			case opcode == networkMotionDna || opcode == networkRawPayload:
				s.lock.RLock()
				for _, conn := range s.connections {
					if pconn != conn {
						output := append([]byte{opcode}[:1], payload...)
						server.WriteToUDP(output, conn.addr)
					}
				}
				s.lock.RUnlock()
			case opcode == networkQueryRooms:
				output := append([]byte{opcode}[:1], getRoomCapacity(payload)...)
				server.WriteToUDP(output, pconn.addr)
			}
		case <-done:
			return
		}
	}
}
