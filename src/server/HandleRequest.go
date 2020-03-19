package server

import (
	"encoding/binary"
	"net"
	"time"
	"ChannelList"
	"Utilization"
	_"sync"
)

var closeTCP = false

func allZero(s []byte) bool {

	defer ChannelList.Recover()
	
	for _, v := range s {
		if v != 0 {
			return false
		}
	}
	return true
}

func HandleRequest(conn net.TCPConn) {
	
	defer ChannelList.Recover()

	defer conn.Close()

	parseChan := make(chan bool, 1)

	var counterRequest = 0

	for {

		if closeTCP{
			go ChannelList.WriteLog("Closing all current sockets...")
			conn.Close()
			break
		}
		
		sizeBuf := make([]byte, 4)

		conn.Read(sizeBuf)

		packetSize := binary.LittleEndian.Uint32(sizeBuf)

		if packetSize < 0 {
			continue
		}

		completePacket := make([]byte, packetSize)

		conn.Read(completePacket)

		if allZero(completePacket) {
			go ChannelList.WriteLog("Connection closed...")
			break
		}

		var message = string(completePacket)

		go ParseMsg(message, conn, parseChan, &counterRequest)

		select {

			case _, ok := <-parseChan:	

				if ok{

				}
		}
	}
}

func ShowUtilization(){
	for{

		if !closeTCP{
			time.Sleep(5 * time.Second)
			Utilization.GetHardwareData()
		}else{
			break
		}

	}
}

func CloseTCPServers(){
	
	defer ChannelList.Recover()

	ChannelList.WriteLog("Closing tcp socket...")

	closeTCP = true
}