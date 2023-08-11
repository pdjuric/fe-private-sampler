package common

import (
	"fmt"
	"github.com/google/uuid"
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

func ParseUuid(uuidString string) (*uuid.UUID, error, error) {
	uuid, err := uuid.Parse(uuidString)
	if err != nil {
		return nil, err, fmt.Errorf("invalid uuid %s", uuidString)
	}

	return &uuid, nil, nil
}
