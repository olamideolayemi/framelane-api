package api

type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Meta    any    `json:"meta,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
	Details any    `json:"details,omitempty"`
}

func NewSuccess(data any, meta any) Response {
	return Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	}
}

func NewError(message string, code string, details any) Response {
	return Response{
		Success: false,
		Error: &Error{
			Message: message,
			Code:    code,
			Details: details,
		},
	}
}
