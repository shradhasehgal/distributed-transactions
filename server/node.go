package main

import (
	"context"
	"log"
	"math"
	"net"
	"os"
	"protos"
	"strings"
	"sync"
	"time"
	"utils"

	logrus "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type distributedTransactionsServer struct {
	protos.UnimplementedDistributedTransactionsServer
	timestampedConcurrencyID        uint32
	nodeToClient                    utils.SafeRPCClientMap
	safeTxnIDToServerInvolved       SafeTxnIdToServersInvolvedPtr
	txnIDToTimestampedConcurrencyID SafeTxnIDToTimestampedConcurrencyID
	objectNameToStatePtr            SafeObjectNameToStatePtr
	txnIDToChannel                  SafeTxnIDToChannelMap
}

func newServer() *distributedTransactionsServer {
	s := &distributedTransactionsServer{
		nodeToClient:                    utils.SafeRPCClientMap{M: make(map[string]protos.DistributedTransactionsClient)},
		safeTxnIDToServerInvolved:       SafeTxnIdToServersInvolvedPtr{M: make(map[string]*(map[string]bool))},
		txnIDToTimestampedConcurrencyID: SafeTxnIDToTimestampedConcurrencyID{M: make(map[string]uint32)},
		timestampedConcurrencyID:        0,
		objectNameToStatePtr:            SafeObjectNameToStatePtr{M: make(map[string]*SafeObjectState)},
		txnIDToChannel:                  SafeTxnIDToChannelMap{M: make(map[uint32]chan bool)}}
	return s
}

var currNodeName string

var logrusLogger = logrus.New()

func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// Creates a log file named <log_type>_<node_name>.log and returns a thread-safe
// logger instance that can be used to append to the log file using logger.Println().
func getLogger(nodeName string, logType string) (*os.File, *log.Logger, error) {
	fileName := logType + "_" + nodeName + ".log"
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrusLogger.WithField("node", currNodeName).Error("Error encountered while creating file ", fileName, ": ", err)
		return nil, nil, err
	}

	logger := log.New(f, "", 0)
	return f, logger, nil
}

func (s *distributedTransactionsServer) BeginTransaction(ctx context.Context, payload *protos.TxnIdPayload) (*protos.Reply, error) {
	s.safeTxnIDToServerInvolved.Mu.Lock()
	defer s.safeTxnIDToServerInvolved.Mu.Unlock()
	// var listOfServersInvolved = []string{}
	var mapOfServersInvolved = make(map[string]bool)
	s.safeTxnIDToServerInvolved.M[payload.TxnId] = &mapOfServersInvolved
	return &protos.Reply{Success: true}, nil
}

func (s *distributedTransactionsServer) CommitCoordinator(ctx context.Context, payload *protos.TxnIdPayload) (*protos.Reply, error) {
	mapOfServersInvolved := GetServersInvolvedInTxn(s)
	passing := true
	for serverName, _ := range *mapOfServersInvolved[payload.TxnId] {
		peer := utils.GetClient(&s.nodeToClient, serverName)
		vote := PeerWrapper(peer, payload, "PreparePeer")
		if vote != nil && vote.Success == false {
			passing = false
			// return &protos.Reply{Success: false}, nil
		}
	}

	for serverName, _ := range *mapOfServersInvolved[payload.TxnId] {
		peer := utils.GetClient(&s.nodeToClient, serverName)
		if !passing {
			PeerWrapper(peer, payload, "AbortPeer")
		} else {
			PeerWrapper(peer, payload, "CommitPeer")
		}
	}

	if !passing {
		return &protos.Reply{Success: false}, nil
	}

	return &protos.Reply{Success: true}, nil
}

func (s *distributedTransactionsServer) CommitPeer(ctx context.Context, payload *protos.TxnIdPayload) (*protos.Reply, error) {
	logrusLogger.WithField("node", currNodeName).Debug("Committing transaction ID ", payload.TxnId)
	timestampedConcurrencyID := s.txnIDToTimestampedConcurrencyID.M[payload.TxnId]
	commitChan := s.txnIDToChannel.M[timestampedConcurrencyID]
	commitChan <- true
	return &protos.Reply{Success: true}, nil
}

func PerformOperationPeerWrapper(peer protos.DistributedTransactionsClient, payload *protos.TransactionOpPayload) *protos.Reply {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resp, err := peer.PerformOperationPeer(ctx, payload)
	if err != nil {
		logrusLogger.WithField("node", currNodeName).Fatal("client.PerformOperationPeer failed: %v", err)
		return nil
	}
	return resp
}

func (s *distributedTransactionsServer) PerformOperationCoordinator(ctx context.Context, payload *protos.TransactionOpPayload) (*protos.Reply, error) {
	s.safeTxnIDToServerInvolved.Mu.RLock()
	defer s.safeTxnIDToServerInvolved.Mu.RUnlock()
	mapOfServersInvolved := s.safeTxnIDToServerInvolved.M[payload.ID]
	// *mapOfServersInvolved = append(*listOfServersInvolved, payload.Branch)
	(*mapOfServersInvolved)[payload.Branch] = true
	logrusLogger.WithField("node", currNodeName).Debug("Performing operation ", payload.Operation, " on branch ", payload.Branch, " for transaction ID ", payload.ID)
	peer := utils.GetClient(&s.nodeToClient, payload.Branch)
	if peer == nil {
		return &protos.Reply{Success: false, Value: 0}, nil
	}
	resp := PerformOperationPeerWrapper(peer, payload)
	var success bool
	var value int32
	if resp != nil && resp.Success {
		success = true
		value = resp.Value
	} else {
		success = false
		value = 0
	}
	return &protos.Reply{Success: success, Value: value}, nil
}

func handleBalanceAlterCommand(s *distributedTransactionsServer, payload *protos.TransactionOpPayload, objectState *SafeObjectState, timestampedConcurrencyID uint32) bool {
	var success bool
	readResult, readValue := handleRead(s, payload, objectState, timestampedConcurrencyID)
	if !readResult {
		success = false
	} else {
		objectState.Mu.Lock()
		var multiplier int32 = 1
		if strings.ToLower(payload.Operation) == "withdraw" {
			multiplier = -1
		}
		var maxReadTimestamp uint32 = 0
		for ts := range objectState.readTimestamps {
			maxReadTimestamp = uint32(math.Max(float64(ts), float64(maxReadTimestamp)))
		}
		if timestampedConcurrencyID >= maxReadTimestamp && timestampedConcurrencyID > objectState.committedTimestamp {
			if _, ok := objectState.tentativeWrites[timestampedConcurrencyID]; !ok {
				objectState.tentativeWrites[timestampedConcurrencyID] = 0
			}
			logrusLogger.WithField("node", currNodeName).Debug("Performing ", payload.Operation, " in transaction ID ", payload.ID, " on account ", objectState.name, ". Previous balance in tentative write list: ", readValue, ". Amount in payload: ", payload.Amount)

			objectState.tentativeWrites[timestampedConcurrencyID] = (readValue + (multiplier * payload.Amount))
			logrusLogger.WithField("node", currNodeName).Debug("Performing ", payload.Operation, " in transaction ID ", payload.ID, " on account ", objectState.name, ". Current balance in tentative write list: ", objectState.tentativeWrites[timestampedConcurrencyID])
			success = true
		} else {
			// Abort transaction
			success = false
		}
		objectState.Mu.Unlock()
	}
	return success
}

func handleRead(s *distributedTransactionsServer, payload *protos.TransactionOpPayload, objectState *SafeObjectState, timestampedConcurrencyID uint32) (bool, int32) {
	var success bool
	var readBalance int32
	for {
		objectState.Mu.Lock()
		if timestampedConcurrencyID > objectState.committedTimestamp {
			var maxTs uint32 = objectState.committedTimestamp
			var val int32 = objectState.committedVal
			for timestamp, balance := range objectState.tentativeWrites {
				if timestamp <= timestampedConcurrencyID {
					maxTs = uint32(math.Max(float64(maxTs), float64(timestamp)))
					val = balance
				}
			}
			if maxTs == objectState.committedTimestamp {
				logrusLogger.WithField("node", currNodeName).Debug("Performing read in transaction ID ", payload.ID, " on account ", objectState.name, " using committed ts: ", maxTs, " value: ", val)
				objectState.readTimestamps[timestampedConcurrencyID] = true
				success = true
				readBalance = val
				break
			} else if maxTs == timestampedConcurrencyID {
				logrusLogger.WithField("node", currNodeName).Debug("Performing read in transaction ID ", payload.ID, " on account ", objectState.name, " using uncommitted ts: ", maxTs, " value: ", val)
				success = true
				readBalance = val
				break
			} else {
				var waitChan chan bool = s.txnIDToChannel.M[maxTs]
				logrusLogger.WithField("node", currNodeName).Debug("Can't perform read in transaction ID ", payload.ID, " on account ", objectState.name, " yet! Will wait for commit/abort of: ", maxTs)
				objectState.Mu.Unlock()
				res := <-waitChan
				if res {
					logrusLogger.WithField("node", currNodeName).Debug("The transaction ID ", maxTs, " that ", timestampedConcurrencyID, " was waiting on has been: COMMITTED!")
				} else {
					logrusLogger.WithField("node", currNodeName).Debug("The transaction ID ", maxTs, " that ", timestampedConcurrencyID, " was waiting on has been: ABORTED!")
				}

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
	txnID := payload.ID
	objectName := payload.Account
	var success bool
	var readValue int32
	var objectState *SafeObjectState
	objectState = GetObjectState(&s.objectNameToStatePtr, objectName)
	if objectState == nil {
		if strings.ToLower(payload.Operation) == "balance" || strings.ToLower(payload.Operation) == "withdraw" {
			logrusLogger.WithField("node", currNodeName).Debug("Can't perform ", strings.ToLower(payload.Operation), " in transaction ID ", payload.ID, " on account ", objectName, ". Aborting because account doesn't exist!")
			// Abort transaction
			return &protos.Reply{Success: false}, nil
		}
		objectState = &SafeObjectState{name: objectName, readTimestamps: make(map[uint32]bool), tentativeWrites: make(map[uint32]int32), committedTimestamp: 0}
		SetObjectState(&s.objectNameToStatePtr, objectName, objectState)
	}
	var timestampedConcurrencyID uint32
	var res bool
	res, timestampedConcurrencyID = GetTimestampedConcurrencyID(&s.txnIDToTimestampedConcurrencyID, txnID)
	if !res {
		timestampedConcurrencyID = SetTimestampedConcurrencyID(&s.txnIDToTimestampedConcurrencyID, txnID, &s.timestampedConcurrencyID)
		CreateTxnChannel(&s.txnIDToChannel, timestampedConcurrencyID)
	}
	logrusLogger.WithField("node", currNodeName).Debug("Timestamped Concurrency ID for transaction ID ", payload.ID, " is ", timestampedConcurrencyID)

	if strings.ToLower(payload.Operation) == "deposit" || strings.ToLower(payload.Operation) == "withdraw" {
		success = handleBalanceAlterCommand(s, payload, objectState, timestampedConcurrencyID)
	} else if strings.ToLower(payload.Operation) == "balance" {
		success, readValue = handleRead(s, payload, objectState, timestampedConcurrencyID)
	}

	return &protos.Reply{Success: success, Value: readValue}, nil
}
func (s *distributedTransactionsServer) AbortCoordinator(ctx context.Context, payload *protos.TxnIdPayload) (*protos.Reply, error) {
	mapOfServersInvolved := GetServersInvolvedInTxn(s)
	logrusLogger.WithField("node", currNodeName).Debug("Aborting transaction ", payload.TxnId, " on servers ", *mapOfServersInvolved[payload.TxnId])
	for serverName, _ := range *mapOfServersInvolved[payload.TxnId] {
		peer := utils.GetClient(&s.nodeToClient, serverName)
		PeerWrapper(peer, payload, "AbortPeer")
	}
	return &protos.Reply{Success: true}, nil
}
func (s *distributedTransactionsServer) AbortPeer(ctx context.Context, payload *protos.TxnIdPayload) (*protos.Reply, error) {
	logrusLogger.WithField("node", currNodeName).Debug("Aborting transaction ", payload.TxnId)

	timestampedConcurrencyID := s.txnIDToTimestampedConcurrencyID.M[payload.TxnId]
	commitChan := s.txnIDToChannel.M[timestampedConcurrencyID]
	commitChan <- true

	s.objectNameToStatePtr.Mu.RLock()
	defer s.objectNameToStatePtr.Mu.RLock()
	for _, objectState := range s.objectNameToStatePtr.M {
		objectState.Mu.Lock()
		s.txnIDToTimestampedConcurrencyID.Mu.RLock()
		timestampedConcurrencyID := s.txnIDToTimestampedConcurrencyID.M[payload.TxnId]
		s.txnIDToTimestampedConcurrencyID.Mu.RUnlock()

		_, ok := objectState.tentativeWrites[timestampedConcurrencyID]
		if ok {
			delete(objectState.tentativeWrites, timestampedConcurrencyID)
		}

		_, okie := objectState.readTimestamps[timestampedConcurrencyID]
		if okie {
			delete(objectState.tentativeWrites, timestampedConcurrencyID)
		}

		objectState.Mu.Unlock()
	}
	return &protos.Reply{Success: true}, nil
}

func PeerWrapper(peer protos.DistributedTransactionsClient, payload *protos.TxnIdPayload, rpcType string) *protos.Reply {
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
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

// func OldPeerWrapper(peer protos.DistributedTransactionsClient, payload *protos.CommitPayload) *protos.Reply {
// 	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
// 	resp, err := peer.PreparePeer(ctx, payload)
// 	if err != nil {
// 		logrusLogger.WithField("node", currNodeName).Fatal("client.PreparePeer failed: %v", err)
// 		return nil
// 	}
// 	return resp
// }

func (s *distributedTransactionsServer) PreparePeer(ctx context.Context, payload *protos.TxnIdPayload) (*protos.Reply, error) {
	logrusLogger.WithField("node", currNodeName).Debug("Preparing transaction ", payload.TxnId)
	return &protos.Reply{Success: true}, nil
}

// func handleTxnMsg(msg *Message, received *SafeReceivedMap, nodeToEncoder *SafeEncoderMap, p *SafeMaxPriority, msgIDToLocalPriority *SafeMsgIDToLocalPriorityMap, safeIsisPq *SafePriorityQueue, msgIDToTransaction *SafeMsgIDToTransaction) {
// 	txn, _ := msg.Payload.(Transaction)
// 	var msgID = txn.MsgID
// 	if !isReceived(received, msgID) {
// 		var maxPo = getPriorityAndIncrement(p)
// 		var proposalMsg = Message{}
// 		proposalMsg.Type = "proposal"
// 		proposalMsg.Payload = &Proposal{ProposedPriority: maxPo, ProposingNode: currNodeName, MsgID: msgID}
// 		logrusLogger.WithField("node", currNodeName).Debug("Received message with ID ", msgID, " from ", txn.CurSenderNode)
// 		logrusLogger.WithField("node", currNodeName).Debug("Unicasting proposal message with priority", maxPo, " to ", txn.SenderNode, "for TxN with message ID: ", msgID)
// 		var localPriority = &LocalPriority{Priority: maxPo, nodeID: currNodeName, msgID: msgID, isAccepted: false}
// 		setMsgIDToTransaction(msgIDToTransaction, msgID, &txn)

// 		safeIsisPq.Push(localPriority)
// 		setMsgIDToLocalPriority(msgIDToLocalPriority, msgID, localPriority)
// 		unicast(getEncoder(nodeToEncoder, txn.SenderNode), proposalMsg)
// 		if txn.SenderNode != currNodeName {
// 			txn.CurSenderNode = currNodeName
// 			msg.Payload = txn
// 			logrusLogger.WithField("node", currNodeName).Debug("Resending message with ID ", msgID)
// 			basicMulticast(nodeToEncoder, msg)
// 		}
// 	}
// }

// func handleProposalMsg(msg *Message, msgToChannel *SafeMsgIDToChannelMap) {
// 	proposal, _ := msg.Payload.(Proposal)
// 	logrusLogger.WithField("node", currNodeName).Debug("Received proposal message from %s", proposal.ProposingNode, " with priority %d", proposal.ProposedPriority, "for message ID: ", proposal.MsgID)
// 	setMsgIDToChannel(msgToChannel, &proposal)
// }

// func handleAcceptMsg(msg *Message, msgIDToLocalPriority *SafeMsgIDToLocalPriorityMap, safeIsisPq *SafePriorityQueue, deliveryLogger *log.Logger, msgIDtoTransaction *SafeMsgIDToTransaction, accountsToBalances *SafeAccountsToBalances, txnLogger *log.Logger, nodeToEncoder *SafeEncoderMap, measurementsLogger *log.Logger) {
// 	acceptance, _ := msg.Payload.(Acceptance)
// 	var acceptedMsgID = acceptance.MsgID
// 	var acceptedPriority = acceptance.AcceptedPriority
// 	var acceptedNode = acceptance.AcceptedNode
// 	logrusLogger.WithField("node", currNodeName).Debug("Received acceptance message with priority ", acceptedPriority, ".", acceptedNode, " for messageID: ", acceptedMsgID)

// 	safeIsisPq.Update(getMsgIDLocalPriority(msgIDToLocalPriority, acceptedMsgID), acceptedPriority, acceptedNode)
// 	safeIsisPq.HandleDeliverable(deliveryLogger, msgIDtoTransaction, accountsToBalances, txnLogger, nodeToEncoder, measurementsLogger)
// }

// func handleConnection(connection net.Conn, received *SafeReceivedMap, nodeToEncoder *SafeEncoderMap, p *SafeMaxPriority, msgToChannel *SafeMsgIDToChannelMap, msgIDToLocalPriority *SafeMsgIDToLocalPriorityMap, safeIsisPq *SafePriorityQueue, deliveryLogger *log.Logger, msgIDtoTransaction *SafeMsgIDToTransaction, accountsToBalances *SafeAccountsToBalances, txnLogger *log.Logger, measurementsLogger *log.Logger) {
// func handleConnection(connection net.Conn, nodeToEncoder *utils.SafeEncoderMap) {

// 	defer connection.Close()
// 	logrusLogger.WithField("node", currNodeName).Debug("Connection from ", connection.RemoteAddr().String())
// 	// decoder := gob.NewDecoder(connection)
// 	// connectingNode := ""
// 	for {
// 		// var msg Message
// 		// err := decoder.Decode(&msg)
// 		// if err != nil {
// 		// 	if err == io.EOF {
// 		// 		logrusLogger.WithField("node", currNodeName).Debug(connectingNode, " has failed!")
// 		// 		deleteNode(nodeToEncoder, connectingNode)
// 		// 		return
// 		// 	}
// 		// 	logrusLogger.WithField("node", currNodeName).Error("Decode error: ", err)
// 		// } else {
// 		// 	if msg.Type == "transaction" {
// 		// 		handleTxnMsg(&msg, received, nodeToEncoder, p, msgIDToLocalPriority, safeIsisPq, msgIDtoTransaction)
// 		// 	} else if msg.Type == "proposal" {
// 		// 		handleProposalMsg(&msg, msgToChannel)
// 		// 	} else if msg.Type == "acceptance" {
// 		// 		handleAcceptMsg(&msg, msgIDToLocalPriority, safeIsisPq, deliveryLogger, msgIDtoTransaction, accountsToBalances, txnLogger, nodeToEncoder, measurementsLogger)
// 		// 	} else if msg.Type == "registration" {
// 		// 		registration, _ := msg.Payload.(Registration)
// 		// 		connectingNode = registration.NodeID
// 		// 	}

// 		// }

// 	}
// }

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

// func sendMsg(encoder *gob.Encoder, msg Message, wg *sync.WaitGroup) {
// 	defer wg.Done()
// 	err := encoder.Encode(msg)
// 	if err != nil {
// 		logrusLogger.WithField("node", currNodeName).Error("Encode error: ", err)
// 	}
// }

// func unicast(encoder *gob.Encoder, msg Message) {
// 	err := encoder.Encode(msg)
// 	if err != nil {
// 		logrusLogger.WithField("node", currNodeName).Error("Encode error: 	", err)
// 	}
// }

// func proposalGreater(p1 *Proposal, p2 *Proposal) bool {
// 	p1Node, _ := strconv.Atoi(p1.ProposingNode[4:])
// 	p2Node, _ := strconv.Atoi(p2.ProposingNode[4:])
// 	var p1Priority = p1.ProposedPriority
// 	var p2Priority = p2.ProposedPriority
// 	return (p1Priority > p2Priority) || (p1Priority == p2Priority && p1Node > p2Node)
// }

// func processTxn(nodeToEncoder *SafeEncoderMap, msg Message, proposals chan *Proposal, totalNodes int) {
// 	var msgID = msg.Payload.(Transaction).MsgID

// 	go basicMulticast(nodeToEncoder, &msg)
// 	var maxProposal = &Proposal{ProposedPriority: 0, ProposingNode: "node0"}
// 	var breakOut = false
// 	var receivedFrom []string
// 	for {
// 		select {
// 		case proposal := <-proposals:
// 			receivedFrom = append(receivedFrom, proposal.ProposingNode)
// 			if proposalGreater(proposal, maxProposal) {
// 				maxProposal = proposal
// 			}
// 			/*
// 				Imagine there are total 3 nodes -> A, B, C. We have received
// 				proposals from A & B. Suppose we enter this loop having received B's proposal,
// 				B dies when we enter the loop and nodeToEncoder now has only A & C.
// 				When we enter receivedFromAllAliveNodes, we will get False because
// 				numReceivedFromAlive will be 1 (for A) and totalAliveNodes will 2 (For A & C).
// 				totalAliveNodes will not change when we're in receivedFromAllAliveNodes.
// 			*/
// 			if receivedFromAllAliveNodes(nodeToEncoder, &receivedFrom) {
// 				breakOut = true
// 				break
// 			}
// 		case <-time.After(15 * time.Second):
// 			logrusLogger.WithField("node", currNodeName).Debug("Hit timeout while waiting for proposals on message ID: ", msgID)
// 			breakOut = true
// 			break
// 		}
// 		if breakOut {
// 			break
// 		}
// 	}
// 	logrusLogger.WithField("node", currNodeName).Debug("Accepted proposal with priority ", maxProposal.ProposedPriority, " from ", maxProposal.ProposingNode, "for message ID ", maxProposal.MsgID)

// 	var AcceptanceMsg = Message{}
// 	AcceptanceMsg.Type = "acceptance"
// 	var accepted = Acceptance{AcceptedPriority: maxProposal.ProposedPriority, AcceptedNode: maxProposal.ProposingNode, MsgID: maxProposal.MsgID}
// 	AcceptanceMsg.Payload = accepted

// 	go basicMulticast(nodeToEncoder, &AcceptanceMsg)
// }

// func basicMulticast(nodeToEncoder *SafeEncoderMap, msg *Message) {
// 	var wg sync.WaitGroup
// 	defer nodeToEncoder.mu.RUnlock()
// 	nodeToEncoder.mu.RLock()
// 	for _, encoder := range nodeToEncoder.m {
// 		wg.Add(1)
// 		go sendMsg(encoder, *msg, &wg)
// 	}
// 	wg.Wait()
// }

// func initGob() {
// 	gob.Register(Registration{})
// 	gob.Register(Transaction{})
// 	gob.Register(Proposal{})
// 	gob.Register(Acceptance{})
// }

func main() {
	arguments := os.Args
	if len(arguments) != 3 {
		logrusLogger.Error("Please provide arguments as <node-name> <config-file>!")
		return
	}
	currNodeName = arguments[1]
	config := arguments[2]
	// initGob()
	utils.InitlogrusLogger(logrusLogger)

	logrusLogger.WithField("node", currNodeName).Debug("Starting node ", currNodeName, " with config file ", config)

	// txnId := 0
	nodeToUrl := map[string]string{}
	// received := SafeReceivedMap{m: make(map[string]int)}

	// msgToChannel := SafeMsgIDToChannelMap{m: make(map[string](chan *Proposal))}
	// msgIDToTransaction := SafeMsgIDToTransaction{m: make(map[string]*Transaction)}
	// accountsToBalances := SafeAccountsToBalances{m: make(map[string]int)}

	// p := SafeMaxPriority{currMaxPriority: 1}
	// msgIDToLocalPriority := SafeMsgIDToLocalPriorityMap{m: make(map[string]*LocalPriority)}
	// safeIsisPq := SafePriorityQueue{}
	// safeIsisPq.pq = make(PriorityQueue, 0)

	// heap.Init(&safeIsisPq.pq)
	// f1, deliveryLogger, _ := getLogger(currNodeName, "delivery")
	// f2, txnLogger, _ := getLogger(currNodeName, "txn")
	// f3, measurementsLogger, _ := getLogger(currNodeName, "measurements")

	// logrusLogger.WithField("node", currNodeName).Debug("Total Nodes to connect to: ", totalNodes)

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

	// scanner := bufio.NewScanner(os.Stdin)
	// var txn = Transaction{}
	// var msg = Message{}

	// for scanner.Scan() {
	// 	command := scanner.Text()
	// 	commandInfo := strings.Fields(command)
	// 	if len(commandInfo) == 0 {
	// 		continue
	// 	}
	// 	if strings.ToLower(commandInfo[0]) == "deposit" {
	// 		if len(commandInfo) != 3 {
	// 			continue
	// 		}

	// 		txn.Type = commandInfo[0]
	// 		txn.SourceAccount = ""
	// 		txn.DestinationAccount = commandInfo[1]
	// 		txn.Amount, _ = strconv.Atoi(commandInfo[2])

	// 	} else if strings.ToLower(commandInfo[0]) == "transfer" {
	// 		if len(commandInfo) != 5 {
	// 			continue
	// 		}

	// 		txn.Type = commandInfo[0]
	// 		txn.SourceAccount = commandInfo[1]
	// 		txn.DestinationAccount = commandInfo[3]
	// 		txn.Amount, _ = strconv.Atoi(commandInfo[4])
	// 	}
	// 	txnId++
	// 	txn.CurSenderNode = currNodeName
	// 	txn.SenderNode = currNodeName
	// 	txn.MsgID = strconv.Itoa(txnId) + "_" + currNodeName
	// 	msg.Type = "transaction"
	// 	msg.Payload = txn
	// 	initTxn(&msgToChannel, txn.MsgID, totalNodes)
	// 	measurementsLogger.Println(txn.MsgID, time.Now().UnixMicro())
	// 	go processTxn(&nodeToEncoder, msg, getMsgIDChannel(&msgToChannel, txn.MsgID), totalNodes)

	// }
	// if err := scanner.Err(); err != nil {
	// 	logrusLogger.WithField("node", currNodeName).Error("Scanner error: ", err)
	// }
	// f1.Close()
	// f2.Close()
	// f3.Close()
}
