package server

import (
	"pojo"
	"net"
	"encoding/json"
	"time"
	"strconv"
	"log"
	"bytes"
	"encoding/binary"
	"io"
	"sync"
	"ChannelList"
	"MongoConnection"
)

var WriteCallback = make(chan bool)
var FileWriteLock = &sync.RWMutex{}

func ParseMsg(msg string, conn net.TCPConn, parseChan chan bool){

	defer ChannelList.Recover()

	messageMap := make(map[string]interface{})

	err := json.Unmarshal([]byte(msg), &messageMap)

	if err != nil{
		parseChan <- false
		go ChannelList.WriteLog(err.Error())
		return
	}

	if messageMap["type"] == "heart_beat"{

		if messageMap["channelName"] == ""{
			conn.Close()
			parseChan <- false
			go ChannelList.WriteLog("Invalid message received..." + msg)
			return
		}

		log.Println("HEART BEAT RECEIVED...")

		messageMap["conn"] = conn
		
		ChannelList.TCPStorage["heart_beat"].BucketData[0] <- messageMap

	}else if messageMap["type"] == "publish"{

		if messageMap["channelName"] == ""{
			conn.Close()
			parseChan <- false
			go ChannelList.WriteLog("Invalid message received..." + msg)
			return
		}

		if messageMap["data"] == ""{
			conn.Close()
			parseChan <- false
			go ChannelList.WriteLog("Data missing..." + msg)
			return
		}

		var channelName = messageMap["channelName"].(string)

		if ChannelList.TCPStorage[channelName] == nil{
			parseChan <- false
			return
		}

		currentTime := time.Now()

		epoch := currentTime.Unix()

		nanoEpoch := currentTime.UnixNano()

		var _id = strconv.FormatInt(nanoEpoch, 10)

		messageMap["_id"] = _id

		var indexNo = epoch % int64(ChannelList.TCPStorage[channelName].Worker)
		
		jsonData, err := json.Marshal(messageMap)

		if err != nil{
			parseChan <- false
			go ChannelList.WriteLog(err.Error())
			return
		}

		if *ChannelList.ConfigTCPObj.Storage.File.Active{

			var byteLen = len(jsonData)
		
			go WriteData(jsonData, channelName, byteLen)

			select {

				case message, ok := <-WriteCallback:	
					if ok{
						
						if !message{
							parseChan <- false
							return
						}
					}

					break
			}

			go WriteOffset(jsonData, channelName, byteLen, _id)

			select {

				case message, ok := <-WriteCallback:	
					if ok{
						
						if !message{
							parseChan <- false
							return
						}
					}		

					break
			}
		}

		if *ChannelList.ConfigTCPObj.Storage.Mongodb.Active{

			var byteLen = len(jsonData)
 
			go WriteMongodbData(nanoEpoch, messageMap["AgentName"].(string), jsonData, channelName, byteLen)

			select {

				case message, ok := <-WriteCallback:	
					if ok{
						
						if !message{
							parseChan <- false
							return
						}
					}		
					break
			}
		} 

		messageMap["conn"] = conn

		ChannelList.TCPStorage[channelName].BucketData[indexNo] <- messageMap

		parseChan <- true

	}else if messageMap["type"] == "subscribe"{

		if messageMap["channelName"] == ""{
			conn.Close()
			parseChan <- false
			go ChannelList.WriteLog("Invalid message received..." + msg)
			return
		}

		if messageMap["contentMatcher"] == ""{
			conn.Close()
			parseChan <- false
			go ChannelList.WriteLog("Content matcher cannot be empty..." + msg)
			return
		}

		var channelName = messageMap["channelName"].(string)

		if ChannelList.TCPStorage[channelName] == nil{
			parseChan <- false
			return
		}

		var socketDetails *pojo.SocketDetails

		if messageMap["contentMatcher"] == "all"{

			socketDetails = &pojo.SocketDetails{
				Conn:conn,
				ContentMatcher: nil,
			}

		}else{
			socketDetails = &pojo.SocketDetails{
				Conn:conn,
				ContentMatcher: messageMap["contentMatcher"].(map[string]interface{}),
			}
		}
		
		ChannelList.TCPSocketDetails[channelName] = append(ChannelList.TCPSocketDetails[channelName], socketDetails) 

		parseChan <- true 

	}else{
		conn.Close()
		parseChan <- false
		go ChannelList.WriteLog("Invalid message type must be either publish or subscribe...")
		return
	}	
}

// func ReadDataUsingOffset(){
// 	var byteSize = (byteLen + 4)

// 	var readByte = make([]byte, byteSize)

// 	_, err = ChannelList.TCPStorage[channelName].FD.ReadAt(readByte, ChannelList.TCPStorage[channelName].Offset)

// 	if err != nil{
// 		log.Println(err.Error())
// 		return
// 	}

// 	ChannelList.TCPStorage[channelName].Offset += int64(byteSize)

// 	readByte = readByte[4:]

// 	messageMap = make(map[string]interface{})

// 	err = json.Unmarshal(readByte, &messageMap)

// 	if err != nil{
// 		go ChannelList.WriteLog(err.Error())
// 		return
// 	}
// }

func WriteMongodbData(_id int64, agentName string, jsonData []byte, channelName string, byteLen int){

	defer ChannelList.Recover()

	var packetBuffer bytes.Buffer

	buff := make([]byte, 4)

	binary.LittleEndian.PutUint32(buff, uint32(byteLen))

	packetBuffer.Write(buff)

	packetBuffer.Write(jsonData)

	var oneDoc = make(map[string]interface{})

	oneDoc["offsetID"] = _id
	oneDoc["AgentName"] = agentName
	oneDoc["cluster"] = *ChannelList.ConfigTCPObj.Server.TCP.ClusterName
	oneDoc["data"] = packetBuffer.Bytes()

	var status, _ = MongoConnection.InsertOne(channelName, oneDoc)

	if !status{
		WriteCallback <- false
	}else{
		WriteCallback <- true
	}
}

func WriteData(jsonData []byte, channelName string, byteLen int){

	defer ChannelList.Recover()

	FileWriteLock.Lock()

	var packetBuffer bytes.Buffer

	buff := make([]byte, 4)

	binary.LittleEndian.PutUint32(buff, uint32(byteLen))

	packetBuffer.Write(buff)

	packetBuffer.Write(jsonData)
 
	_, err := ChannelList.TCPStorage[channelName].FD.Write(packetBuffer.Bytes())

	FileWriteLock.Unlock()
	
	if (err != nil && err != io.EOF ){
		go ChannelList.WriteLog(err.Error())
		WriteCallback <- false
		return
	}

	WriteCallback <- true	
}

func WriteOffset(jsonData []byte, channelName string, byteLen int, _id string){

	defer ChannelList.Recover()
	
	FileWriteLock.Lock()

	var byteSize = (byteLen + 4)

	ChannelList.TCPStorage[channelName].Offset += int64(byteSize)

	var offset = strconv.FormatInt(ChannelList.TCPStorage[channelName].Offset, 10)

	var offsetString = *ChannelList.ConfigTCPObj.Server.TCP.ClusterName+"|"+offset+"|"+_id+"\r\n"

	var packetBuffer bytes.Buffer

	buff := make([]byte, 2)

	binary.LittleEndian.PutUint16(buff, uint16(len(offsetString)))

	packetBuffer.Write(buff)
	packetBuffer.Write([]byte(offsetString))

	_, err := ChannelList.TCPStorage[channelName].TableFD.Write(packetBuffer.Bytes())

	FileWriteLock.Unlock()

	if (err != nil && err != io.EOF){

		go ChannelList.WriteLog(err.Error())

		WriteCallback <- false

	}

	WriteCallback <- true
}