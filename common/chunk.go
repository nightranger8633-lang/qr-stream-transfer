package common

type Chunk struct {
	ID        int    `json:"id"`
	Total     int    `json:"total"`
	Data      string `json:"data"`
	CRC32     string `json:"crc32"`
	Timestamp int64  `json:"timestamp"`
}
