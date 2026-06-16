package output

import "encoding/json"

type Envelope struct {
	OK     bool        `json:"ok"`
	Data   interface{} `json:"data,omitempty"`
	Notice *Notice     `json:"_notice,omitempty"`
}

type ErrorEnvelope struct {
	OK     bool       `json:"ok"`
	Error  ErrorInfo  `json:"error"`
	Notice *Notice    `json:"_notice,omitempty"`
}

type ErrorInfo struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

type Notice struct {
	Update *UpdateNotice `json:"update,omitempty"`
	Skills *SkillsNotice `json:"skills,omitempty"`
}

type UpdateNotice struct {
	Current string `json:"current"`
	Latest  string `json:"latest"`
	Message string `json:"message"`
	Command string `json:"command"`
}

type SkillsNotice struct {
	Current string `json:"current"`
	Target  string `json:"target"`
	Message string `json:"message"`
	Command string `json:"command"`
}

func NewEnvelope(data interface{}, notice *Notice) Envelope {
	return Envelope{OK: true, Data: data, Notice: notice}
}

func NewErrorEnvelope(errType, message, hint string, notice *Notice) ErrorEnvelope {
	return ErrorEnvelope{
		OK:     false,
		Error:  ErrorInfo{Type: errType, Message: message, Hint: hint},
		Notice: notice,
	}
}

func (e Envelope) Marshal() []byte {
	b, _ := json.Marshal(e)
	return b
}

func (e ErrorEnvelope) Marshal() []byte {
	b, _ := json.Marshal(e)
	return b
}

func (n *Notice) HasAny() bool {
	return n != nil && (n.Update != nil || n.Skills != nil)
}