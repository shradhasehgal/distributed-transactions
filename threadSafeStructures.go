package main

import (
	"encoding/gob"
	"sync"
)

type SafeEncoderMap struct {
	mu sync.RWMutex
	m  map[string]*gob.Encoder
}

func getTotalNodes(encoderMap *SafeEncoderMap) int {
	defer encoderMap.mu.RUnlock()
	encoderMap.mu.RLock()
	return len(encoderMap.m)
}

func getEncoder(encoderMap *SafeEncoderMap, nodeName string) *gob.Encoder {
	defer encoderMap.mu.RUnlock()
	encoderMap.mu.RLock()
	return encoderMap.m[nodeName]
}

func setEncoder(encoderMap *SafeEncoderMap, nodeName string, encoder *gob.Encoder) {
	defer encoderMap.mu.Unlock()
	encoderMap.mu.Lock()
	encoderMap.m[nodeName] = encoder
}

func deleteNode(encoderMap *SafeEncoderMap, nodeName string) {
	defer encoderMap.mu.Unlock()
	encoderMap.mu.Lock()
	delete(encoderMap.m, nodeName)
}

func receivedFromAllAliveNodes(encoderMap *SafeEncoderMap, receivedFrom *[]string) bool {
	defer encoderMap.mu.RUnlock()
	encoderMap.mu.RLock()
	totalAliveNodes := len(encoderMap.m)
	numReceivedFromAlive := 0
	for _, nodeName := range *receivedFrom {
		_, found := encoderMap.m[nodeName]
		// We have found a received proposal from a node that's alive (at least
		// according to our map). It may actually be dead and the handleConnection
		// goroutine may be waiting to update this in the encoderMap, but that's
		// okay. Eventually when we send acceptance messages, the node would not
		// be in encoderMap.
		if found {
			numReceivedFromAlive++
		}
	}
	return numReceivedFromAlive == totalAliveNodes
}

type SafeReceivedMap struct {
	mu sync.RWMutex
	m  map[string]int
}

func isReceived(received *SafeReceivedMap, msgId string) bool {
	defer received.mu.Unlock()
	received.mu.Lock()
	_, found := received.m[msgId]
	if !found {
		received.m[msgId] = 1
	}
	return found
}

// func markAsReceived(received *SafeReceivedMap, msgId string) {
// 	defer received.mu.Unlock()
// 	received.mu.Lock()

// }

type SafeMaxPriority struct {
	mu              sync.Mutex
	currMaxPriority int
}

func initializePriority(p *SafeMaxPriority) {
	p.mu.Lock()
	p.currMaxPriority = 1
	p.mu.Unlock()
}

func getPriorityAndIncrement(p *SafeMaxPriority) int {
	p.mu.Lock()
	me := p.currMaxPriority
	p.currMaxPriority++
	p.mu.Unlock()
	return me
}

func setPriority(p *SafeMaxPriority, value int) {
	p.mu.Lock()
	p.currMaxPriority = Max(p.currMaxPriority, value+1)
	p.mu.Unlock()
}

// type SafeMsgIDToLocalPriorityMap struct {
// 	mu sync.RWMutex
// 	m  map[string]*LocalPriority
// }

// func setMsgIDToLocalPriority(p *SafeMsgIDToLocalPriorityMap, msgID string, localPriority *LocalPriority) {
// 	defer p.mu.Unlock()
// 	p.mu.Lock()
// 	p.m[msgID] = localPriority
// }

// func getMsgIDLocalPriority(p *SafeMsgIDToLocalPriorityMap, msgID string) *LocalPriority {
// 	defer p.mu.RUnlock()
// 	p.mu.RLock()
// 	return p.m[msgID]
// }

// type SafeMsgIDToChannelMap struct {
// 	mu sync.RWMutex
// 	m  map[string]chan *Proposal
// }

// func setMsgIDToChannel(p *SafeMsgIDToChannelMap, proposal *Proposal) {
// 	defer p.mu.Unlock()
// 	p.mu.Lock()
// 	p.m[proposal.MsgID] <- proposal
// }

// func initTxn(p *SafeMsgIDToChannelMap, msgID string, channelSize int) {
// 	defer p.mu.Unlock()
// 	p.mu.Lock()
// 	p.m[msgID] = make(chan *Proposal, channelSize)
// }

// func getMsgIDChannel(p *SafeMsgIDToChannelMap, msgID string) chan *Proposal {
// 	defer p.mu.RUnlock()
// 	p.mu.RLock()
// 	return p.m[msgID]
// }

// type SafeAccountsToBalances struct {
// 	mu sync.RWMutex
// 	m  map[string]int
// }

// func printBalances(accountsToBalances *SafeAccountsToBalances, txn *Transaction, measurementsLogger *log.Logger) {
// 	measurementsLogger.Println(txn.MsgID, time.Now().UnixMicro())
// 	accounts := make([]string, 0, len(accountsToBalances.m))
// 	for k := range accountsToBalances.m {
// 		accounts = append(accounts, k)
// 	}
// 	sort.Strings(accounts)
// 	balances := "BALANCES "
// 	for _, accountName := range accounts {
// 		if accountsToBalances.m[accountName] > 0 {
// 			balances += accountName + ":" + strconv.Itoa(accountsToBalances.m[accountName]) + " "
// 		}
// 	}
// 	fmt.Println(balances)
// 	// txnLogger.Printf(balances)
// 	// if txn.Type == "deposit" {
// 	// 	logger.Printf("%s %s %d\n", txn.Type, txn.DestinationAccount, txn.Amount)
// 	// } else {
// 	// 	logger.Printf("%s %s %s %d\n", txn.Type, txn.SourceAccount, txn.DestinationAccount, txn.Amount)
// 	// }
// }

// func checkAndUpdateAccountBalance(p *SafeAccountsToBalances, txn *Transaction, measurementsLogger *log.Logger) bool {
// 	defer p.mu.Unlock()
// 	p.mu.Lock()
// 	source := txn.SourceAccount
// 	destination := txn.DestinationAccount
// 	amount := txn.Amount
// 	ret := true
// 	if amount < 0 {
// 		ret = false
// 	} else {
// 		if strings.ToLower(txn.Type) == "transfer" {
// 			_, okSource := p.m[source]
// 			// If the key does not exist
// 			if !okSource {
// 				ret = false
// 			} else if p.m[source] < amount {
// 				ret = false
// 			} else {
// 				p.m[source] -= amount
// 				_, okDestination := p.m[destination]
// 				// If the key does not exist
// 				if okDestination {
// 					p.m[destination] += amount
// 				} else {
// 					p.m[destination] = amount
// 				}
// 			}
// 		} else {
// 			_, ok := p.m[destination]
// 			if ok {
// 				p.m[destination] += amount
// 			} else {
// 				p.m[destination] = amount
// 			}
// 		}
// 	}
// 	printBalances(p, txn, measurementsLogger)
// 	return ret
// }

// type SafeMsgIDToTransaction struct {
// 	mu sync.RWMutex
// 	m  map[string]*Transaction
// }

// func setMsgIDToTransaction(p *SafeMsgIDToTransaction, msgID string, txn *Transaction) {
// 	defer p.mu.Unlock()
// 	p.mu.Lock()
// 	p.m[msgID] = txn
// }

// func getTransactionFromMsgID(p *SafeMsgIDToTransaction, msgID string) *Transaction {
// 	defer p.mu.RUnlock()
// 	p.mu.RLock()
// 	return p.m[msgID]
// }
