package common

import (
	"fmt"
	"github.com/fentec-project/gofe/data"
	"math/big"
	"net"
	"time"
)

func GetIPv4() net.IP {
	//conn, err := net.Dial("udp", "8.8.8.8:80")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//defer conn.Close()

	//localAddr := conn.LocalAddr().(*net.UDPAddr)

	//return localAddr.IP
	return net.IPv4(127, 0, 0, 1)
}

func Now() time.Time {
	return time.Now().Truncate(time.Second)
}

func GetIntPtrFromDuration(d *time.Duration) *int64 {
	if d == nil {
		return nil
	}

	decryptionTime := d.Nanoseconds()
	return &decryptionTime
}

func NewMatrix(rows int, ints []int, repeat int) (data.Matrix, error) {
	vecLen := len(ints) / (rows / repeat)

	if len(ints)%(rows/repeat) != 0 {
		return nil, fmt.Errorf("unable to create matrix, invalid number of elements")
	}

	matrix := make([]data.Vector, rows)

	for i := 0; i < vecLen*rows; i++ {
		if i%vecLen == 0 {
			matrix[i/vecLen] = make([]*big.Int, 0)
		}
		matrix[i/vecLen] = append(matrix[i/vecLen], big.NewInt(int64(ints[i%len(ints)])))
	}
	return matrix, nil
}
