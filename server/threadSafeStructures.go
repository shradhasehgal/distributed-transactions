package main

import (
	"sync"
)

type TransactionState int64

const (
	Committed TransactionState = iota
	Aborted
	In_Progress
)

func (s TransactionState) String() string {
	switch s {
	case Committed:
		return "COMMITTED"
	case Aborted:
		return "ABORTED"
	case In_Progress:
		return "IN PROGRESS"
	}
	return "unknown"
}

type SafeTransactionState struct {
	Mu    sync.RWMutex
	state TransactionState
}

type SafeTimestampedConcurrencyIDToStatePtr struct {
	Mu sync.RWMutex
	M  map[uint32]*SafeTransactionState
}

func InitTimestampedConcurrencyIDToStatePtr(s *SafeTimestampedConcurrencyIDToStatePtr, timestampedConcurrencyID uint32) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.M[timestampedConcurrencyID] = &SafeTransactionState{state: In_Progress}
}

func GetTransactionState(timestampedConcurrencyIDToStatePtr *SafeTimestampedConcurrencyIDToStatePtr, timestampedConcurrencyID uint32) *SafeTransactionState {
	defer timestampedConcurrencyIDToStatePtr.Mu.RUnlock()
	timestampedConcurrencyIDToStatePtr.Mu.RLock()
	return timestampedConcurrencyIDToStatePtr.M[timestampedConcurrencyID]
}

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
	name               string
}

type SafeTxnIdToServersInvolvedPtr struct {
	Mu sync.RWMutex
	M  map[string]*(map[string]bool) // txn ID given by client to a pointer to map of server names involved in TXN
}

type SafeObjectsSet struct {
	Mu sync.RWMutex
	M  map[string]bool // object names involved in TXN
}

type SafeTimestampedConcurrencyIDToObjectsInvolved struct {
	Mu sync.RWMutex
	M  map[uint32]*SafeObjectsSet // timestampedConcurrencyID to map of object names involved in TXN
}

func InitSafeTimestampedConcurrencyIDToObjectsInvolved(s *SafeTimestampedConcurrencyIDToObjectsInvolved, timestampedConcurrencyID uint32) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.M[timestampedConcurrencyID] = &SafeObjectsSet{M: make(map[string]bool)}
}

func AddObjectToTimestampedConcurrencyIDToObjectsInvolved(s *SafeTimestampedConcurrencyIDToObjectsInvolved, timestampedConcurrencyID uint32, objectName string) {
	s.Mu.RLock()
	objectsInvolved := s.M[timestampedConcurrencyID]
	s.Mu.RUnlock()
	objectsInvolved.Mu.RLock()
	_, ok := objectsInvolved.M[objectName]
	if ok {
		objectsInvolved.Mu.RUnlock()
		return
	} else {
		objectsInvolved.Mu.RUnlock()
		objectsInvolved.Mu.Lock()
		objectsInvolved.M[objectName] = true
		objectsInvolved.Mu.Unlock()
	}
}

func GetServersInvolvedInTxn(s *distributedTransactionsServer) map[string]*(map[string]bool) {
	s.safeTxnIDToServerInvolved.Mu.RLock()
	defer s.safeTxnIDToServerInvolved.Mu.RUnlock()
	mapOfServersInvolved := s.safeTxnIDToServerInvolved.M
	return mapOfServersInvolved
}

type SafeTxnIDToChannelMap struct {
	Mu sync.RWMutex
	M  map[uint32]chan bool
}

func CreateTxnChannel(timestampedConcurrencyIDToChannel *SafeTxnIDToChannelMap, timestampedConcurrencyID uint32) {
	defer timestampedConcurrencyIDToChannel.Mu.Unlock()
	timestampedConcurrencyIDToChannel.Mu.Lock()
	txnChannel := make(chan bool, 1)
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
