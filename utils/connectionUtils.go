package utils

import (
	"context"
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()

		conn, err := grpc.DialContext(ctx, address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		if err != nil {
			logrusLogger.WithField("node", currNodeName).Debug("Error encountered while establishing TCP connection with ", address, " - ", err)
			continue
		}
		client := protos.NewDistributedTransactionsClient(conn)
		SetClient(nodeToClient, nodeName, client)
		break
	}
	logrusLogger.WithField("node", currNodeName).Debug("Connected to  ", nodeName, "->", address)
}
