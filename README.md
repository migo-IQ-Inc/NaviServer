# NaviServer
Navisens Golang UDP Server

Install golang on server, then run this to build and run server

```$ go build -o server . && ./server```

Or just run the executable in a screen for deploying server.

Add the environment variable `GOOS=linux` to build for (for example) linux.

## Server

`server.config` stores the tiny configuration necessary for server, in form of strict JSON.

```
{
    "serverHost":                      "0.0.0.0",
    "serverPort":                      8080,
    "maxNumberOfServerRooms":          100,
    "maxNumberOfConnectionsPerServer": 10,
    "connectionTimeoutInSeconds":      5
}
```

`serverHost` string: is the ip of the server host. The default uses the current local ip.
`serverPort` integer: specifies the port to host the server on.
`maxNumberOfServerRooms` integer: limits the number of server rooms to host
`maxNumberOfConnectionsPerServer` integer: limits the number of users on the same server. This number does not depend on server capabilities, but rather should reflect how connections will be used on the client side, because if too many users are being served on the same server, the client may become overloaded and unusable due to consuming all available bandwidth and/or computing power.
`connectionTimeoutInSeconds` integer: the timeout before removing a connection from a server room. For example, when a connection switches rooms, the server will only check for connection timeout periodically, and on the second timeout check, if the connection has not responded to a particular server room, it will be removed from that room (but not the new room it switched to). Servers have twice the timeout length, and are only deleted when no connections remain.

## SDK Usage

Additional features are provided for starting a server, moving rooms, sending packets, and interpreting queries.

**Starting a server**

`startUDP()`
`startUDP(room)`
`startUDP(host, port)`
`startUDP(room, host, port)`

Any unspecified fields are set to default values. Note that the default host/port currently only works for local Navisens employees.

**Changing server room**

`setUDPRoom(room)`

Room names are always strings.

**Sending packets**

`sendUDPPacket(string)` - sends a udp packet which is encrypted and sent securely to server. Server will not be able to decrypt packet. Only receiving client will be able to read packet.
`sendUDPQueryRooms(array of strings)` - sends a request to server to get the current number of active connections in each server room. (see handling below on how to interpret feedback). Note that it was designed purposefully that the client cannot get a list of servers. The server does NOT know the names of any servers, as all names are hashed! It is the developer's responsibility to code in any server rooms needed, and for all clients to acknowledge room names appropriately. This is for privacy concerns, as multiple different developers can host rooms on the same server, but only the developer (or users) who started a room should know of its existence. Privacy through obscurity.

**Receiving callbacks**

To receive callbacks, implement the following two methods in `MotionDnaSDK`

`receiveNetworkData(MotionDna)` is the new callback for receiving motionDna from network users. It has its own queue, and is processed separately from the internal motionDna.
`receiveNetworkData(NetworkCode, map)` serves all non-motiondna types. Use NetworkCode as a switch

NetworkCode values:

`0: RAW_NETWORK_DATA`
* map holds exactly two values:
* `{"ID": string}` is a 64-byte key whose prefix matches/contains the prefix of the deviceID (whichever is shorter).
* `{"payload": string}` the payload that was send using an above call of `sendUDPPacket`.

`1: ROOM_CAPACITY_STATUS`
* response from a `sendUDPQueryRooms` call
* map holds multiple values where the key is the room name requested, and the value is the number of current active connections in the room (including current device)

`2: EXCEEDED_ROOM_CONNECTION_CAPACITY`
* error code if attempting to broadcast to a room which already exists, but currently has reached its max capacity and cannot create a new connection
* map is null

`3: EXCEEDED_SERVER_ROOM_CAPACITY`
* error code if attempting to broadcast to a room which does not exist, but the server has reached the max number of rooms and cannot create a new room
* maps is null

**TODO**

* Implement a quiet mode for server
* Throttle send-packet speeds to something reasonable
* Optionally shut down / suspend / ban an app from sending packets if it continues to spam
* Better server logs
* Save server status rooms etc. and reload
* Implement server hotswap for restarting server without downtime
