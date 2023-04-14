package main

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"os"
	"protos"
	"strings"
	"sync"
	"time"
	"utils"

	"github.com/sirupsen/logrus"
)

var logrusLogger = logrus.New()
var clientID string

func getServerNames(nodeToClient map[string]protos.DistributedTransactionsClient) []string {
	servers := make([]string, len(nodeToClient))
	i := 0
	for k := range nodeToClient {
		servers[i] = k
		i++
	}
	return servers

}
func pickRandomNode(nodeToClient map[string]protos.DistributedTransactionsClient, servers []string) string {
	rand.Seed(time.Now().Unix())
	chosenNode := servers[rand.Intn(len(servers))]
	return chosenNode
}

func beginTransaction(nodeName string, client protos.DistributedTransactionsClient, payload *protos.BeginTxnPayload) *protos.Reply {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.BeginTransaction(ctx, payload)
	if err != nil {
		logrusLogger.WithField("node", clientID).Fatal("client.BeginTransaction failed: %v", err)
		return nil
	}
	return resp
}

func main() {
	arguments := os.Args
	if len(arguments) != 3 {
		fmt.Printf("Please provide arguments as <client-id> <config-file>!")
		return
	}
	var wg sync.WaitGroup
	clientID = arguments[1]
	config := arguments[2]
	utils.InitlogrusLogger(logrusLogger)
	nodeToUrl := map[string]string{}
	nodeToClient := utils.SafeRPCClientMap{M: make(map[string]protos.DistributedTransactionsClient)}
	logrusLogger.WithField("node", clientID).Debug("Starting node ", clientID, " with config file ", config)
	utils.ParseConfigFile(config, nodeToUrl)
	for nodeName, address := range nodeToUrl {
		wg.Add(1)
		go utils.EstablishConnection(clientID, nodeName, address, &wg, &nodeToClient, logrusLogger)
	}
	wg.Wait()
	servers := getServerNames(nodeToClient.M)

	scanner := bufio.NewScanner(os.Stdin)
	txnID := 0

	var coordinatorClient protos.DistributedTransactionsClient
	for scanner.Scan() {
		command := scanner.Text()
		commandInfo := strings.Fields(command)
		if len(commandInfo) == 0 {
			continue
		}
		if strings.ToLower(commandInfo[0]) == "begin" {
			txnID++
			if len(commandInfo) != 1 {
				continue
			}
			coordinator := pickRandomNode(nodeToClient.M, servers)
			logrusLogger.WithField("node", clientID).Debug("Coordinator for this transaction is: ", coordinator)
			txnID++
			coordinatorClient = nodeToClient.M[coordinator]
			reply := beginTransaction(clientID, coordinatorClient, &protos.BeginTxnPayload{TxnId: fmt.Sprint(txnID, "-", clientID)})
			if reply.Success {
				fmt.Println("OK")
			}
		} else if strings.ToLower(commandInfo[0]) == "transfer" {
			if len(commandInfo) != 5 {
				continue
			}
		}
	}
}
