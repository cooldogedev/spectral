package frame

const (
	IDAcknowledgement = iota

	IDConnectionRequest
	IDConnectionResponse
	IDConnectionClose

	IDStreamRequest
	IDStreamResponse
	IDStreamData
	IDStreamClose

	IDMTURequest
	IDMTUResponse
)
