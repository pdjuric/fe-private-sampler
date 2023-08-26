package main

import (
	. "fe/common"
	. "fe/server"
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
		Scheme: "http",
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
