package main

import ()

type amqp_error struct {
	name        string
	explanation string
	method      message
}

func NewAmapError(name string, explanationStr string, method message) *amqp_error {
	return &amqp_error{
		name:        name,
		explanation: explanationStr,
		method:      method,
	}
}

func MakeExceptionFrame(channel uint16, amqpErrorInfo *amqp_error) frame {
	var classId, methodId uint16
	var message message

	shouldClose, Code, Text := GetAmqpErrorInfo(amqpErrorInfo.name)
	exlanationStr := MakeErrorText(Text, amqpErrorInfo.explanation)

	if amqpErrorInfo.method != nil {
		classId, methodId = amqpErrorInfo.method.id()
	} else {
		classId, methodId = 0, 0
	}

	// make close frame
	if shouldClose || channel == 0 {
		message = &connectionClose{
			ReplyCode: Code,
			ReplyText: exlanationStr,
			ClassId:   classId,
			MethodId:  methodId,
		}
	} else {
		message = &channelClose{
			ReplyCode: Code,
			ReplyText: exlanationStr,
			ClassId:   classId,
			MethodId:  methodId,
		}
	}
	return &methodFrame{
		ChannelId: channel,
		ClassId:   classId,
		MethodId:  methodId,
		Method:    message,
	}
}

func MakeErrorText(text string, exlanationStr string) string {
	resultStr := text + "-" + exlanationStr
	if len(resultStr) > 255 {
		return resultStr[:252] + "..."
	}

	return resultStr
}
