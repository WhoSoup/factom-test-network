package main

import "fmt"

type IPGenerator struct {
	id uint32
	ip []byte
}

func NewIPGenerator() *IPGenerator {
	ig := new(IPGenerator)
	ig.id = 0
	ig.ip = []byte{0, 0, 0}
	return ig
}

func (ig *IPGenerator) Next() (uint32, string) {
	for i := 2; i >= 0; i-- {
		ig.ip[i]++
		if ig.ip[i] == 255 {
			ig.ip[i] = 1
		} else {
			break
		}
	}
	ig.id++

	return ig.id, fmt.Sprintf("127.%d.%d.%d", ig.ip[0], ig.ip[1], ig.ip[2])
}
