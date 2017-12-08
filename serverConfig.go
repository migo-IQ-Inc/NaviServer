package main

import (
	"encoding/json"
	"os"
)

// Config ...
// Configuration details of server
type Config struct {
	Host            string `json:"serverHost"`
	Port            int    `json:"serverPort"`
	NumberOfServers int    `json:"maxNumberOfServerRooms"`
	ServerCapacity  int    `json:"maxNumberOfConnectionsPerServer"`
	Timeout         int    `json:"connectionTimeoutInSeconds"`
}

// LoadConfig ...
// loads a configuration file in the current directory
func LoadConfig() Config {
	file, _ := os.Open("server.config")
	decoder := json.NewDecoder(file)
	config := Config{}
	err := decoder.Decode(&config)
	if didError(err, "Invalid configuration file") {
		return Config{"", 8080, 100, 10, 5}
	}
	return config
}
