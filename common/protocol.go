package common

type PacketType string

const (
	PacketTypeControl PacketType = "control"
	PacketTypeChunk   PacketType = "chunk"
)

type ControlCmd string

const (
	ControlStart ControlCmd = "START"
	ControlEnd   ControlCmd = "END"
)

type Packet struct {
	Type      PacketType  `json:"type"`
	Command   ControlCmd  `json:"command,omitempty"`
	SessionID string      `json:"session_id,omitempty"`
	FileName  string      `json:"file_name,omitempty"`
	Chunk     *Chunk      `json:"chunk,omitempty"`
	Meta      *PacketMeta `json:"meta,omitempty"`
}

type PacketMeta struct {
	TotalChunks int   `json:"total_chunks"`
	FileSize    int64 `json:"file_size"`
	Timestamp   int64 `json:"timestamp"`
}
