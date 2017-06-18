package jhula

import (
	"log"
//	"io/ioutil"
	"io"
	"os"
)

var (
	VTerse		 *log.Logger
	Terse		 *log.Logger
	Verbose		 *log.Logger
	Warning		 *log.Logger
	Error		 *log.Logger
)

func logInit(fileName string) {
	file,err := os.OpenFile(fileName,os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

	if err != nil {
		log.Fatalln("Failed to open log file: ",err)
	}

	VTerse = log.New(file,"TRACE: ",log.Lshortfile)

	Terse = log.New(file,"TRACE: ",log.Ltime|log.Lshortfile)

	Verbose = log.New(file,"TRACE: ",log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(io.MultiWriter(file,os.Stderr),"WARNING: ",log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(io.MultiWriter(file,os.Stderr),"ERROR: ",log.Ldate|log.Ltime|log.Lshortfile)
}



