package models

import (
	"icooclaw/pkg/bus"
)

type SenderInfo = bus.SenderInfo

type OutboundMediaMessage = bus.OutboundMediaMessage

type OutboundMessage = bus.OutboundMessage

type InboundMessage = bus.InboundMessage

func BusToOutMessage(msg bus.OutboundMessage) OutboundMessage {
	return msg
}
