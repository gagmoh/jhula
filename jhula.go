//conn.SetReadDeadline(time.Now().Add(timeoutDuration))
package jhula

import (
//	"fmt"
	"net"
	"bytes"
	"os"
	"os/signal"
	"sync"
)

//type fn func(*InQ)

//***************************************************************
// SCOPE : EXPOSED
// Description: Jhula.New instantiates new instance for jhula. Jhula Instance is needed for each new tcp port
// Arguments: 
//		Arg1  maximum number of parallel TCP connectiions supported on listening server port
// 		Arg2  Size of Queue for ingressing packets per TCP connection
// 		Arg3  Size of Queue for outgoing packets per TCP connection
// 		Arg4  Count of threads spawned to receive incoming packets and handover to application
// 		Arg5  Count of threads spawned to handle outgoing packet request. Not used currently
// E.g jh := jhula.New(3,2,2,5,2)
//**************************************************************
func New(maxConCount int,InQSize int , OutQSize int, inThCnt int, outThCnt int) *jhulaHead {
	return &jhulaHead{
		ConCount: 0,
		JhConf: &jhulaConf{
			maxCon: maxConCount,  // maximum number of connections allowed on server port
			InQSize: InQSize, // size of incoming queue
			OutQSize: OutQSize, // size of outgoing queue. multiple applications threads may add to same qu
			InThreadCnt: inThCnt, // number of go threads to handle incoming pkt. each will call app func.
			OutThreadCnt: outThCnt, // not used currently
			},
		wg:  new(sync.WaitGroup),	// counting semaphore to track number of go threads
		ConDet: make([]*ConnDet,0,maxConCount), //slice to store pointer to ConnDet
		interrupt: make(chan os.Signal, 1), // channel to handle os interrupt signal
	}
}

//***************************************************************
// SCOPE       : EXPOSED	
// Description : RegisterInc Function called by application to register the function handler(s) for incoming packets.
// Arguments   :
//		Arg1 function pointer with argument of pointer type (INQ)
// E.g. jh.RegisterInc(tmp) , where tmp is function name - func tmp (inQ *jhula.InQ)
//***************************************************************
func (jh *jhulaHead) RegisterInc(fnList ...func(*InQ)) {
	jh.JhConf.registerInc(fnList...)
}

//***************************************************************
// SCOPE       : EXPOSED 
// Description : EstCon function called by application to start listening on TCP port
// Arguments   :
//      Arg1 TCP port in string type. e.g. :4463
// 	Arg2 Semaphore to handle graceful clean-up. This will be needed when application calls EstCon in sep thread.
// E.g. jh.EstCon(":8081")
//***************************************************************
func (jh *jhulaHead) EstCon(port string, ExtSync *sync.WaitGroup) error {

	if ExtSync != nil {
		ExtSync.Add(1)
	}
	logInit("log"+port+".txt")
	// register to receive OS interrupts
	signal.Notify(jh.interrupt,os.Interrupt,os.Kill)
	go jh.handleInterrupt()

	addr,err := net.ResolveTCPAddr("tcp4",port)

	if err != nil {
		Error.Println("Error in resolving address ",err.Error() )
		return err
	}

	tcpSock ,err := net.ListenTCP("tcp4",addr)
	jh.Sock = tcpSock

	Verbose.Println("Starting to Listen on port: ",port)

	if err != nil {
		Error.Printf("failure :%s to open port in listen mode %s \n",(err.Error()),port )
		return err
	}

	defer func() {
		tcpSock.Close()
		ExtSync.Done()
	}()

	for {
		newConn,err := tcpSock.AcceptTCP()
		if err != nil {
			if err.Error() != "EOF" {

				Warning.Printf("failurr %s to establish conn with incoming sock requesst. TCP Port %s \n ",(err.Error()),port )
			break
			}
		} else {
			Warning.Println("Success in establishing connection. TCP Port ",port )
		}

		if (jh.ConCount+1) >= jh.JhConf.maxCon {
			Warning.Println("Max limit for connection to server port reached. Dropping new connection",port)
			newConn.Close()
			continue
		}

		// Each new connection request will be handled in different go routine 
		jh.ConCount = jh.ConCount + 1
		jh.wg.Add(1)
		go jh.handleIncoming(newConn,port)
	}

	for _,con := range jh.ConDet {
		Warning.Println("Abort len : ",len(con.Abort) )
		if len(con.Abort) > 0 {  	// hack.Abort Already triggered. No action required
			con.Abort <- true
		}
	}
	Verbose.Println("waiting for clean up . TCP Port ",port)
	jh.wg.Wait()
	Verbose.Println("Done waiting. Cleanup done for TCP port ",port)
	close(jh.interrupt) // Takes care of return from interrupt handler routine. 
	return err
}

// handleInterrupt catches the interrupt signal and ensure thread clean-up.
func (jh *jhulaHead) handleInterrupt() {
	//defer jh.wg.Done()
	select {
	case _,ok := <-jh.interrupt:
		if !ok {
			return
		}
	        for  _,con := range jh.ConDet {
			con.Abort <- true
		}
		jh.Sock.Close()
		return
	}
}

// jhulaHead.New function triggers Pkt Queue Creation for created TCPConn socket
// Arg1 Connection identifier for new tcp connection
// Arg2 TCP port details as passed by application
func (jh *jhulaHead) newJH(conn *net.TCPConn, portStr string) (conDet *ConnDet) {
	conDet = &ConnDet{
			ConnPtr: conn, 		// Tcp socket
			PortId: portStr,	//port number
			Abort: make(chan bool, 1),	//closing socket
			conWg: new(sync.WaitGroup),	//sync graceful clean-up of go routines
		}
	conDet.ConPktQ = conDet.confJhulaPktQ(jh.JhConf.InQSize,
					jh.JhConf.OutQSize,
					jh.JhConf.InThreadCnt,
					jh.JhConf.OutThreadCnt)

	// add custom socket struct to jhula structure
	jh.ConDet = append(jh.ConDet,conDet)

	//initialize to incoming threads configured by application
	conDet.conWg.Add(jh.JhConf.InThreadCnt)

	// Pass callback function information to pktq struct
	conDet.ConPktQ.regNewThreads(jh.JhConf.InCallBack, jh.JhConf.InThreadCnt,conDet.conWg)

	return
}

//handleIncoming function thread to isolate processing of new incoming thread
func (jh *jhulaHead) handleIncoming(conn *net.TCPConn, portStr string) {
	Terse.Println("Function handleIncoming : start processing to new connection on port ",portStr)

	conDet := jh.newJH(conn,portStr)

	// spawn new go routine to listen to incoming pkt q and increment sync semaphore
	conDet.conWg.Add(1)
        go conDet.recvTcpPkt()

	// spawn new go routine to transmit packets on server port and increment sync semaphore
	conDet.conWg.Add(1)
	go conDet.txTcpPkt()

	conDet.conWg.Wait()
	Verbose.Println("HandleIncoming done waiting ",portStr)
	conDet.cleanUp()
	close(conDet.Abort)

	jh.ConCount = jh.ConCount -1
	//Delete connection details from JH. TODO: Change slice to map in-future
	for i,con := range jh.ConDet {
		if con == conDet {
			jh.ConDet = append(jh.ConDet[:i], jh.ConDet[i+1:]...)
			break
		}
	}
	jh.wg.Done()
}

//CleanUp function handles graceful closure of Jhula Instance
func (conn *ConnDet) cleanUp() {
	//close(conn.ConPktQ.InPacket)
	close(conn.ConPktQ.QTowApp)
	close(conn.ConPktQ.OutPacket)
	close(conn.ConPktQ.Buf)
	close(conn.ConPktQ.OutBuf)
	conn.ConnPtr.Close()
}
// recvTcpPkt handles incoming packets.
func (conn * ConnDet) recvTcpPkt() {
	 VTerse.Println("Function recvTcpPkt : start monitoring incoming pkt queue for port  ",conn.PortId)

	 defer conn.conWg.Done()

	 tmpB := make([]byte,2048)

         for {
		 tmp,ok := <-conn.ConPktQ.Buf
		 if !ok {
			 Warning.Println("Function recvTcpPort : ingress channel closed. Cleanup ")
			 return
		 }

                 cnt,err := conn.ConnPtr.Read(tmpB)
		 //fmt.Println(cnt)

		 select {
		 case <-conn.Abort:
			return
		 default:
                 if err != nil {
                         if err.Error() == "EOF" {
                                Warning.Println("Failed to read from socket. Sock closed ",err.Error() )
				continue
                        } else {
				Warning.Println("Error reading from socket : ",err.Error() )
                        }
                        conn.Abort <- true
                 } else {
		 //hack : TODO readonly buffer size bytes in iteration
		 if cnt > 2048 {
			 cnt = 2048
		 }
  		 // done the hardway.TODO : Packet Housekeeping mechanism 
		 for  i:=0; i<cnt;i++ {
			 tmp.WriteByte(tmpB[i])
		 }

		 nQ := &InQ{
			 InPacket: tmp,
			 ConForApp: conn,
		 }
		 conn.ConPktQ.QTowApp <- nQ
	 }
	 }

        }
}

// tcpTxPkt handles tansmits outgoing packet requests on socket
func (conn *ConnDet) txTcpPkt() {
	defer conn.conWg.Done()

	for {
		select {
		case buf,ok := <-conn.ConPktQ.OutBuf:
			if !ok {
				Warning.Println("Function txTcpPkt : Out channel is closed. Failed to send")
				return
			}
			_,err := conn.ConnPtr.Write(buf.Bytes())
			if err != nil {
				Error.Println("failure to send packet :",err.Error() )
				//Add code to trigger cleanup
				conn.Abort <- true
			} else {
				buf.Reset()
				conn.ConPktQ.OutPacket <- buf
			}

		case <-conn.Abort:
			return
		}
	}
}

// registerInc saves callback function list in Jhula Configuration
func (jC *jhulaConf) registerInc(fnList ...func(*InQ)) {
	jC.InCallBack = append(jC.InCallBack,fnList...)
}

//***************************************************************
// SCOPE       : EXPOSED 
// Description : AddToTXQ called by application code to forward packet for transmission
// Arguments   :
//		Arg1 : Pointer of type bytes.Buffer. Application responsible for this buffer clean-up post function call.
// Eg          : inQ.ConForApp.AddToTxQ(tst)
//***************************************************************
func (conn *ConnDet) AddToTxQ(outPkt *bytes.Buffer) {
	conn.ConPktQ.addToTxQ(outPkt)
}

func (conDet *ConnDet) SockClose() {
	conDet.ConnPtr.Close()
}
