package message

import (
	"net/mail"
	"strings"
)

// The canonical format of the MIME header keys
const (
	FromKey       = "From"
	ToKey         = "To"
	CcKey         = "Cc"
	ReplyToKey    = "Reply-To"
	SubjectKey    = "Subject"
	SenderKey     = "Sender"
	MessageIdKey  = "Message-Id"
	ReturnPathKey = "Return-Path"
)

// Line delimiter according to RFC5322
// See https://www.rfc-editor.org/rfc/rfc5322#section-2.1
const RFC5322LineDelimiter = "\r\n"

// A parsed mail message with fully buffered message body
type BufferedMessage struct {
	Header mail.Header
	Body   []byte
}

func toRFC5322LineDelimiter(message string) string {
	return strings.Replace(message, "\n", RFC5322LineDelimiter, -1)
}
