package main

import (
	"errors"
)

const (
	ALL_AUTH_TYPES = "PLAIN AMQPLAIN"
)

func GetUserAndPass(authType string, response []byte) (userName string, passwd string, err error) {
	switch authType {
	case "PLAIN":
		return PlainAuthTypeGetUserNameAndPass(response)
	case "AMQPLAIN":
	}

	return "", "", errors.New("cheetahd auth error")
}

// PLAIN auth type parse username and passwd
func PlainAuthTypeGetUserNameAndPass(response []byte) (userName string, passwd string, err error) {
	userNameLength := getNextZeroPos(response[1:])
	passwdLength := getNextZeroPos(response[userNameLength+2:])
	if (userNameLength + passwdLength + 2) == len(response) {
		userName = string(response[1 : userNameLength+1])
		passwd = string(response[userNameLength+2:])
		err = nil
	} else {
		userName = ""
		passwd = ""
		err = errors.New("client send auth user and pass error")
	}

	return
}

func getNextZeroPos(response []byte) (count int) {
	for _, value := range response {
		if value == 0 {
			return
		}
		count = count + 1
	}

	return
}
