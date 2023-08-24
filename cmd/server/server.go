package main

import (
	. "fe/internal/common"
	. "fe/internal/server"
	"fmt"
)

func ServerMain() {
	// todo add args

	err := InitLogger("server")
	if err != nil {
		fmt.Println(err)
		return
	}

	ip := IP{
		Schema: "http",
		IPv4:   GetIPv4(),
		Port:   "8080",
	}

	GobInit()

	server := InitServer()
	server.StartTaskDaemon(StartTaskWorker)
	server.RunHttpServer(ip)
}

func main() {
	ServerMain()
}
