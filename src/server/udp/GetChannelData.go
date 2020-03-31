package udp

import(
	"encoding/binary"
	"sync"
	_"log"
	"ChannelList"
	"time"
	"net"
	"os"
	"ByteBuffer"
	"pojo"
	"strconv"
	"io/ioutil"
)


var ChannelMethod = &ChannelMethods{}

type ChannelMethods struct{
	sync.Mutex
}


func checkCreateDirectory(conn net.UDPConn, packetObject pojo.UDPPacketStruct, checkDirectoryChan chan bool){

	defer ChannelList.Recover()

	var consumerName = packetObject.ChannelName + packetObject.SubscriberName

	var directoryPath = ChannelList.UDPStorage[packetObject.ChannelName].Path+"/"+consumerName

	if _, err := os.Stat(directoryPath); err == nil{

		checkDirectoryChan <- true

	}else if os.IsNotExist(err){

		errDir := os.MkdirAll(directoryPath, 0755)

		if errDir != nil {
			
			ThroughUDPClientError(conn, err.Error())

			DeleteUDPChannelSubscriberList(packetObject.ChannelName, consumerName)

			checkDirectoryChan <- false

			return

		}

		go ChannelList.WriteLog("Subscriber directory created successfully...")

		checkDirectoryChan <- true

	}else{

		checkDirectoryChan <- true

	}
}

func createSubscriberOffsetFile(index int, conn net.UDPConn, packetObject pojo.UDPPacketStruct, start_from string, partitionOffsetSubscriber chan int64){

	defer ChannelList.Recover()

	var consumerName = packetObject.ChannelName + packetObject.SubscriberName

	var directoryPath = ChannelList.UDPStorage[packetObject.ChannelName].Path+"/"+consumerName

	var consumerOffsetPath = directoryPath+"\\"+packetObject.SubscriberName+"_offset_"+strconv.Itoa(index)+".index"

	if _, err := os.Stat(consumerOffsetPath); err == nil{

		if start_from == "BEGINNING"{

			partitionOffsetSubscriber <- 0

		}else if start_from == "NOPULL"{

			partitionOffsetSubscriber <- (-1)

		}else if start_from == "LASTRECEIVED"{

			dat, err := ioutil.ReadFile(consumerOffsetPath)

			if err != nil{

				ThroughUDPClientError(conn, err.Error())

				DeleteUDPChannelSubscriberList(packetObject.ChannelName, consumerName)

				partitionOffsetSubscriber <- 0

				return

			}

			if len(dat) == 0{

				partitionOffsetSubscriber <- 0

			}else{

				partitionOffsetSubscriber <- int64(binary.BigEndian.Uint64(dat))

			}

		}else{

			partitionOffsetSubscriber <- 0

		}

		fDes, err := os.OpenFile(consumerOffsetPath,
			os.O_WRONLY, os.ModeAppend)

		if err != nil {

			ThroughUDPClientError(conn, err.Error())

			DeleteUDPChannelSubscriberList(packetObject.ChannelName, consumerName)

			partitionOffsetSubscriber <- 0

			return
		}

		AddSubscriberFD(index, packetObject, fDes)

	}else if os.IsNotExist(err){

		fDes, err := os.Create(consumerOffsetPath)

		if err != nil{

			ThroughUDPClientError(conn, err.Error())

			DeleteUDPChannelSubscriberList(packetObject.ChannelName, consumerName)

			partitionOffsetSubscriber <- 0

			return

		}

		AddSubscriberFD(index, packetObject, fDes)

		partitionOffsetSubscriber <- 0

	}else{

		if start_from == "BEGINNING"{

			partitionOffsetSubscriber <- 0

		}else if start_from == "NOPULL"{

			partitionOffsetSubscriber <- (-1)

		}else if start_from == "LASTRECEIVED"{

			dat, err := ioutil.ReadFile(consumerOffsetPath)

			if err != nil{

				ThroughUDPClientError(conn, err.Error())

				DeleteUDPChannelSubscriberList(packetObject.ChannelName, consumerName)

				partitionOffsetSubscriber <- 0

				return

			}

			if len(dat) == 0{

				partitionOffsetSubscriber <- 0

			}else{

				partitionOffsetSubscriber <- int64(binary.BigEndian.Uint64(dat))

			}

		}else{

			partitionOffsetSubscriber <- 0

		}

	}

}

func checkCreateGroupDirectory(channelName string, groupName string, checkDirectoryChan chan bool){

	defer ChannelList.Recover()

	var directoryPath = ChannelList.UDPStorage[channelName].Path+"/"+groupName

	if _, err := os.Stat(directoryPath); err == nil{

		checkDirectoryChan <- true

	}else if os.IsNotExist(err){

		errDir := os.MkdirAll(directoryPath, 0755)

		if errDir != nil {
			
			ThroughGroupError(channelName, groupName, err.Error())

			checkDirectoryChan <- false

			return

		}

		go ChannelList.WriteLog("Subscriber directory created successfully...")

		checkDirectoryChan <- true

	}else{

		checkDirectoryChan <- true

	}

}

func createSubscriberGroupOffsetFile(index int, channelName string, groupName string, packetObject pojo.UDPPacketStruct, partitionOffsetSubscriber chan int64, start_from string){

	defer ChannelList.Recover()

	var directoryPath = ChannelList.UDPStorage[channelName].Path+"/"+groupName

	var consumerOffsetPath = directoryPath+"\\"+groupName+"_offset_"+strconv.Itoa(index)+".index"

	if _, err := os.Stat(consumerOffsetPath); err == nil{

		if start_from == "BEGINNING"{

			partitionOffsetSubscriber <- 0

		}else if start_from == "NOPULL"{

			partitionOffsetSubscriber <- (-1)

		}else if start_from == "LASTRECEIVED"{

			dat, err := ioutil.ReadFile(consumerOffsetPath)

			if err != nil{

				ThroughGroupError(channelName, groupName, err.Error())

				partitionOffsetSubscriber <- 0

				return

			}

			if len(dat) == 0{

				partitionOffsetSubscriber <- 0

			}else{

				partitionOffsetSubscriber <- int64(binary.BigEndian.Uint64(dat))

			}

		}else{

			partitionOffsetSubscriber <- 0

		}

		fDes, err := os.OpenFile(consumerOffsetPath,
			os.O_WRONLY, os.ModeAppend) //race

		if err != nil {

			ThroughGroupError(channelName, groupName, err.Error())

			partitionOffsetSubscriber <- 0

			return
		}

		AddSubscriberFD(index, packetObject, fDes) // race

	}else if os.IsNotExist(err){

		fDes, err := os.Create(consumerOffsetPath)

		if err != nil{

			ThroughGroupError(channelName, groupName, err.Error())

			partitionOffsetSubscriber <- 0

			return

		}

		AddSubscriberFD(index, packetObject, fDes)

		partitionOffsetSubscriber <- 0

	}else{

		if start_from == "BEGINNING"{

			partitionOffsetSubscriber <- 0

		}else if start_from == "NOPULL"{

			partitionOffsetSubscriber <- (-1)

		}else if start_from == "LASTRECEIVED"{

			dat, err := ioutil.ReadFile(consumerOffsetPath)

			if err != nil{

				ThroughGroupError(channelName, groupName, err.Error())

				partitionOffsetSubscriber <- 0

				return

			}

			if len(dat) == 0{

				partitionOffsetSubscriber <- 0

			}else{

				partitionOffsetSubscriber <- int64(binary.BigEndian.Uint64(dat))

			}

		}else{

			partitionOffsetSubscriber <- 0

		}

	}

}

func SubscribeGroupChannel(channelName string, groupName string, packetObject pojo.UDPPacketStruct, start_from string){

	defer ChannelList.Recover()

	var checkDirectoryChan = make(chan bool, 1)

	var offsetByteSize = make([]int64, ChannelList.UDPStorage[packetObject.ChannelName].PartitionCount)

	var partitionOffsetSubscriber = make(chan int64, 1)

	go checkCreateGroupDirectory(channelName, groupName, checkDirectoryChan)

	if false == <-checkDirectoryChan{

		return

	}

	packetObject.SubscriberFD = CreateSubscriberGrpFD(packetObject.ChannelName)

	for i:=0;i<ChannelList.UDPStorage[packetObject.ChannelName].PartitionCount;i++{

		go createSubscriberGroupOffsetFile(i, channelName, groupName, packetObject, partitionOffsetSubscriber, start_from) // race

		offsetByteSize[i] = <-partitionOffsetSubscriber

	}

	if len(packetObject.SubscriberFD) == 0{

		ThroughGroupError(channelName, groupName, INVALID_SUBSCRIBER_OFFSET)

		return

	}

	for i:=0;i<ChannelList.UDPStorage[packetObject.ChannelName].PartitionCount;i++{

		go func(index int, cursor int64, packetObject pojo.UDPPacketStruct){

			defer ChannelList.Recover()

			var groupMtx sync.Mutex

			var filePath = ChannelList.UDPStorage[packetObject.ChannelName].Path+"/"+packetObject.ChannelName+"_partition_"+strconv.Itoa(index)+".br"

			file, err := os.Open(filePath)

			if err != nil {

				ThroughGroupError(packetObject.ChannelName, packetObject.GroupName, err.Error())

				return
			}

			defer file.Close()

			var sentMsg = make(chan bool, 1)

			defer close(sentMsg)

			var exitLoop = false

			for{

				if exitLoop{

					var consumerGroupLen = GetChannelGrpMapLen(packetObject.ChannelName, packetObject.GroupName)

					if consumerGroupLen <= 0{

						go CloseSubscriberGrpFD(packetObject)

					}

					break

				}

				fileStat, err := os.Stat(filePath)
		 
				if err != nil {

					ThroughGroupError(packetObject.ChannelName, packetObject.GroupName, err.Error())

					exitLoop = true
					
					continue
					
				}

				if cursor == -1{

					cursor = fileStat.Size()

				}

				if cursor < fileStat.Size(){

					data := make([]byte, 8)

					count, err := file.ReadAt(data, cursor)

					if err != nil {

						continue

					}

					if int64(count) > 0 && int64(count) < fileStat.Size(){

						cursor += 8

						var packetSize = binary.BigEndian.Uint64(data)

						restPacket := make([]byte, int64(packetSize))

						totalByteLen, errPacket := file.ReadAt(restPacket, cursor)

						if errPacket != nil{

							continue

						}

						if totalByteLen > 0{

							cursor += int64(packetSize)

							var byteFileBuffer = ByteBuffer.Buffer{
								Endian:"big",
							}

							byteFileBuffer.Wrap(restPacket)

							var messageTypeByte = byteFileBuffer.GetShort()
							var messageTypeLen = int(binary.BigEndian.Uint16(messageTypeByte))
							var messageType = byteFileBuffer.Get(messageTypeLen)

							var channelNameByte = byteFileBuffer.GetShort()
							var channelNameLen = int(binary.BigEndian.Uint16(channelNameByte))
							var channelName = byteFileBuffer.Get(channelNameLen)

							var producer_idByte = byteFileBuffer.GetShort()
							var producer_idLen = int(binary.BigEndian.Uint16(producer_idByte))
							var producer_id = byteFileBuffer.Get(producer_idLen)

							var agentNameByte  = byteFileBuffer.GetShort()
							var agentNameLen = int(binary.BigEndian.Uint16(agentNameByte))
							var agentName = byteFileBuffer.Get(agentNameLen)

							var idByte = byteFileBuffer.GetLong()
							var id = binary.BigEndian.Uint64(idByte)

							var bodyPacketSize = int64(packetSize) - int64(2 + messageTypeLen + 2 + channelNameLen + 2 + producer_idLen + 2 + agentNameLen + 8)

							var bodyBB = byteFileBuffer.Get(int(bodyPacketSize))

							var newTotalByteLen = 2 + messageTypeLen + 2 + channelNameLen + 2 + producer_idLen + 2 + agentNameLen + 8 + len(bodyBB)

							var byteSendBuffer = ByteBuffer.Buffer{
								Endian:"big",
							}

							byteSendBuffer.PutLong(newTotalByteLen) // total packet length

							byteSendBuffer.PutByte(byte(0)) // status code

							byteSendBuffer.PutShort(messageTypeLen) // total message type length

							byteSendBuffer.Put([]byte(messageType)) // message type value

							byteSendBuffer.PutShort(channelNameLen) // total channel name length

							byteSendBuffer.Put([]byte(channelName)) // channel name value

							byteSendBuffer.PutShort(producer_idLen) // producerid length

							byteSendBuffer.Put([]byte(producer_id)) // producerid value

							byteSendBuffer.PutShort(agentNameLen) // agentName length

							byteSendBuffer.Put([]byte(agentName)) // agentName value

							byteSendBuffer.PutLong(int(id)) // backend offset

							byteSendBuffer.Put(bodyBB) // actual body

							// log.Println(ChannelList.UDPStorage[packetObject.ChannelName][packetObject.GroupName])

							go sendGroup(index, groupMtx, int(cursor), packetObject, byteSendBuffer, sentMsg) // race

							message, ok := <-sentMsg

							if ok{

								if !message{

									exitLoop = true


								}else{

									exitLoop = false

								}
							}

						}else{

							time.Sleep(1 * time.Second)

						}

					}else{

						time.Sleep(1 * time.Second)
					}

				}else{

					cursor = fileStat.Size()

					time.Sleep(1 * time.Second)
				}

			}

		}(i, offsetByteSize[i], packetObject)

	}

}

func SubscribeChannel(conn net.UDPConn, packetObject pojo.UDPPacketStruct, start_from string){

	defer ChannelList.Recover()

	var consumerName = packetObject.ChannelName + packetObject.SubscriberName

	var checkDirectoryChan = make(chan bool, 1)

	var offsetByteSize = make([]int64, ChannelList.UDPStorage[packetObject.ChannelName].PartitionCount)

	var partitionOffsetSubscriber = make(chan int64, 1)

	go checkCreateDirectory(conn, packetObject, checkDirectoryChan)

	if false == <-checkDirectoryChan{

		return

	}

	packetObject.SubscriberFD = make([]*os.File, ChannelList.UDPStorage[packetObject.ChannelName].PartitionCount)

	for i:=0;i<ChannelList.UDPStorage[packetObject.ChannelName].PartitionCount;i++{

		go createSubscriberOffsetFile(i, conn, packetObject, start_from, partitionOffsetSubscriber)

		offsetByteSize[i] = <-partitionOffsetSubscriber

	}

	if len(packetObject.SubscriberFD) == 0{

		ThroughUDPClientError(conn, INVALID_SUBSCRIBER_OFFSET)

		DeleteUDPChannelSubscriberList(packetObject.ChannelName, consumerName)

		return

	}

	var subscriberMtx sync.Mutex

	for i:=0;i<ChannelList.UDPStorage[packetObject.ChannelName].PartitionCount;i++{

		go func(index int, cursor int64, conn net.UDPConn, packetObject pojo.UDPPacketStruct){

			defer ChannelList.Recover()

			var filePath = ChannelList.UDPStorage[packetObject.ChannelName].Path+"/"+packetObject.ChannelName+"_partition_"+strconv.Itoa(index)+".br"

			file, err := os.Open(filePath)

			defer file.Close()

			if err != nil {
				
				ThroughUDPClientError(conn, err.Error())

				DeleteUDPChannelSubscriberList(packetObject.ChannelName, consumerName)

				return
			}

			var sentMsg = make(chan bool, 1)

			defer close(sentMsg)

			var exitLoop = false

			for{

				if exitLoop{

					conn.Close()

					go CloseSubscriberGrpFD(packetObject)

					DeleteUDPChannelSubscriberList(packetObject.ChannelName, consumerName)

					break

				}

				fileStat, err := os.Stat(filePath)
		 
				if err != nil {

					ThroughUDPClientError(conn, err.Error())

					DeleteUDPChannelSubscriberList(packetObject.ChannelName, consumerName)

					break
					
				}

				if cursor == -1{

					cursor = fileStat.Size()

				}

				if cursor < fileStat.Size(){

					data := make([]byte, 8)

					count, err := file.ReadAt(data, cursor)

					if err != nil {

						continue

					}

					if int64(count) > 0 && int64(count) < fileStat.Size(){

						cursor += 8

						var packetSize = binary.BigEndian.Uint64(data)

						restPacket := make([]byte, packetSize)

						totalByteLen, errPacket := file.ReadAt(restPacket, cursor)

						if errPacket != nil{

							continue

						}

						if totalByteLen > 0{

							cursor += int64(packetSize)

							var byteFileBuffer = ByteBuffer.Buffer{
								Endian:"big",
							}

							byteFileBuffer.Wrap(restPacket)

							var messageTypeByte = byteFileBuffer.GetShort()
							var messageTypeLen = int(binary.BigEndian.Uint16(messageTypeByte))
							var messageType = byteFileBuffer.Get(messageTypeLen)

							var channelNameByte = byteFileBuffer.GetShort()
							var channelNameLen = int(binary.BigEndian.Uint16(channelNameByte))
							var channelName = byteFileBuffer.Get(channelNameLen)

							var producer_idByte = byteFileBuffer.GetShort()
							var producer_idLen = int(binary.BigEndian.Uint16(producer_idByte))
							var producer_id = byteFileBuffer.Get(producer_idLen)

							var agentNameByte  = byteFileBuffer.GetShort()
							var agentNameLen = int(binary.BigEndian.Uint16(agentNameByte))
							var agentName = byteFileBuffer.Get(agentNameLen)

							var idByte = byteFileBuffer.GetLong()
							var id = binary.BigEndian.Uint64(idByte)

							var bodyPacketSize = int64(packetSize) - int64(2 + messageTypeLen + 2 + channelNameLen + 2 + producer_idLen + 2 + agentNameLen + 8)

							var bodyBB = byteFileBuffer.Get(int(bodyPacketSize))

							var newTotalByteLen = 2 + messageTypeLen + 2 + channelNameLen + 2 + producer_idLen + 2 + agentNameLen + 8 + len(bodyBB)

							var byteSendBuffer = ByteBuffer.Buffer{
								Endian:"big",
							}

							byteSendBuffer.PutLong(newTotalByteLen) // total packet length

							byteSendBuffer.PutByte(byte(0)) // status code

							byteSendBuffer.PutShort(messageTypeLen) // total message type length

							byteSendBuffer.Put([]byte(messageType)) // message type value

							byteSendBuffer.PutShort(channelNameLen) // total channel name length

							byteSendBuffer.Put([]byte(channelName)) // channel name value

							byteSendBuffer.PutShort(producer_idLen) // producerid length

							byteSendBuffer.Put([]byte(producer_id)) // producerid value

							byteSendBuffer.PutShort(agentNameLen) // agentName length

							byteSendBuffer.Put([]byte(agentName)) // agentName value

							byteSendBuffer.PutLong(int(id)) // backend offset

							byteSendBuffer.Put(bodyBB) // actual body

							go send(index, int(cursor), subscriberMtx, packetObject, conn, byteSendBuffer, sentMsg)

							message, ok := <-sentMsg

							if ok{

								if !message{

									exitLoop = true

								}else{

									exitLoop = false

								}

							}

						}else{

							time.Sleep(1 * time.Second)

						}

					}else{

						time.Sleep(1 * time.Second)
					}

				}else{

					cursor = fileStat.Size()

					time.Sleep(1 * time.Second)
				}
			}


		}(i, offsetByteSize[i], conn, packetObject)

	}

}

func sendGroup(index int, groupMtx sync.Mutex, cursor int, packetObject pojo.UDPPacketStruct, packetBuffer ByteBuffer.Buffer, sentMsg chan bool){

	defer ChannelList.Recover()

	groupMtx.Lock()

	defer groupMtx.Unlock()

	var groupId = 0

	var group *pojo.UDPPacketStruct

	RETRY:

	group, groupId = GetValue(packetObject.ChannelName, packetObject.GroupName, &groupId, index)

	if group == nil{

		sentMsg <- false

		return
	}
	
	_, err := group.Conn.Write(packetBuffer.Array())
	
	if err != nil {

		group, groupId = GetValue(packetObject.ChannelName, packetObject.GroupName, &groupId, index)

		if group == nil{

			sentMsg <- false

			return
		}

		goto RETRY

	}

	byteArrayCursor := make([]byte, 8)
	binary.BigEndian.PutUint64(byteArrayCursor, uint64(cursor))

	sentMsg <- WriteSubscriberGrpOffset(index, packetObject, byteArrayCursor)
}

func send(index int, cursor int, subscriberMtx sync.Mutex, packetObject pojo.UDPPacketStruct, conn net.UDPConn, packetBuffer ByteBuffer.Buffer, sentMsg chan bool){ 

	defer ChannelList.Recover()

	subscriberMtx.Lock()
	defer subscriberMtx.Unlock()

	_, err := conn.Write(packetBuffer.Array())
	
	if err != nil {
	
		go ChannelList.WriteLog(err.Error())

		sentMsg <- false

		return
	}

	byteArrayCursor := make([]byte, 8)
	binary.BigEndian.PutUint64(byteArrayCursor, uint64(cursor))

	sentMsg <- WriteSubscriberGrpOffset(index, packetObject, byteArrayCursor)

}

func (e *ChannelMethods) SendAck(messageMap pojo.UDPPacketStruct, ackChan chan bool){

	defer ChannelList.Recover()

	var byteBuffer = ByteBuffer.Buffer{
		Endian:"big",
	}

	byteBuffer.PutLong(len(messageMap.Producer_id))

	byteBuffer.PutByte(byte(0)) // status code

	byteBuffer.Put([]byte(messageMap.Producer_id))

	_, err := messageMap.Conn.Write(byteBuffer.Array())

	if err != nil{

		go ChannelList.WriteLog(err.Error())

		ackChan <- false

		return
	}

	ackChan <- true

}