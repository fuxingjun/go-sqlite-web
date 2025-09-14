package models

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func OK(data any, msg string) *Response {
	return &Response{Code: 0, Data: data, Message: msg}
}

func Err(msg string) *Response {
	return &Response{Code: -1, Error: msg}
}

func ErrWithData(msg string, data any) *Response {
	return &Response{Code: -1, Error: msg, Data: data}
}
