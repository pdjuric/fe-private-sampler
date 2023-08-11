package main

import (
	. "fe/internal/common"
	. "fe/internal/server"
)

func main() {
	// todo add args

	logger := GetLoggerForFile("main", "server")
	SetLogger(logger)

	server := InitServer()
	StartTaskDaemon(server.GetTaskChannel(), StartTaskWorker, logger) // todo accept

	ip := IP{
		Schema: "http",
		IPv4:   GetIPv4(),
		Port:   "8080",
	}

	server.HttpServer = InitHttpServer("server", ip, server.GetEndpoints())

	server.RunHttpServer()
}
