package jhula

import (
	"bytes"
	"net"
	"sync"
	"os"
)

type InQ  struct {
	InPacket *bytes.Buffer
	ConForApp *ConnDet
}

type OutQ struct {
	OutPacket chan *bytes.Buffer
}

// scope : Internal
type pktQ struct {
	QTowApp chan *InQ
	OutQ
	Buf  chan *bytes.Buffer
	OutBuf chan *bytes.Buffer
}

type ConnDet struct {
	ConnPtr *net.TCPConn	//sock val returned from accept function
	ConPktQ *pktQ		// pointer to incoming and outgoing queue structure
	PortId  string		// port id structure
	Abort   chan bool	// for trigger cleanup.
	conWg   *sync.WaitGroup	// to sync graceful cleanup of go routines
}

type jhulaHead struct {
	ConCount int  // count number of incoming connections to tcp server port
	JhConf *jhulaConf // pointer to configuration
	ConDet []*ConnDet //pointer to custom socket structure
	wg *sync.WaitGroup //counting semaphore to capture number of threads. for clean-up purpose.

	interrupt chan os.Signal //handle interrupt from OS

	Sock	*net.TCPListener
	ClosePort chan bool
}

// scope : internal
type jhulaConf struct {
	InQSize int		// define queue size for incoming packets
        OutQSize int		// define queue size of outgoing packets
        InThreadCnt int		// define number of go routines to process incoming pkt q
        OutThreadCnt int	// not used
	maxCon int		// max number of connections supported
	InCallBack []func(*InQ) // application defined callback to handle incoming packets
}
