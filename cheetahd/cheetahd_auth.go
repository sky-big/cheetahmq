package main

import ()

const (
	ALL_AUTH_TYPES                           = "PLAIN AMQPLAIN"
	SUCCESS_AUTH                             = 1
	AUTH_HANDLE_RESONSE_SUCCESS              = 2
	AUTH_PLAIN_RESPONSE_HANDLE_TEXT_ERROR    = 3
	AUTH_AMQPPLAIN_RESONSE_HANDLE_TEXT_ERROR = 4
	AUTH_CR_DEMO_RESPONSE_HANDLE_CHALLENGE   = 5
)

type Auther interface {
	Description() string
	HandleResponse() int
	AuthPassCorrectness() bool
	GetUserName() string
	GetUserPasswd() string
	SetResponse(string)
}

// new auth
func NewAuth(authType string) (auth Auther) {
	switch authType {
	case "PLAIN":
		auth = &AuthPlain{}
	case "AMQPLAIN":
		auth = &AuthAmqpPlain{}
	default:
		auth = &AuthCrDemo{}
	}

	return
}
