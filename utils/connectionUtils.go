package utils

import (
	"protos"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func EstablishConnection(currNodeName string, nodeName string, address string, wg *sync.WaitGroup, nodeToClient *SafeRPCClientMap, logrusLogger *logrus.Logger) {
	// Decrement the counter when the goroutine completes.
	defer wg.Done()
	for {
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		conn, err := grpc.Dial(address, opts...)
		if err != nil {
			logrusLogger.WithField("node", currNodeName).Debug("Error encountered while establishing TCP connection with ", address, " - ", err)
			time.Sleep(10 * time.Second)
			continue
		}
		client := protos.NewDistributedTransactionsClient(conn)
		SetClient(nodeToClient, nodeName, client)

		break
	}
	logrusLogger.WithField("node", currNodeName).Debug("Connected to  ", nodeName, "->", address)
}
