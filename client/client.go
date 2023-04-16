package main

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"os"
	"protos"
	"strconv"
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

func beginTransaction(nodeName string, client protos.DistributedTransactionsClient, payload *protos.TxnIdPayload) *protos.Reply {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.BeginTransaction(ctx, payload)
	if err != nil {
		logrusLogger.WithField("node", clientID).Fatal("client.BeginTransaction failed: %v", err)
		return nil
	}
	return resp
}

func abortTransaction(nodeName string, client protos.DistributedTransactionsClient, payload *protos.TxnIdPayload) *protos.Reply {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.AbortCoordinator(ctx, payload)
	if err != nil {
		logrusLogger.WithField("node", clientID).Fatal("client.AbortCoordinator failed: %v", err)
		return nil
	}
	return resp
}

func commitTransaction(nodeName string, client protos.DistributedTransactionsClient, payload *protos.TxnIdPayload) *protos.Reply {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.CommitCoordinator(ctx, payload)
	if err != nil {
		logrusLogger.WithField("node", clientID).Fatal("client.CommitCoordinator failed: %v", err)
		return nil
	}
	return resp
}

func performOp(nodeName string, client protos.DistributedTransactionsClient, payload *protos.TransactionOpPayload) *protos.Reply {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.PerformOperationCoordinator(ctx, payload)
	if err != nil {
		logrusLogger.WithField("node", clientID).Fatal("client.PerformOperationCoordinator failed: %v", err)
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
			reply := beginTransaction(clientID, coordinatorClient, &protos.TxnIdPayload{TxnId: fmt.Sprint(txnID, "-", clientID)})
			if reply.Success {
				fmt.Println("OK")
			}
		} else if strings.ToLower(commandInfo[0]) == "deposit" {
			if len(commandInfo) != 3 {
				continue
			}
			destinationDetails := strings.Split(commandInfo[1], ".")
			amount, _ := strconv.ParseInt(commandInfo[2], 10, 32)

			reply := performOp(clientID, coordinatorClient, &protos.TransactionOpPayload{ID: fmt.Sprint(txnID, "-", clientID), Operation: "DEPOSIT", Account: destinationDetails[1], Branch: destinationDetails[0], Amount: int32(amount)})
			if reply.Success {
				fmt.Println("OK")
			} else {
				fmt.Println("ABORTED")
			}
		} else if strings.ToLower(commandInfo[0]) == "withdraw" {
			if len(commandInfo) != 3 {
				continue
			}
			destinationDetails := strings.Split(commandInfo[1], ".")
			amount, _ := strconv.ParseInt(commandInfo[2], 10, 32)

			reply := performOp(clientID, coordinatorClient, &protos.TransactionOpPayload{ID: fmt.Sprint(txnID, "-", clientID), Operation: "WITHDRAW", Account: destinationDetails[1], Branch: destinationDetails[0], Amount: int32(amount)})
			if reply.Success {
				fmt.Println("OK")
			} else {
				fmt.Println("ABORTED")
			}
		} else if strings.ToLower(commandInfo[0]) == "balance" {
			if len(commandInfo) != 2 {
				continue
			}
			destinationDetails := strings.Split(commandInfo[1], ".")
			reply := performOp(clientID, coordinatorClient, &protos.TransactionOpPayload{ID: fmt.Sprint(txnID, "-", clientID), Operation: "BALANCE", Account: destinationDetails[1], Branch: destinationDetails[0]})
			if reply.Success {
				fmt.Printf("%s = %d", commandInfo[1], reply.Value)
			} else {
				fmt.Println("ABORTED")
			}
		} else if strings.ToLower(commandInfo[0]) == "commit" {
			if len(commandInfo) != 1 {
				continue
			}
			reply := commitTransaction(clientID, coordinatorClient, &protos.TxnIdPayload{TxnId: fmt.Sprint(txnID, "-", clientID)})
			if reply.Success {
				fmt.Println("COMMIT OK")
			} else {
				fmt.Println("ABORTED")
			}
		} else if strings.ToLower(commandInfo[0]) == "abort" {
			if len(commandInfo) != 1 {
				continue
			}
			reply := abortTransaction(clientID, coordinatorClient, &protos.TxnIdPayload{TxnId: fmt.Sprint(txnID, "-", clientID)})
			if reply.Success {
				fmt.Println("ABORTED")
			}
		}
	}
}
