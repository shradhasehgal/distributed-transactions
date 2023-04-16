package main

import (
	"sync"
)

type TimestampVal struct {
	value     int32
	timestamp uint32
}
type SafeObjectState struct {
	Mu                 sync.RWMutex
	readTimestamps     map[uint32]bool
	tentativeWrites    map[uint32]int32
	committedVal       int32
	committedTimestamp uint32
	maxReadTimestamp   uint32
	name               string
}

type SafeTxnIdToServersInvolvedPtr struct {
	Mu sync.RWMutex
	M  map[string]*([]string) // txn ID given by client to a pointer to list of server names involved in TXN
}

func GetServersInvolvedInTxn(s *distributedTransactionsServer) map[string]*([]string) {
	s.safeTxnIDToServerInvolved.Mu.RLock()
	defer s.safeTxnIDToServerInvolved.Mu.RUnlock()
	listOfServersInvolved := s.safeTxnIDToServerInvolved.M
	return listOfServersInvolved
}

type SafeTxnIDToChannelMap struct {
	Mu sync.RWMutex
	M  map[uint32]chan bool
}

func CreateTxnChannel(timestampedConcurrencyIDToChannel *SafeTxnIDToChannelMap, timestampedConcurrencyID uint32) {
	defer timestampedConcurrencyIDToChannel.Mu.Unlock()
	timestampedConcurrencyIDToChannel.Mu.Lock()
	txnChannel := make(chan bool)
	timestampedConcurrencyIDToChannel.M[timestampedConcurrencyID] = txnChannel
}

type SafeObjectNameToStatePtr struct {
	Mu sync.RWMutex
	M  map[string]*SafeObjectState // Object name t
}

func GetObjectState(objectNameToStatePtr *SafeObjectNameToStatePtr, objectName string) *SafeObjectState {
	defer objectNameToStatePtr.Mu.RUnlock()
	objectNameToStatePtr.Mu.RLock()
	if val, ok := objectNameToStatePtr.M[objectName]; ok {
		return val
	}
	return nil
}

func SetObjectState(objectNameToStatePtr *SafeObjectNameToStatePtr, objectName string, objectState *SafeObjectState) {
	defer objectNameToStatePtr.Mu.Unlock()
	objectNameToStatePtr.Mu.Lock()
	objectNameToStatePtr.M[objectName] = objectState
}

type SafeTxnIDToTimestampedConcurrencyID struct {
	Mu sync.RWMutex
	M  map[string]uint32 // txn ID given by client to the transaction ID used during timestamped concurrency
}

func GetTimestampedConcurrencyID(txnIDToTimestampedConcurrencyID *SafeTxnIDToTimestampedConcurrencyID, txnID string) (bool, uint32) {
	defer txnIDToTimestampedConcurrencyID.Mu.Unlock()
	txnIDToTimestampedConcurrencyID.Mu.Lock()
	if val, ok := txnIDToTimestampedConcurrencyID.M[txnID]; ok {
		return true, val
	}
	return false, 0
}

func SetTimestampedConcurrencyID(txnIDToTimestampedConcurrencyID *SafeTxnIDToTimestampedConcurrencyID, txnID string, timestampedConcurrencyID *uint32) uint32 {
	defer txnIDToTimestampedConcurrencyID.Mu.Unlock()
	txnIDToTimestampedConcurrencyID.Mu.Lock()
	*timestampedConcurrencyID++
	txnIDToTimestampedConcurrencyID.M[txnID] = *timestampedConcurrencyID
	return *timestampedConcurrencyID
}
