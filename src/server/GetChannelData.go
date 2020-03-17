package server

import(
	_"pojo"
	"encoding/json"
	"bytes"
	"encoding/binary"
	"sync"
	_"log"
	"ChannelList"
	"time"
	"net"
)

type ChannelMethods struct{
	sync.RWMutex
}

func (e *ChannelMethods) GetChannelData(){

	defer ChannelList.Recover()

	for channelName := range ChannelList.TCPStorage {

	    e.runChannel(channelName)

	}

}

func (e *ChannelMethods) runChannel(channelName string){

	defer ChannelList.Recover()

	for index := range ChannelList.TCPStorage[channelName].BucketData{

		time.Sleep(100)

		go func(BucketData chan map[string]interface{}, channelName string){

			defer ChannelList.Recover()

			defer close(BucketData)

			var waitgroup sync.WaitGroup

			for{

				select {

					case message, ok := <-BucketData:	

						if ok{

							var subchannelName = message["channelName"].(string)

							if(channelName == subchannelName && channelName != "heart_beat"){	

								var conn = message["conn"].(net.TCPConn)

								delete(message, "conn")

								waitgroup.Add(1)

								go e.sendMessageToClient(conn, message, channelName, &waitgroup)

								waitgroup.Wait()
							}
						}		
						break
				}		
			}

		}(ChannelList.TCPStorage[channelName].BucketData[index], channelName)
	}
}

func (e *ChannelMethods) sendMessageToClient(conn net.TCPConn, message map[string]interface{}, channelName string, wg *sync.WaitGroup){

	defer ChannelList.Recover()

	defer wg.Done()

	var waitgroup sync.WaitGroup

	var waitAckgroup sync.WaitGroup

	for index := range ChannelList.TCPSocketDetails[channelName]{

		var packetBuffer bytes.Buffer

		if len(ChannelList.TCPSocketDetails[channelName]) <= index{
			break
		} 

		if ChannelList.TCPSocketDetails[channelName][index].ContentMatcher == nil{

			jsonData, err := json.Marshal(message)

			if err != nil{
				go ChannelList.WriteLog(err.Error())
				break
			}

			sizeBuff := make([]byte, 4)

			binary.LittleEndian.PutUint32(sizeBuff, uint32(len(jsonData)))
			packetBuffer.Write(sizeBuff)
			packetBuffer.Write(jsonData)

			waitgroup.Add(1)

			go e.send(channelName, index, packetBuffer, &waitgroup)

			waitgroup.Wait()

		}else{

			var cm = ChannelList.TCPSocketDetails[channelName][index].ContentMatcher

			var matchFound = true

			var messageData = message["data"].(map[string]interface{})


			if _, found := cm["$and"]; found {
			    
			    matchFound = AndMatch(messageData, cm)

			}else if _, found := cm["$or"]; found {

				matchFound = OrMatch(messageData, cm)

			}else if _, found := cm["$eq"]; found {

				if cm["$eq"] == "all"{

					matchFound = true

				}else{

					matchFound = false

				}

			}else{

				matchFound = false

			}

			if matchFound == true{

				jsonData, err := json.Marshal(message)

				if err != nil{
					go ChannelList.WriteLog(err.Error())
					break
				}

				sizeBuff := make([]byte, 4)

				binary.LittleEndian.PutUint32(sizeBuff, uint32(len(jsonData)))
				packetBuffer.Write(sizeBuff)
				packetBuffer.Write(jsonData)

				waitgroup.Add(1)

				go e.send(channelName, index, packetBuffer, &waitgroup)

				waitgroup.Wait()

			}

		}

	}

	waitAckgroup.Add(1)

	go e.SendAck(message, conn, &waitAckgroup)

	waitAckgroup.Wait()
}

func (e *ChannelMethods) send(channelName string, index int, packetBuffer bytes.Buffer,  wg *sync.WaitGroup){

	defer ChannelList.Recover()

	var totalRetry = 0
		
	if len(ChannelList.TCPSocketDetails[channelName]) > index{

		RETRY:

		totalRetry += 1

		if totalRetry > 5{

			wg.Done()

			return

		}

		e.Lock()

		_, err := ChannelList.TCPSocketDetails[channelName][index].Conn.Write(packetBuffer.Bytes())
		
		e.Unlock()

		if err != nil {
		
			go ChannelList.WriteLog(err.Error())

			e.Lock()

			var channelArray = ChannelList.TCPSocketDetails[channelName]
			copy(channelArray[index:], channelArray[index+1:])
			channelArray[len(channelArray)-1] = nil
			ChannelList.TCPSocketDetails[channelName] = channelArray[:len(channelArray)-1]

			e.Unlock()

			goto RETRY

		}else{

			wg.Done()

		}
	}
}

func (e *ChannelMethods) SendAck(messageMap map[string]interface{}, conn net.TCPConn, wg *sync.WaitGroup){

	defer ChannelList.Recover()

	defer wg.Done()

	var messageResp = make(map[string]interface{})

	messageResp["producer_id"] = messageMap["producer_id"].(string)

	jsonData, err := json.Marshal(messageResp)

	if err != nil{

		go ChannelList.WriteLog(err.Error())

		return
	}

	var packetBuffer bytes.Buffer

	buff := make([]byte, 4)

	binary.LittleEndian.PutUint32(buff, uint32(len(jsonData)))

	packetBuffer.Write(buff)

	packetBuffer.Write(jsonData)

	e.Lock()
	_, err = conn.Write(packetBuffer.Bytes())
	e.Unlock()

	if err != nil{

		go ChannelList.WriteLog(err.Error())

	}
}