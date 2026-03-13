package alg_ecdsa

import (
	"github.com/bnb-chain/tss-lib/tss"
)

// MessageRouter 在 TSS 各方之间路由消息（keygen/signing 的每轮消息）。
// 实现方可以是内存 channel（单机测试）或 WebSocket 中继（服务端转发给各节点）。
type MessageRouter interface {
	// Send 将某方产生的消息发给目标方。
	// fromIndex 为发送方在 PartyIDs 中的下标，msg 为 tss.Message。
	// 实现方应根据 msg.GetTo() 与 msg.IsBroadcast() 决定发给谁。
	Send(fromIndex int, msg tss.Message) error

	// Receive 向指定方投递一条收到的消息，用于 Update。
	// wireBytes 为 WireBytes() 的返回值；fromIndex 为发送方下标；isBroadcast 与 msg.IsBroadcast() 一致。
	Receive(toIndex int, wireBytes []byte, fromIndex int, isBroadcast bool) error
}

// InProcessRouter 单机多 party 时用 channel 在内存中转发消息，用于测试或 demo。
// sortedIDs 与 parties 顺序一致，用于 ParseWireMessage 时解析 From。
type InProcessRouter struct {
	Parties    []tss.Party
	SortedIDs  tss.SortedPartyIDs
	ErrCh      chan *tss.Error
	partyByKey map[string]int
}

// NewInProcessRouter 用 parties 与 sortedIDs 构建 InProcessRouter。
func NewInProcessRouter(parties []tss.Party, sortedIDs tss.SortedPartyIDs, errCh chan *tss.Error) *InProcessRouter {
	r := &InProcessRouter{Parties: parties, SortedIDs: sortedIDs, ErrCh: errCh}
	r.partyByKey = make(map[string]int)
	for i, p := range parties {
		if p != nil {
			r.partyByKey[string(p.PartyID().Key)] = i
		}
	}
	return r
}

func (r *InProcessRouter) Send(fromIndex int, msg tss.Message) error {
	bz, _, err := msg.WireBytes()
	if err != nil {
		if r.ErrCh != nil {
			r.ErrCh <- wrapPartyError(r.Parties[fromIndex], err)
		}
		return err
	}
	from := msg.GetFrom()
	isBroadcast := msg.IsBroadcast()
	to := msg.GetTo()

	if isBroadcast {
		for i := range r.Parties {
			if i == from.Index {
				continue
			}
			_ = r.Receive(i, bz, from.Index, true)
		}
	} else if len(to) > 0 {
		for _, pid := range to {
			if idx, ok := r.partyByKey[string(pid.Key)]; ok {
				_ = r.Receive(idx, bz, from.Index, false)
			}
		}
	}
	return nil
}

func (r *InProcessRouter) Receive(toIndex int, wireBytes []byte, fromIndex int, isBroadcast bool) error {
	if toIndex < 0 || toIndex >= len(r.Parties) || r.SortedIDs == nil || fromIndex >= len(r.SortedIDs) {
		return nil
	}
	party := r.Parties[toIndex]
	fromPartyID := r.SortedIDs[fromIndex]
	parsed, err := tss.ParseWireMessage(wireBytes, fromPartyID, isBroadcast)
	if err != nil {
		if r.ErrCh != nil {
			r.ErrCh <- wrapPartyError(party, err)
		}
		return err
	}
	if _, err := party.Update(parsed); err != nil && r.ErrCh != nil {
		r.ErrCh <- err
	}
	return nil
}

func wrapPartyError(party tss.Party, err error) *tss.Error {
	if e, ok := err.(*tss.Error); ok {
		return e
	}
	type wrap interface{ WrapError(error, ...*tss.PartyID) *tss.Error }
	if w, ok := party.(wrap); ok {
		return w.WrapError(err)
	}
	return tss.NewError(err, "", 0, party.PartyID())
}
