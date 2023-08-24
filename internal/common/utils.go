package common

import (
	"log"
	"net"
	"time"
)

func GetIPv4() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func Now() time.Time {
	return time.Now().Truncate(time.Second)
}

func GetMaxAndMinValue(array []int) int {
	max := array[0]
	for _, coefficient := range array {
		if coefficient > max {
			max = coefficient
		}
	}
	return max + 1
}
