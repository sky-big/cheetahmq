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

// get username and passwd
func (auth *AuthAmqpPlain) GetUserAndPass() error {
	responseReader := strings.NewReader(auth.response)

	table, err := readTable(responseReader)
	if err != nil {
		return err
	}

	// username
	if userName, ok := table["LOGIN"]; ok {
		auth.userName = userName.(string)
	}
	// passwd
	if passwd, ok := table["PASSWORD"]; ok {
		auth.passwd = passwd.(string)
	}

	return nil

}

func (auth *AuthAmqpPlain) AuthPassCorrectness() bool {
	return false
}

func (auth *AuthAmqpPlain) GetUserName() string {
	return auth.userName
}

func (auth *AuthAmqpPlain) GetUserPasswd() string {
	return auth.passwd
}
