package protocol

type Op string

const (
	OpType    Op = "type"
	OpCapture Op = "capture"
	OpHistory Op = "history"
	OpWait    Op = "wait"
	OpKill    Op = "kill"
)

type Request struct {
	Op      Op                `json:"op"`
	Payload []string          `json:"payload,omitempty"`
	Options map[string]string `json:"options,omitempty"`
}

type Response struct {
	Status  string `json:"status"` // "ok" or "error"
	Message string `json:"message,omitempty"`
	Data    string `json:"data,omitempty"` // For capture output
}
