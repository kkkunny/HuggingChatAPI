package dto

type StreamMessage struct {
	Type    StreamMessageType      `json:"type"`
	SubType *StreamMessageSubType  `json:"subtype,omitempty"` // only StreamMessageTypeTool
	UUID    *string                `json:"uuid,omitempty"`    // only StreamMessageTypeTool
	Eta     *float64               `json:"eta,omitempty"`     // only StreamMessageTypeTool && StreamMessageSubTypeEta
	Call    *StreamMessageToolCall `json:"call,omitempty"`    // only StreamMessageTypeTool && (StreamMessageSubTypeCall || StreamMessageSubTypeResult)
	Status  *StreamMessageStatus   `json:"status,omitempty"`  // only StreamMessageTypeStatus || (StreamMessageTypeTool && StreamMessageSubTypeResult)
	Token   *string                `json:"token,omitempty"`   // only StreamMessageTypeStream
	Text    *string                `json:"text,omitempty"`    // only StreamMessageTypeFinalAnswer
	Message *string                `json:"message,omitempty"` // only StreamMessageTypeStatus && StreamMessageStatusTitle
	Error   error                  `json:"-"`                 // only StreamMessageTypeError
	Name    *string                `json:"name,omitempty"`    // only StreamMessageTypeFile
	SHA     *string                `json:"sha,omitempty"`     // only StreamMessageTypeFile
	MIME    *string                `json:"mime,omitempty"`    // only StreamMessageTypeFile
}

type StreamMessageType string

const (
	StreamMessageTypeStatus      StreamMessageType = "status"
	StreamMessageTypeStream      StreamMessageType = "stream"
	StreamMessageTypeFinalAnswer StreamMessageType = "finalAnswer"
	StreamMessageTypeError       StreamMessageType = "error"
	StreamMessageTypeTool        StreamMessageType = "tool"
	StreamMessageTypeFile        StreamMessageType = "file"
	StreamMessageTypeTitle       StreamMessageType = "title"
	StreamMessageTypeReasoning   StreamMessageType = "reasoning"
)

type StreamMessageSubType string

const (
	StreamMessageSubTypeCall   StreamMessageSubType = "call"
	StreamMessageSubTypeEta    StreamMessageSubType = "eta"
	StreamMessageSubTypeResult StreamMessageSubType = "result"
)

type StreamMessageToolCall struct {
	Name       string                     `json:"name"`
	Parameters StreamMessageToolParameter `json:"parameters"`
}

type StreamMessageToolParameter struct {
	Prompt string `json:"prompt"`
	Width  string `json:"width"`
	Height string `json:"height"`
}

type StreamMessageStatus string

const (
	StreamMessageStatusStarted   StreamMessageStatus = "started"
	StreamMessageStatusTitle     StreamMessageStatus = "title"
	StreamMessageStatusSuccess   StreamMessageStatus = "success"
	StreamMessageStatusKeepAlive StreamMessageStatus = "keepAlive"
)
