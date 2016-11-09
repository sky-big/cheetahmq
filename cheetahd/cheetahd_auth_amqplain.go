package main

import (
	"strings"
)

type AuthAmqpPlain struct {
	userName string
	passwd   string
	response string
}

func (auth *AuthAmqpPlain) Description() string {
	return "QPid AMQPLAIN mechanism"
}

func (auth *AuthAmqpPlain) SetResponse(response string) {
	auth.response = response
}

// get username and passwd
func (auth *AuthAmqpPlain) HandleResponse() int {
	responseReader := strings.NewReader(auth.response)

	table, err := readTable(responseReader)
	if err != nil {
		return AUTH_AMQPPLAIN_RESONSE_HANDLE_TEXT_ERROR
	}

	// username
	if userName, ok := table["LOGIN"]; ok {
		auth.userName = userName.(string)
	}
	// passwd
	if passwd, ok := table["PASSWORD"]; ok {
		auth.passwd = passwd.(string)
	}

	return AUTH_HANDLE_RESONSE_SUCCESS

}

func (auth *AuthAmqpPlain) AuthPassCorrectness() bool {
	return true
}

func (auth *AuthAmqpPlain) GetUserName() string {
	return auth.userName
}

func (auth *AuthAmqpPlain) GetUserPasswd() string {
	return auth.passwd
}
