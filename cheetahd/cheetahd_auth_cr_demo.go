package main

type AuthCrDemo struct {
	userName string
	passwd   string
	response string
}

func (auth *AuthCrDemo) Description() string {
	return "Demo challenge-response authentication"
}

func (auth *AuthCrDemo) SetResponse(response string) {
	auth.response = response
}

// get username and passwd
func (auth *AuthCrDemo) HandleResponse() int {
	if auth.userName == "" {
		auth.userName = auth.response
		return AUTH_CR_DEMO_RESPONSE_HANDLE_CHALLENGE
	}

	auth.passwd = auth.response

	return AUTH_HANDLE_RESONSE_SUCCESS
}

func (auth *AuthCrDemo) AuthPassCorrectness() bool {
	return true
}

func (auth *AuthCrDemo) GetUserName() string {
	return auth.userName
}

func (auth *AuthCrDemo) GetUserPasswd() string {
	return auth.passwd
}
