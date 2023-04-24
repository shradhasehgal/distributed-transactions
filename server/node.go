package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"protos"
	"sort"
	"strings"
	"sync"
	"time"
	"utils"

	logrus "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type distributedTransactionsServer struct {
	protos.UnimplementedDistributedTransactionsServer
	timestampedConcurrencyID                  uint32
	nodeToClient                              utils.SafeRPCClientMap
	safeTxnIDToServerInvolved                 SafeTxnIdToServersInvolvedPtr
	timestampedConcurrencyIDSet               SafeTimestampedConcurrencyIDSet
	objectNameToStatePtr                      SafeObjectNameToStatePtr
	txnIDToChannel                            SafeTxnIDToChannelMap
	timestampedConcurrencyIDToState           SafeTimestampedConcurrencyIDToStatePtr
	timestampedConcurrencyIDToObjectsInvolved SafeTimestampedConcurrencyIDToObjectsInvolved
}

func newServer() *distributedTransactionsServer {
	s := &distributedTransactionsServer{
		nodeToClient:                              utils.SafeRPCClientMap{M: make(map[string]protos.DistributedTransactionsClient)},
		safeTxnIDToServerInvolved:                 SafeTxnIdToServersInvolvedPtr{M: make(map[string]*(map[string]bool))},
		timestampedConcurrencyIDSet:               SafeTimestampedConcurrencyIDSet{M: make(map[string]bool)},
		timestampedConcurrencyID:                  0,
		objectNameToStatePtr:                      SafeObjectNameToStatePtr{M: make(map[string]*SafeObjectState)},
		txnIDToChannel:                            SafeTxnIDToChannelMap{M: make(map[uint32]chan bool)},
		timestampedConcurrencyIDToState:           SafeTimestampedConcurrencyIDToStatePtr{M: make(map[string]*SafeTransactionState)},
		timestampedConcurrencyIDToObjectsInvolved: SafeTimestampedConcurrencyIDToObjectsInvolved{M: make(map[string]*SafeObjectsSet)},
	}
	return s
}

var timeoutVal = 24 * time.Hour
var currNodeName string

var logrusLogger = logrus.New()

func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func (s *distributedTransactionsServer) BeginTransaction(ctx context.Context, payload *protos.TxnIdPayload) (*protos.Reply, error) {
	s.safeTxnIDToServerInvolved.Mu.Lock()
	defer s.safeTxnIDToServerInvolved.Mu.Unlock()
	// var listOfServersInvolved = []string{}
	var mapOfServersInvolved = make(map[string]bool)
	s.safeTxnIDToServerInvolved.M[payload.TxnId] = &mapOfServersInvolved
	return &protos.Reply{Success: true}, nil
}

func PreparePeerGoRoutine(peer protos.DistributedTransactionsClient, payload *protos.TxnIdPayload, prepareChannel chan bool) {
	vote := PeerWrapper(peer, payload, "PreparePeer")
	if vote == nil || !vote.Success {
		prepareChannel <- false
	}
	prepareChannel <- true
}

func CommitGoRoutine(peer protos.DistributedTransactionsClient, payload *protos.TxnIdPayload, commitChannel chan bool, passing bool) {
	if !passing {
		PeerWrapper(peer, payload, "AbortPeer")
	} else {
		PeerWrapper(peer, payload, "CommitPeer")
	}
	commitChannel <- true
}

func (s *distributedTransactionsServer) CommitCoordinator(ctx context.Context, payload *protos.TxnIdPayload) (*protos.Reply, error) {
	mapOfServersInvolved := GetServersInvolvedInTxn(s)
	if len(*mapOfServersInvolved[payload.TxnId]) == 0 {
		return &protos.Reply{Success: true}, nil
	}
	prepareChannel := make(chan bool, len(*mapOfServersInvolved[payload.TxnId]))
	for serverName := range *mapOfServersInvolved[payload.TxnId] {
		peer := utils.GetClient(&s.nodeToClient, serverName)
		go PreparePeerGoRoutine(peer, payload, prepareChannel)
	}

	countSeversInvolved := 0
	passing := true
	for {
		resp := <-prepareChannel
		if !resp {
			passing = false
		}
		countSeversInvolved++
		if countSeversInvolved == len(*mapOfServersInvolved[payload.TxnId]) {
			break
		}
	}

	commitChannel := make(chan bool, len(*mapOfServersInvolved[payload.TxnId]))
	for serverName := range *mapOfServersInvolved[payload.TxnId] {
		peer := utils.GetClient(&s.nodeToClient, serverName)
		go CommitGoRoutine(peer, payload, commitChannel, passing)
	}

	for {
		<-commitChannel
		countSeversInvolved--
		if countSeversInvolved == 0 {
			break
		}
	}

	if !passing {
		return &protos.Reply{Success: false}, nil
	}

	return &protos.Reply{Success: true}, nil
}

func (s *distributedTransactionsServer) CommitPeer(ctx context.Context, payload *protos.TxnIdPayload) (*protos.Reply, error) {
	logrusLogger.WithField("node", currNodeName).Debug("Committing transaction ID ", payload.TxnId)
	timestampedConcurrencyID := payload.TxnId

	objectsInvolvedPtr := GetSafeTimestampedConcurrencyIDToObjectsInvolved(&s.timestampedConcurrencyIDToObjectsInvolved, timestampedConcurrencyID)
	for objectName := range (*objectsInvolvedPtr).M {
		logrusLogger.WithField("node", currNodeName).Debug("Committing object ", objectName)

		var waitingOnTxnID string = ""

		for {
			objectState := GetObjectState(&s.objectNameToStatePtr, objectName)
			objectState.Mu.RLock()
			keys := make([]string, 0, len(objectState.tentativeWrites))

			for k := range objectState.tentativeWrites {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

			for idx, val := range keys {
				if timestampedConcurrencyID == val {
					if idx != 0 {
						waitingOnTxnID = keys[idx-1]
					}
				}
			}
			objectState.Mu.RUnlock()
			if waitingOnTxnID != "" {
				logrusLogger.WithField("node", currNodeName).Debug("Waiting on txn ID ", waitingOnTxnID, " to commit before committing txn ID ", timestampedConcurrencyID)
				transactionState := GetTransactionState(&s.timestampedConcurrencyIDToState, waitingOnTxnID)
				var state TransactionState
				state = In_Progress
				for state == In_Progress {
					transactionState.Mu.RLock()
					state = transactionState.state
					logrusLogger.WithField("node", currNodeName).Debug("Waiting for state of txn ID ", waitingOnTxnID, " to not be ", In_Progress, ". Current state: ", state)
					transactionState.Mu.RUnlock()
					// Sleep to allow commit of that transaction
					time.Sleep(500 * time.Millisecond)
				}

				logrusLogger.WithField("node", currNodeName).Debug("The transaction ID ", waitingOnTxnID, " that ", timestampedConcurrencyID, " was waiting on has been ", state, "!")
				if state == Committed {
					logrusLogger.WithField("node", currNodeName).Debug("The transaction ID ", waitingOnTxnID, " that ", timestampedConcurrencyID, " was waiting on has been ", state, "! Moving on to next object")
					break
				}
			} else {
				logrusLogger.WithField("node", currNodeName).Debug("No need to wait on any other transactions to commit for objectName ", objectName, " before committing txn ID ", timestampedConcurrencyID)
				break
			}
		}
		// SetObjectCommitted(&s.objectNameToStatePtr, objectName)
	}

	for objectName, objectState := range s.objectNameToStatePtr.M {
		objectState.Mu.Lock()
		_, ok := objectState.tentativeWrites[timestampedConcurrencyID]
		if ok {
			objectState.committedVal = objectState.tentativeWrites[timestampedConcurrencyID]
			objectState.committedTimestamp = timestampedConcurrencyID
			delete(objectState.tentativeWrites, timestampedConcurrencyID)
		}
		if objectState.committedVal > 0 {
			fmt.Printf("%s: %d, ", objectName, objectState.committedVal)
		}

		objectState.Mu.Unlock()
	}

	transactionState := GetTransactionState(&s.timestampedConcurrencyIDToState, timestampedConcurrencyID)
	transactionState.Mu.Lock()
	transactionState.state = Committed
	transactionState.Mu.Unlock()

	return &protos.Reply{Success: true}, nil
}

func PerformOperationPeerWrapper(peer protos.DistributedTransactionsClient, payload *protos.TransactionOpPayload) *protos.Reply {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutVal)
	defer cancel()
	resp, err := peer.PerformOperationPeer(ctx, payload)
	if err != nil {
		logrusLogger.WithField("node", currNodeName).Fatal("client.PerformOperationPeer failed: %v", err)
		return nil
	}
	return resp
}

func PerformAbortCoordinatorWrapper(coordinator protos.DistributedTransactionsClient, payload *protos.TxnIdPayload) *protos.Reply {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutVal)
	defer cancel()
	resp, err := coordinator.AbortCoordinator(ctx, payload)
	if err != nil {
		logrusLogger.WithField("node", currNodeName).Fatal("client.AbortCoordinator failed: %v", err)
		return nil
	}
	return resp
}

func (s *distributedTransactionsServer) PerformOperationCoordinator(ctx context.Context, payload *protos.TransactionOpPayload) (*protos.Reply, error) {
	s.safeTxnIDToServerInvolved.Mu.RLock()
	defer s.safeTxnIDToServerInvolved.Mu.RUnlock()
	mapOfServersInvolved := s.safeTxnIDToServerInvolved.M[payload.ID]
	(*mapOfServersInvolved)[payload.Branch] = true
	logrusLogger.WithField("node", currNodeName).Debug("Performing operation ", payload.Operation, " on branch ", payload.Branch, " for transaction ID ", payload.ID)
	peer := utils.GetClient(&s.nodeToClient, payload.Branch)

	resp := PerformOperationPeerWrapper(peer, payload)
	var success bool
	var value int32 = 0
	if resp != nil && resp.Success {
		success = true
		value = resp.Value
	} else {
		me := utils.GetClient(&s.nodeToClient, currNodeName)
		abortResult := PerformAbortCoordinatorWrapper(me, &protos.TxnIdPayload{TxnId: payload.ID})
		if abortResult == nil || !abortResult.Success {
			logrusLogger.WithField("node", currNodeName).Fatal("PerformAbortCoordinatorWrapper failed")
		}
		success = false
		if resp != nil {
			value = resp.Value
		}
	}
	return &protos.Reply{Success: success, Value: value}, nil
}

func isTsGreater(t1 string, t2 string) bool {
	return t1 > t2
}

func handleAlterAtomic(s *distributedTransactionsServer, payload *protos.TransactionOpPayload, objectState *SafeObjectState, timestampedConcurrencyID string) bool {
	var readResult, success bool
	var readBalance int32
	for {
		logrusLogger.WithField("node", currNodeName).Debug("Waiting to acquire lock on ", objectState.name)
		objectState.Mu.Lock()
		if timestampedConcurrencyID > objectState.committedTimestamp {
			var maxTs string = objectState.committedTimestamp
			var val int32 = objectState.committedVal
			for timestamp, balance := range objectState.tentativeWrites {
				if timestamp <= timestampedConcurrencyID {
					if isTsGreater(timestamp, maxTs) {
						maxTs = timestamp
					}
					val = balance
				}
			}
			if maxTs == objectState.committedTimestamp {
				if objectState.committedTimestamp == "" && strings.ToLower(payload.Operation) == "withdraw" {
					logrusLogger.WithField("node", currNodeName).Debug("Can't perform withdraw in transaction ID ", payload.ID, " on account ", objectState.name, " because account does not exist yet!")

					success = false
					break
				}
				logrusLogger.WithField("node", currNodeName).Debug("Performing read in transaction ID ", payload.ID, " on account ", objectState.name, " using committed ts: ", maxTs, " value: ", val)

				objectState.readTimestamps[timestampedConcurrencyID] = true
				readResult = true
				readBalance = val
				break
			} else if maxTs == timestampedConcurrencyID {
				logrusLogger.WithField("node", currNodeName).Debug("Performing read in transaction ID ", payload.ID, " on account ", objectState.name, " using uncommitted ts: ", maxTs, " value: ", val)
				readResult = true
				readBalance = val
				break
			} else {
				logrusLogger.WithField("node", currNodeName).Debug("Can't perform read in transaction ID ", payload.ID, " on account ", objectState.name, " yet! Will wait for commit/abort of: ", maxTs)
				objectState.Mu.Unlock()
				transactionState := GetTransactionState(&s.timestampedConcurrencyIDToState, maxTs)
				logrusLogger.WithField("node", currNodeName).Debug("Waiting for pointer to transaction state pointer")
				fmt.Printf("%p\n", transactionState)
				var state TransactionState = In_Progress
				for state == In_Progress {
					transactionState.Mu.RLock()
					// fmt.Printf("%p", transactionState)
					state = transactionState.state
					// logrusLogger.WithField("node", currNodeName).Debug(timestampedConcurrencyID, " is waiting for state of txn ID ", maxTs, " to not be ", In_Progress, ". Current state: ", state)
					transactionState.Mu.RUnlock()
					time.Sleep(500 * time.Millisecond)
				}
				logrusLogger.WithField("node", currNodeName).Debug("The transaction ID ", maxTs, " that ", timestampedConcurrencyID, " was waiting on has transitioned to state: ", state, "!")
			}
		} else {
			logrusLogger.WithField("node", currNodeName).Debug("Can't perform ", payload.Operation, "  in transaction ID ", payload.ID, " on account ", objectState.name, " because it failed comparison with committed timestamp in READ phase!")

			readResult = false
			// Abort transaction
			break
		}
	}
	if !readResult {
		success = false
	} else {
		var multiplier int32 = 1
		if strings.ToLower(payload.Operation) == "withdraw" {
			multiplier = -1
		}
		var maxReadTimestamp string = ""
		for ts := range objectState.readTimestamps {
			if isTsGreater(ts, maxReadTimestamp) {
				maxReadTimestamp = ts
			}
		}
		logrusLogger.WithField("node", currNodeName).Debug("Current transaction ID ", timestampedConcurrencyID, ", maxReadTimestamp ", maxReadTimestamp, ", committedTimestamp ", objectState.committedTimestamp, "!")
		if timestampedConcurrencyID >= maxReadTimestamp && timestampedConcurrencyID > objectState.committedTimestamp {
			// if _, ok := objectState.tentativeWrites[timestampedConcurrencyID]; !ok {
			// 	objectState.tentativeWrites[timestampedConcurrencyID] = 0
			// }
			logrusLogger.WithField("node", currNodeName).Debug("Performing ", payload.Operation, " in transaction ID ", payload.ID, " on account ", objectState.name, ". Previous balance in tentative write list: ", readBalance, ". Amount in payload: ", payload.Amount)

			objectState.tentativeWrites[timestampedConcurrencyID] = (readBalance + (multiplier * payload.Amount))
			logrusLogger.WithField("node", currNodeName).Debug("Performing ", payload.Operation, " in transaction ID ", payload.ID, " on account ", objectState.name, ". Current balance in tentative write list: ", objectState.tentativeWrites[timestampedConcurrencyID])
			success = true
		} else {
			logrusLogger.WithField("node", currNodeName).Debug("Current transaction ID ", timestampedConcurrencyID, "did not satisfy timestamped concurrency WRITE condition. ABORTING!")

			// Abort transaction
			success = false
		}
	}
	objectState.Mu.Unlock()
	return success
}

func handleRead(s *distributedTransactionsServer, payload *protos.TransactionOpPayload, objectState *SafeObjectState, timestampedConcurrencyID string) (bool, int32) {
	var success bool
	var readBalance int32
	for {
		objectState.Mu.Lock()
		if timestampedConcurrencyID > objectState.committedTimestamp {
			var maxTs string = objectState.committedTimestamp
			var val int32 = objectState.committedVal
			for timestamp, balance := range objectState.tentativeWrites {
				if timestamp <= timestampedConcurrencyID {
					if isTsGreater(timestamp, maxTs) {
						maxTs = timestamp
					}
					val = balance
				}
			}
			if maxTs == objectState.committedTimestamp {
				if objectState.committedTimestamp == "" {
					readBalance = -1
					success = false
					break
				}
				objectState.readTimestamps[timestampedConcurrencyID] = true
				logrusLogger.WithField("node", currNodeName).Debug("Performing read in transaction ID ", payload.ID, " on account ", objectState.name, " using committed ts: ", maxTs, " value: ", val)

				success = true
				readBalance = val
				break
			} else if maxTs == timestampedConcurrencyID {
				logrusLogger.WithField("node", currNodeName).Debug("Performing read in transaction ID ", payload.ID, " on account ", objectState.name, " using uncommitted ts: ", maxTs, " value: ", val)
				success = true
				readBalance = val
				break
			} else {
				// var waitChan chan bool = s.txnIDToChannel.M[maxTs]

				logrusLogger.WithField("node", currNodeName).Debug("Can't perform read in transaction ID ", payload.ID, " on account ", objectState.name, " yet! Will wait for commit/abort of: ", maxTs)
				objectState.Mu.Unlock()
				// res := <-waitChan
				transactionState := GetTransactionState(&s.timestampedConcurrencyIDToState, maxTs)
				var state TransactionState = In_Progress
				for state == In_Progress {
					transactionState.Mu.RLock()
					state = transactionState.state
					// logrusLogger.WithField("node", currNodeName).Debug("Waiting for state of txn ID ", maxTs, " to not be ", In_Progress, ". Current state: ", state)
					transactionState.Mu.RUnlock()
					time.Sleep(500 * time.Millisecond)
				}
				logrusLogger.WithField("node", currNodeName).Debug("The transaction ID ", maxTs, " that ", timestampedConcurrencyID, " was waiting on has transitioned to state: ", state, "!")
			}
		} else {
			success = false
			// Abort transaction
			break
		}
	}
	objectState.Mu.Unlock()
	return success, readBalance
}

func (s *distributedTransactionsServer) PerformOperationPeer(ctx context.Context, payload *protos.TransactionOpPayload) (*protos.Reply, error) {
	logrusLogger.WithField("node", currNodeName).Debug("Performing operation ", payload.Operation, " itself for transaction ID ", payload.ID)
	timestampedConcurrencyID := payload.ID
	var res bool
	res = GetTimestampedConcurrencyID(&s.timestampedConcurrencyIDSet, timestampedConcurrencyID)
	if !res {
		SetTimestampedConcurrencyID(&s.timestampedConcurrencyIDSet, timestampedConcurrencyID)
		InitTimestampedConcurrencyIDToStatePtr(&s.timestampedConcurrencyIDToState, timestampedConcurrencyID)
		InitSafeTimestampedConcurrencyIDToObjectsInvolved(&s.timestampedConcurrencyIDToObjectsInvolved, timestampedConcurrencyID)
	}
	objectName := payload.Account
	AddObjectToTimestampedConcurrencyIDToObjectsInvolved(&s.timestampedConcurrencyIDToObjectsInvolved, timestampedConcurrencyID, objectName)
	logrusLogger.WithField("node", currNodeName).Debug("Timestamped Concurrency ID for transaction ID ", payload.ID, " is ", timestampedConcurrencyID)

	var success bool
	var readValue int32
	var objectState *SafeObjectState
	objectState = GetObjectState(&s.objectNameToStatePtr, objectName)
	// if objectState == nil || !IsObjectReadable(&s.objectNameToStatePtr, objectName, timestampedConcurrencyID) {
	if objectState == nil {
		if strings.ToLower(payload.Operation) == "balance" || strings.ToLower(payload.Operation) == "withdraw" {
			logrusLogger.WithField("node", currNodeName).Debug("Can't perform ", strings.ToLower(payload.Operation), " in transaction ID ", payload.ID, " on account ", objectName, ". Aborting because account doesn't exist!")
			// Abort transaction
			return &protos.Reply{Success: false, Value: -1}, nil
		}
		logrusLogger.WithField("node", currNodeName).Debug("Object state for account ", objectName, " doesn't exist yet. Creating it!")
		if objectState == nil {
			objectState = &SafeObjectState{name: objectName, readTimestamps: make(map[string]bool), tentativeWrites: make(map[string]int32), committedTimestamp: ""}
			SetObjectState(&s.objectNameToStatePtr, objectName, objectState)
		}
	}

	if strings.ToLower(payload.Operation) == "deposit" || strings.ToLower(payload.Operation) == "withdraw" {
		success = handleAlterAtomic(s, payload, objectState, timestampedConcurrencyID)
	} else if strings.ToLower(payload.Operation) == "balance" {
		success, readValue = handleRead(s, payload, objectState, timestampedConcurrencyID)
	}

	return &protos.Reply{Success: success, Value: readValue}, nil
}
func (s *distributedTransactionsServer) AbortCoordinator(ctx context.Context, payload *protos.TxnIdPayload) (*protos.Reply, error) {
	mapOfServersInvolved := GetServersInvolvedInTxn(s)
	logrusLogger.WithField("node", currNodeName).Debug("Aborting transaction ", payload.TxnId, " on servers ", *mapOfServersInvolved[payload.TxnId])
	for serverName := range *mapOfServersInvolved[payload.TxnId] {
		peer := utils.GetClient(&s.nodeToClient, serverName)
		PeerWrapper(peer, payload, "AbortPeer")
	}
	return &protos.Reply{Success: true}, nil
}
func (s *distributedTransactionsServer) AbortPeer(ctx context.Context, payload *protos.TxnIdPayload) (*protos.Reply, error) {
	logrusLogger.WithField("node", currNodeName).Debug("Aborting transaction ", payload.TxnId)
	timestampedConcurrencyID := payload.TxnId

	s.objectNameToStatePtr.Mu.RLock()
	defer s.objectNameToStatePtr.Mu.RUnlock()
	for _, objectState := range s.objectNameToStatePtr.M {
		objectState.Mu.Lock()

		_, ok := objectState.tentativeWrites[timestampedConcurrencyID]
		if ok {
			delete(objectState.tentativeWrites, timestampedConcurrencyID)
		}

		_, okie := objectState.readTimestamps[timestampedConcurrencyID]
		if okie {
			delete(objectState.readTimestamps, timestampedConcurrencyID)
		}

		objectState.Mu.Unlock()
	}
	transactionState := GetTransactionState(&s.timestampedConcurrencyIDToState, timestampedConcurrencyID)
	transactionState.Mu.Lock()
	transactionState.state = Aborted
	logrusLogger.WithField("node", currNodeName).Debug("Updating transaction state pointer: ")
	fmt.Printf("%p\n", transactionState)
	transactionState.Mu.Unlock()

	return &protos.Reply{Success: true}, nil
}

func PeerWrapper(peer protos.DistributedTransactionsClient, payload *protos.TxnIdPayload, rpcType string) *protos.Reply {
	ctx, _ := context.WithTimeout(context.Background(), timeoutVal)
	var resp *protos.Reply
	var err error

	if rpcType == "AbortPeer" {
		resp, err = peer.AbortPeer(ctx, payload)
	} else if rpcType == "CommitPeer" {
		resp, err = peer.CommitPeer(ctx, payload)
	} else if rpcType == "PreparePeer" {
		resp, err = peer.PreparePeer(ctx, payload)
	}

	if err != nil {
		logrusLogger.WithField("node", currNodeName).Fatal("client.%s failed: %v", rpcType, err)
		return nil
	}
	return resp
}

func (s *distributedTransactionsServer) PreparePeer(ctx context.Context, payload *protos.TxnIdPayload) (*protos.Reply, error) {
	logrusLogger.WithField("node", currNodeName).Debug("PREPARE PHASE: Validating consistency of transaction ", payload.TxnId)
	var success bool = true
	timestampedConcurrencyID := payload.TxnId
	for _, objectState := range s.objectNameToStatePtr.M {
		objectState.Mu.RLock()

		balance, ok := objectState.tentativeWrites[timestampedConcurrencyID]
		if ok {
			if balance < 0 {
				logrusLogger.WithField("node", currNodeName).Debug("Transaction ", payload.TxnId, " has a negative balance for object ", objectState.name, ". Aborting.")
				success = false
				objectState.Mu.RUnlock()
				break
			}
		}

		objectState.Mu.RUnlock()
	}

	return &protos.Reply{Success: success}, nil
}

// Starts listening on all available unicast and anycast IP addresses
// of the local system, at the specified port.
func listenOnPort(port string) (net.Listener, error) {
	address := ":" + port
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logrusLogger.WithField("node", currNodeName).Error("Error encountered while listening on port ", port, ": ", err)
		return nil, err
	}
	return listener, nil
}

func main() {
	arguments := os.Args
	if len(arguments) != 3 {
		logrusLogger.Error("Please provide arguments as <node-name> <config-file>!")
		return
	}
	currNodeName = arguments[1]
	config := arguments[2]
	utils.InitlogrusLogger(logrusLogger, logrus.DebugLevel)

	logrusLogger.WithField("node", currNodeName).Debug("Starting node ", currNodeName, " with config file ", config)

	nodeToUrl := map[string]string{}

	utils.ParseConfigFile(config, nodeToUrl)
	var wg sync.WaitGroup

	port := strings.Split(nodeToUrl[currNodeName], ":")[1]
	listener, err := listenOnPort(port)
	if err != nil {
		log.Fatal(err)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	s := newServer()
	protos.RegisterDistributedTransactionsServer(grpcServer, s)

	for nodeName, address := range nodeToUrl {
		wg.Add(1)
		go utils.EstablishConnection(currNodeName, nodeName, address, &wg, &s.nodeToClient, logrusLogger)
	}

	grpcServer.Serve(listener)

	wg.Wait()

}
