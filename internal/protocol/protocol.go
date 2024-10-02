package protocol

var Magic = []byte{0x20, 0x24, 0x10, 0x01}

type ConnectionID int64

type StreamID int64

const SendBufferSize = 1024 * 1024 * 7

const ReceiveBufferSize = 1024 * 1024 * 7

const MaxPacketSize = 1452

const PacketHeaderSize = 20
