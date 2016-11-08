package main

import ()

const (
	ALL_AUTH_TYPES = "PLAIN AMQPLAIN"
)

type Auther interface {
	Description() string
	GetUserAndPass() error
	AuthPassCorrectness() bool
	GetUserName() string
	GetUserPasswd() string
}

// new auth
func NewAuth(authType string, response string) (auth Auther) {
	switch authType {
	case "PLAIN":
		auth = &AuthPlain{response: response}
	case "AMQPLAIN":
	}

	return
}

// parse username passwd by auth type
func ParseUserNameAndPass(auth Auther) error {
	return auth.GetUserAndPass()
}

// auth correctness
func AuthCorrectness(auth Auther) bool {
	return auth.AuthPassCorrectness()
}
