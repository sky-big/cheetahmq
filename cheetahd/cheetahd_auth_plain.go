package main

import ()

type AuthPlain struct {
	userName string
	passwd   string
	response string
}

func (auth *AuthPlain) Description() string {
	return "SASL PLAIN authentication mechanism"
}

func (auth *AuthPlain) SetResponse(response string) {
	auth.response = response
}

// PLAIN auth type parse username and passwd
func (auth *AuthPlain) HandleResponse() int {
	response := []byte(auth.response)

	if len(response) > 2 {
		userNameLength := getNextZeroPos(response[1:])
		if (userNameLength + 2) < len(response) {
			passwdLength := getNextZeroPos(response[userNameLength+2:])
			if (userNameLength + passwdLength + 2) == len(response) {
				auth.userName = string(response[1 : userNameLength+1])
				auth.passwd = string(response[userNameLength+2:])
				return AUTH_HANDLE_RESONSE_SUCCESS
			} else {
				auth.userName = ""
				auth.passwd = ""
				return AUTH_PLAIN_RESPONSE_HANDLE_TEXT_ERROR
			}
		} else {
			auth.userName = ""
			auth.passwd = ""
			return AUTH_PLAIN_RESPONSE_HANDLE_TEXT_ERROR
		}
	} else {
		auth.userName = ""
		auth.passwd = ""
		return AUTH_PLAIN_RESPONSE_HANDLE_TEXT_ERROR
	}

	return AUTH_PLAIN_RESPONSE_HANDLE_TEXT_ERROR
}

func (auth *AuthPlain) AuthPassCorrectness() bool {
	return true
}

func (auth *AuthPlain) GetUserName() string {
	return auth.userName
}

func (auth *AuthPlain) GetUserPasswd() string {
	return auth.passwd
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
