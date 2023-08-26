package main

import (
	. "fe/authority"
	. "fe/common"
	"fmt"
)

func AuthorityMain() {
	// todo add args

	err := InitLogger("authority")
	if err != nil {
		fmt.Println(err)
		return
	}

	ip := IP{
		Scheme: "http",
		IPv4:   GetIPv4(),
		Port:   "8082",
	}

	GobInit()

	authority := InitAuthority()
	authority.StartTaskDaemon(StartTaskWorker)
	authority.RunHttpServer(ip)
}

func main() {
	AuthorityMain()
}
