package common

type Chunk struct {
	ID    int    `json:"id"`
	Total int    `json:"total"`
	Data  string `json:"data"`
	CRC32 uint32 `json:"crc32"`
}
