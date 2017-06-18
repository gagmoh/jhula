package  jhula

import (
	//"fmt"
	"bytes"
	"io"
	"sync"
)

// confJhulaPktQ creates queue to handle incoming and outgoing packets
func (conPkt *ConnDet) confJhulaPktQ(inQSize int, outQSize int,  inThreadCnt int, outThreadCnt int) (ptQ *pktQ) {
	ptQ = new(pktQ)

	//channel resposible to hold pointer to pkt going towards application.
	ptQ.QTowApp = make(chan *InQ, inQSize)

	// channel holds the packets added by application code for egress
	ptQ.OutPacket = make(chan *bytes.Buffer, outQSize)

	// channel holds incoming packets received on socket.

	ptQ.Buf = make(chan *bytes.Buffer, inQSize)

	// packet sent out on socket.
	ptQ.OutBuf = make(chan *bytes.Buffer,1)


	// Allocate inQSize pointers to block size 2048 and fill the BUF channel
	for i:=0; i< inQSize ; i++ {
		ptQ.Buf <-  bytes.NewBuffer(make([]byte,0,2048))
	}
	// Allocate outQSize pointers and fill the OutPacket channel
	for i:=0;i<outQSize;i++ {
		ptQ.OutPacket <- bytes.NewBuffer(make([]byte,0,2048))
	}

	return
}

// Function called to register application handler function for incoming packet and spawn new threads
func (pkQ *pktQ) regNewThreads(InCallBack []func(*InQ), inThreadCnt int, wg *sync.WaitGroup) {

	// Create inThreadCnt go routines for handling incoming packets in parallel
	for iFn:=0 ; iFn < inThreadCnt ; iFn++ {
		//Anonymous function
		go func(tid int) {
			defer wg.Done()
			for {
			select {
				case tmp,ok := <-pkQ.QTowApp : // wait for something written to inPacket Queue
				if !ok {
					// channel is closed
					Verbose.Println("Channel QTowApp closed. Clean-up initiated.")
					return
				}
				for _,fn := range InCallBack {
					Verbose.Println("Invoke function to handle incoming packet. Thread Id: ",tid)
					fn(tmp)
				}

				tmp.InPacket.Reset()
				pkQ.Buf <- tmp.InPacket //put the addres back to buffer channel for use.
			}
			}
		}(iFn)
	}

}

// Non-Blocking Function call (if spare slot in out queue) called by application code to send out pkt on socket
func (pkQ *pktQ) addToTxQ(outPkt *bytes.Buffer) {
	var tmp *bytes.Buffer

	tmp,ok := <-pkQ.OutPacket
	if !ok {
		Error.Println("Failure to send message")
		return
	}
	io.Copy(tmp,outPkt)

	pkQ.OutBuf <-tmp

}
