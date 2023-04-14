package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"protos"
	"strings"
	"sync"
	"utils"

	logrus "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type SafeTxnIdToServersInvolvedPtr struct {
	Mu sync.RWMutex
	M  map[string]*([]string) // txn ID given by client to a pointer to list of server names involved in TXN
}

type distributedTransactionsServer struct {
	protos.UnimplementedDistributedTransactionsServer
	txnID                     uint64
	safeTxnIDToServerInvolved SafeTxnIdToServersInvolvedPtr
}

func newServer() *distributedTransactionsServer {
	s := &distributedTransactionsServer{safeTxnIDToServerInvolved: SafeTxnIdToServersInvolvedPtr{M: make(map[string]*([]string))}}
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

func (s *distributedTransactionsServer) BeginTransaction(ctx context.Context, payload *protos.BeginTxnPayload) (*protos.Reply, error) {
	s.safeTxnIDToServerInvolved.Mu.Lock()
	defer s.safeTxnIDToServerInvolved.Mu.Unlock()
	fmt.Printf("yoyo")
	var listOfServersInvolved = []string{}
	s.safeTxnIDToServerInvolved.M[payload.TxnId] = &listOfServersInvolved
	return &protos.Reply{Success: true}, nil
}

func (s *distributedTransactionsServer) CommitCoordinator(ctx context.Context, payload *protos.CommitPayload) (*protos.Reply, error) {
	return &protos.Reply{Success: true}, nil
}

func (s *distributedTransactionsServer) CommitPeer(ctx context.Context, payload *protos.CommitPayload) (*protos.Reply, error) {
	return &protos.Reply{Success: true}, nil
}

func (s *distributedTransactionsServer) PerformOperationCoordinator(ctx context.Context, payload *protos.TransactionOpPayload) (*protos.Reply, error) {
	s.safeTxnIDToServerInvolved.Mu.RLock()
	defer s.safeTxnIDToServerInvolved.Mu.RUnlock()
	listOfServersInvolved := s.safeTxnIDToServerInvolved.M[payload.ID]
	*listOfServersInvolved = append(*listOfServersInvolved, payload.Branch)
	logrusLogger.WithField("node", currNodeName).Debug("Performing operation ", payload.Operation, " on branch ", payload.Branch, " for transaction ID ", payload.ID)
	return &protos.Reply{Success: true}, nil
}

func (s *distributedTransactionsServer) PerformOperationPeer(ctx context.Context, payload *protos.TransactionOpPayload) (*protos.Reply, error) {
	return &protos.Reply{Success: true}, nil
}
func (s *distributedTransactionsServer) AbortCoordinator(ctx context.Context, payload *protos.AbortPayload) (*protos.Reply, error) {
	return &protos.Reply{Success: true}, nil
}
func (s *distributedTransactionsServer) AbortPeer(ctx context.Context, payload *protos.AbortPayload) (*protos.Reply, error) {
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
	nodeToClient := utils.SafeRPCClientMap{M: make(map[string]protos.DistributedTransactionsClient)}

	s := strings.Split(nodeToUrl[currNodeName], ":")
	listener, err := listenOnPort(s[1])
	if err != nil {
		log.Fatal(err)
	}

	for nodeName, address := range nodeToUrl {
		wg.Add(1)
		go utils.EstablishConnection(currNodeName, nodeName, address, &wg, &nodeToClient, logrusLogger)
	}
	wg.Wait()

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	protos.RegisterDistributedTransactionsServer(grpcServer, newServer())
	grpcServer.Serve(listener)
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
