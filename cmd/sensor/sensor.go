package main

import (
	. "fe/common"
	. "fe/sensor"
	"fmt"
	"net"
	"os"
)

func SensorMain() {
	// init server
	args := os.Args[1:]
	argsCnt := len(args)
	defaultArgs := []string{GetIPv4().String(), "8081", "http"}

	if argsCnt == 1 || argsCnt > 3 {
		panic("arg error")
	}

	if len(args) != 3 {
		args = append(args, defaultArgs[argsCnt:]...)
	}

	ipv4 := net.ParseIP(args[0])
	if ipv4 == nil {
		fmt.Errorf("not a valid ipv4")
		return
	}

	fmt.Printf("sensor started with args: %s \n", args)

	ip := IP{
		Schema: args[2],
		IPv4:   net.ParseIP(args[0]),
		Port:   args[1],
	}

	GobInit()

	sensor := InitSensor()
	sensor.StartTaskDaemon(StartTaskWorker)
	sensor.RunHttpServer(ip)
}

func main() {
	SensorMain()
}
