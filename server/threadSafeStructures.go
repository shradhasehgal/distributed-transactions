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
	M  map[string]*SafeTransactionState
}

func InitTimestampedConcurrencyIDToStatePtr(s *SafeTimestampedConcurrencyIDToStatePtr, timestampedConcurrencyID string) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.M[timestampedConcurrencyID] = &SafeTransactionState{state: In_Progress}
}

func GetTransactionState(timestampedConcurrencyIDToStatePtr *SafeTimestampedConcurrencyIDToStatePtr, timestampedConcurrencyID string) *SafeTransactionState {
	defer timestampedConcurrencyIDToStatePtr.Mu.RUnlock()
	timestampedConcurrencyIDToStatePtr.Mu.RLock()
	return timestampedConcurrencyIDToStatePtr.M[timestampedConcurrencyID]
}

type SafeObjectState struct {
	Mu                 sync.RWMutex
	readTimestamps     map[string]bool
	tentativeWrites    map[string]int32
	committedVal       int32
	committedTimestamp string
	name               string
	isCommitted        bool
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
	M  map[string]*SafeObjectsSet // timestampedConcurrencyID to map of object names involved in TXN
}

func InitSafeTimestampedConcurrencyIDToObjectsInvolved(s *SafeTimestampedConcurrencyIDToObjectsInvolved, timestampedConcurrencyID string) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.M[timestampedConcurrencyID] = &SafeObjectsSet{M: make(map[string]bool)}
}

func GetSafeTimestampedConcurrencyIDToObjectsInvolved(s *SafeTimestampedConcurrencyIDToObjectsInvolved, timestampedConcurrencyID string) *SafeObjectsSet {
	defer s.Mu.RUnlock()
	s.Mu.RLock()
	return s.M[timestampedConcurrencyID]
}

func AddObjectToTimestampedConcurrencyIDToObjectsInvolved(s *SafeTimestampedConcurrencyIDToObjectsInvolved, timestampedConcurrencyID string, objectName string) {
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

func SetObjectCommitted(s *SafeObjectNameToStatePtr, objectName string) {
	s.Mu.RLock()
	objectState := s.M[objectName]
	s.Mu.RUnlock()
	objectState.Mu.RLock()
	if objectState.isCommitted {
		objectState.Mu.RUnlock()
		return
	} else {
		objectState.Mu.RUnlock()
		objectState.Mu.Lock()
		objectState.isCommitted = true
		objectState.Mu.Unlock()
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

func IsObjectReadable(objectNameToStatePtr *SafeObjectNameToStatePtr, objectName string, timestampedConcurrencyID string) bool {
	defer objectNameToStatePtr.Mu.RUnlock()
	objectNameToStatePtr.Mu.RLock()
	if val, ok := objectNameToStatePtr.M[objectName]; ok {
		val.Mu.RLock()
		defer val.Mu.RUnlock()
		if val.committedTimestamp != "" {
			return true
		} else {
			if _, ok := val.tentativeWrites[timestampedConcurrencyID]; ok {
				return true
			} else {
				return false
			}
		}
	}
	return false
}

func SetObjectState(objectNameToStatePtr *SafeObjectNameToStatePtr, objectName string, objectState *SafeObjectState) {
	defer objectNameToStatePtr.Mu.Unlock()
	objectNameToStatePtr.Mu.Lock()
	objectNameToStatePtr.M[objectName] = objectState
}

type SafeTimestampedConcurrencyIDSet struct {
	Mu sync.RWMutex
	M  map[string]bool
}

func GetTimestampedConcurrencyID(txnIDToTimestampedConcurrencyID *SafeTimestampedConcurrencyIDSet, txnID string) bool {
	defer txnIDToTimestampedConcurrencyID.Mu.RUnlock()
	txnIDToTimestampedConcurrencyID.Mu.RLock()
	if _, ok := txnIDToTimestampedConcurrencyID.M[txnID]; ok {
		return true
	}
	return false
}

func SetTimestampedConcurrencyID(txnIDToTimestampedConcurrencyID *SafeTimestampedConcurrencyIDSet, txnID string) {
	defer txnIDToTimestampedConcurrencyID.Mu.Unlock()
	txnIDToTimestampedConcurrencyID.Mu.Lock()
	txnIDToTimestampedConcurrencyID.M[txnID] = true
}
