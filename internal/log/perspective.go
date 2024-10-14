package log

type Perspective byte

const (
	PerspectiveClient Perspective = iota
	PerspectiveServer
)

func perspectiveToString(perspective Perspective) string {
	switch perspective {
	case PerspectiveClient:
		return "client"
	case PerspectiveServer:
		return "server"
	default:
		return "invalid"
	}
}
