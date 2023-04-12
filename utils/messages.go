package utils

type Message struct {
	Type    string
	Payload interface{}
}

// type Transaction struct {
// 	Type          string
// 	TargetAccount string
// 	SenderNode    string
// 	CurSenderNode string
// 	Amount        int
// 	MsgID         string
// }

// type Proposal struct {
// 	ProposedPriority int
// 	ProposingNode    string
// 	MsgID            string
// }

// type Acceptance struct {
// 	AcceptedPriority int
// 	AcceptedNode     string
// 	MsgID            string
// }

// type Registration struct {
// 	NodeID string
// }
