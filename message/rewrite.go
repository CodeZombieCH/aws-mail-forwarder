package message

import (
	"log"
	"net/mail"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/codezombiech/aws-mail-forwarder/config"
)

func ProcessMessageHeader(config *config.ParsedConfig, header mail.Header, newSender *mail.Address) error {
	log.Print("Processing message headers...\n")
	fromHeader := header.Get(FromKey)

	// REPLY-TO header
	// Add "Reply-To:" with the "From" address if it doesn't already exists
	if _, exists := header[ReplyToKey]; !exists {
		setHeader(header, ReplyToKey, []string{fromHeader})
	}

	// FROM header
	setHeader(header, FromKey, []string{newSender.String()})

	// SUBJECT header
	// Add a prefix to the Subject
	if len(config.SubjectPrefix) > 0 {
		subjectHeader := header.Get(SubjectKey)
		subjectHeader = config.SubjectPrefix + subjectHeader
		setHeader(header, SubjectKey, []string{subjectHeader})
	}

	// TO header
	// TODO: we should always set a To header!
	// No sure what this silly `ToEmail` config is good for, to override the mapping???
	// Reasoning: https://github.com/arithmetric/aws-lambda-ses-forwarder/pull/46
	//
	// ==> well, as silly as it sounds, leaving the To header as is is actually a good thing:
	// while it gives a small spam score penalty, the original To header allows the recipient to
	// know to what address the forwarded email was originally sent to.
	// SES does not seem to care of the original To does not match the actual forwarded recipients email address
	//
	// Example:
	// From:				sender@example.com ==> config.from or public@example.com
	// To:					public@example.com ==> public@example.com
	// Actual recipient:	                       private@example.com

	// Replace original 'To' header with a manually defined one
	if len(config.ToEmail) > 0 {
		setHeader(header, ToKey, []string{config.ToEmail})
	}

	// Remove the Return-Path header
	removeHeader(header, ReturnPathKey)

	// Remove Sender header
	removeHeader(header, SenderKey)

	// Remove Message-ID header
	removeHeader(header, MessageIdKey)

	// Remove all DKIM-Signature headers to prevent triggering an
	// "InvalidParameterValue: Duplicate header 'DKIM-Signature'" error.
	// These signatures will likely be invalid anyways, since the From
	// header was modified.
	for k := range header {
		if strings.HasSuffix(k, "Dkim-Signature") {
			removeHeader(header, k)
		}
	}

	log.Print("Processing message headers succeeded\n")

	return nil
}

func SetDebugHeaders(header mail.Header, messageMetadata events.SimpleEmailMessage) {
	// Add debugging headers
	setHeader(header, "X-Forwarder-Message-Id", []string{messageMetadata.MessageID}) // The unique ID assigned to the email by Amazon SES
	setHeader(header, "X-Forwarder-Original-From", messageMetadata.CommonHeaders.From)
	setHeader(header, "X-Forwarder-Function-Name", []string{os.Getenv("AWS_LAMBDA_FUNCTION_NAME")})
}

func setHeader(header mail.Header, key string, values []string) {
	header[key] = values
	log.Printf("Setting header %v: %v", key, values)
}

func removeHeader(header mail.Header, key string) {
	delete(header, key)
	log.Printf("Removing header %v", key)
}
