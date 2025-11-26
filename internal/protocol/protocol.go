package protocol

type Op string

const (
	OpSpawn   Op = "spawn"
	OpType    Op = "type"
	OpCapture Op = "capture"
	OpHistory Op = "history"
	OpWait    Op = "wait"
)

type Request struct {
	Op      Op                `json:"op"`
	ID      string            `json:"id"`
	Payload []string          `json:"payload,omitempty"` // For spawn args or text content
	Options map[string]string `json:"options,omitempty"`
}

type Response struct {
	Status  string `json:"status"` // "ok" or "error"
	Message string `json:"message,omitempty"`
	Data    string `json:"data,omitempty"` // For capture output
}
