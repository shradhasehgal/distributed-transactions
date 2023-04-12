package utils

import (
	"encoding/gob"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

func EstablishConnection(currNodeName string, nodeName string, address string, wg *sync.WaitGroup, nodeToEncoder *SafeEncoderMap, outgoingConnectionsDone chan bool, logrusLogger *logrus.Logger) {
	defer wg.Done()
	for {
		connection, err := net.Dial("tcp", address)
		if err != nil {
			logrusLogger.WithField("node", currNodeName).Debug("Error encountered while establishing TCP connection with ", address, " - ", err)
			time.Sleep(2 * time.Second)
			continue
		}
		encoder := gob.NewEncoder(connection)
		SetEncoder(nodeToEncoder, nodeName, encoder)
		// var msg = Message{Type: "registration", Payload: Registration{NodeID: currNodeName}}
		// unicast(encoder, msg)
		break
	}
	outgoingConnectionsDone <- true
	logrusLogger.WithField("node", currNodeName).Debug("Connected to ", nodeName)
}
