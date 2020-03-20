package main

import (
	"brahmaputra"
	"log"
	"time"
)


func main() {
	
	var brahm = &brahmaputra.CreateProperties{
		Host:"127.0.0.1",
		Port:"8100",
		AuthToken:"dkhashdkjshakhdksahkdghsagdghsakdsa",
		ConnectionType:"tcp",
		ChannelName:"Abhik",
		AppType:"consumer",
		OffsetPath:"D:\\pounze_go_project\\brahmaputra\\subscriber_offset.offset", //writes last offset received
		AlwaysStartFrom:"BEGINNING", // BEGINNING | NOPULL | LASTRECEIVED,
	}

	brahm.Connect()

	var cm = `
		{
			"$eq":"all"
		}
	`

	brahm.Subscribe(cm)

	time.Sleep(2 * time.Second)

	// var count = 0

	for{
		select{
			case message, ok := <-brahmaputra.SubscriberChannel:	
				if ok{
						
					// count += 1 
						
					log.Println(message)

				}	

			break
		}
	}
}