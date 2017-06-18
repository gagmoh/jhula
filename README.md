Package Name : jhula

Requirement:
 
	a) Multiple clients can connect simultanously to same tcp port.
 
	b) Single TCP client-server connection operates independent of other connections on same port.

	c) Application has multiple threads of execution for proessing incoming packets and sending packets for single tcp socket. E.g. Well Known application is openflow implementation, where multiple packetIn messages can be serviced independent of each other.

	d) Application configurable on per TCP port basis:
		1) Maximum number of client connections allowed
		2) Number of go threads to spawn for processing incoming packets.
		3) Size of incoming packet queue
		4) Size of outgoing packet queue

Description: Package provides underlying infrastrcuture to application for above mentioned requirements.

	Step 1: Application calls for new instance of jhula package for tcp port
		jh = jhula.New(maxConCount int,InQSize int , OutQSize int, inThCnt int, outThCnt int) *jhulaHead

	Step 2: Application registers the callback function(s) for handling incoming packets
		jh.RegisterInc(tmp) 	// function name -> tmp

	Step 3: Application asks jhula to start listening on TCP port.
		jh.EstCon(":8081",nil)  // nil represents sync semaphore address
	
	jhula watches for interrupts to gracefully close the go threads.

Example Code :

package main

import "fmt"
import "jhula"
import "bytes"

func main() {

  fmt.Println("Launching server...")
  //jhula.New(maxConCount int,InQSize int , OutQSize int, inThCnt int, outThCnt int) *jhulaHead {
  jh := jhula.New(3,2,2,5,2)

  //Register callback with jhula
  jh.RegisterInc(tmp)

  // Ask jhula to start listening for client requests on port 8081
  jh.EstCon(":8081",nil)   //nil represent sync semaphore address

}

func tmp (inQ *jhula.InQ) {

	// Message from jhula. Extract the packet .
        newMsg := (inQ.InPacket.Bytes())

        tst := bytes.NewBuffer(make([]byte,0,2048))
        //n,err := tst.Write([]byte("hello\n") )
        _,err := tst.Write(newMsg)
        _,_ = tst.Write([]byte ("\n") )
        if err != nil {
                fmt.Println(err)
        }
	// Application asks for transmission by calling AddToTxQ(*bytes.Buffer)
	// Transmit the packet. Application needs to use custom socket that came with packet.
        inQ.ConForApp.AddToTxQ(tst)
}

