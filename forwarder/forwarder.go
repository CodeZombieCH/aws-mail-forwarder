package forwarder

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/mail"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/codezombiech/aws-mail-forwarder/config"
	"github.com/codezombiech/aws-mail-forwarder/envelope"
	"github.com/codezombiech/aws-mail-forwarder/message"
	"github.com/codezombiech/aws-mail-forwarder/sender"
	"github.com/codezombiech/aws-mail-forwarder/storage"
)

type Forwarder struct {
	config  *config.ParsedConfig
	storage *storage.Storage
	sender  *sender.Sender
}

func NewForwarder(config *config.ParsedConfig, awsConfig aws.Config) *Forwarder {
	return &Forwarder{
		config:  config,
		storage: storage.NewStorage(awsConfig, config.S3.BucketName),
		sender:  sender.NewSender(awsConfig),
	}
}

func (f *Forwarder) Forward(event events.SimpleEmailService) error {
	// For more details about the event, see
	// https://docs.aws.amazon.com/ses/latest/dg/receiving-email-notifications-contents.html#receiving-email-notifications-contents-mail-object

	messageId := event.Mail.MessageID

	if f.isSpamOrVirus(&event) {
		if err := f.markAsSpamVirus(messageId); err != nil {
			return err
		}
		return nil
	}

	transformedRecipients, err := f.transformRecipients(event.Receipt.Recipients)
	if err != nil {
		f.markAsFailed(messageId)
		return err
	}

	transformedSender, err := f.transformSender(event.Mail.CommonHeaders.From, transformedRecipients)
	if err != nil {
		f.markAsFailed(messageId)
		return err
	}

	message, err := f.fetchMessage(messageId)
	if err != nil {
		f.markAsFailed(messageId)
		return err
	}

	err = f.processMessageHeader(message.Header, transformedSender)
	if err != nil {
		f.markAsFailed(messageId)
		return err
	}

	f.setDebugHeaders(message.Header, event.Mail)

	messageBytes, err := f.buildMessage(message)
	if err != nil {
		f.markAsFailed(messageId)
		return err
	}

	err = f.sendMessage(transformedSender.String(), transformedRecipients[0].Transformed, messageId, messageBytes)
	if err != nil {
		f.markAsFailed(messageId)
		return err
	}

	err = f.markAsForwarded(messageId)
	if err != nil {
		return err
	}

	return nil
}

func (f *Forwarder) isSpamOrVirus(event *events.SimpleEmailService) bool {
	isSpamOrVirus := false

	// See https://docs.aws.amazon.com/ses/latest/dg/receiving-email-notifications-contents.html#receiving-email-notifications-contents-spamverdict-object
	if event.Receipt.SpamVerdict.Status == "FAIL" {
		log.Printf("Message marked as spam")
		isSpamOrVirus = true
	}

	// See https://docs.aws.amazon.com/ses/latest/dg/receiving-email-notifications-contents.html#receiving-email-notifications-contents-virusverdict-object
	if event.Receipt.VirusVerdict.Status == "FAIL" {
		log.Printf("Message marked as virus")
		isSpamOrVirus = true
	}

	return isSpamOrVirus
}

func (f *Forwarder) transformRecipients(recipients []string) ([]envelope.TransformationResult, error) {
	log.Print("Transforming recipients...")

	log.Printf("Original recipients: %v", recipients)

	transformedRecipients, err := envelope.TransformRecipients(f.config, recipients)
	if err != nil {
		return nil, fmt.Errorf("failed to transform recipients: %w", err)
	}

	// Check transformed recipient count
	count := 0
	for _, v := range transformedRecipients {
		count += len(v.Transformed)
	}
	if count < 1 {
		return nil, errors.New("no recipients after transformation")
	}

	log.Print("Transforming recipients succeeded")
	return transformedRecipients, nil
}

func (f *Forwarder) transformSender(senders []string, transformedRecipients []envelope.TransformationResult) (*mail.Address, error) {
	log.Printf("Original senders: %v", senders)

	transformedSender, err := envelope.TransformSenders(f.config, senders, transformedRecipients)
	if err != nil {
		return nil, fmt.Errorf("failed to transform senders: %w", err)
	}
	return transformedSender, nil
}

func (f *Forwarder) fetchMessage(mailId string) (*message.BufferedMessage, error) {
	log.Print("Fetching message...\n")
	key := f.config.S3.Incoming.NewPrefix + mailId
	messageReader, size, err := f.storage.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get message with key %s: %w", key, err)
	}
	defer messageReader.Close()

	log.Printf("Mail size is %.1f MiB", float64(size)/(1024*1024))
	if size > int64(40*1024*1024) {
		// Should most certainly never happen as the limit for incoming messages is also 40MB
		// See https://aws.amazon.com/about-aws/whats-new/2022/04/amazon-ses-v2-supports-email-size-40mb-inbound-outbound-emails-default/
		log.Print("Mail sending will most likely fail (max supported mail size is 40MB)")
	}

	mailMessage, err := mail.ReadMessage(messageReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	body, err := io.ReadAll(mailMessage.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read message body into memory: %w", err)
	}

	log.Print("Fetching message succeeded")

	return &message.BufferedMessage{
		Header: mailMessage.Header,
		Body:   body,
	}, nil
}

func (f *Forwarder) processMessageHeader(header mail.Header, newSender *mail.Address) error {
	log.Print("Processing message headers...")

	err := message.ProcessMessageHeader(f.config, header, newSender)
	if err != nil {
		return fmt.Errorf("failed to process message header: %w", err)
	}

	log.Print("Processing message headers succeeded")

	return nil
}

func (f *Forwarder) setDebugHeaders(header mail.Header, messageMetadata events.SimpleEmailMessage) {
	message.SetDebugHeaders(header, messageMetadata)
}

func (f *Forwarder) buildMessage(msg *message.BufferedMessage) ([]byte, error) {
	data, err := message.BuildMail(msg)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (f *Forwarder) sendMessage(sender string, recipientAddresses []*mail.Address, originalMessageId string, data []byte) error {
	log.Print("Sending message...")

	log.Printf("Recipients: %v", recipientAddresses)

	recipients := []string{}
	for i := 0; i < len(recipientAddresses); i++ {
		address := recipientAddresses[i].String()
		recipients = append(recipients, address)
	}

	forwardedMessageId, err := f.sender.SendMessage(sender, recipients, data)
	if err != nil {
		log.Printf("Failed to send message: %v", err)

		// Store failed outgoing mail
		key := f.config.S3.Outgoing.FailedPrefix + originalMessageId
		storeErr := f.storeMessage(key, data)
		if storeErr != nil {
			log.Printf("Failed to store failed message at %s: %v", key, storeErr)
		}

		return err
	}

	// Store succeeded outgoing mail
	key := f.config.S3.Outgoing.SentPrefix + originalMessageId
	if err = f.storeMessage(key, data); err != nil {
		return fmt.Errorf("failed to store sent message: %w", err)
	}

	log.Printf("Sending message succeeded with message ID %s", *forwardedMessageId)
	return nil
}

func (f *Forwarder) storeMessage(key string, data []byte) error {
	reader := bytes.NewReader(data)

	_, err := f.storage.Put(key, reader)
	if err != nil {
		return fmt.Errorf("failed to store message at %s: %w", key, err)
	}

	log.Printf("Stored message at %s", key)
	return nil
}

func (f *Forwarder) moveMessage(sourceKey string, targetKey string) error {
	err := f.storage.Move(sourceKey, targetKey)
	if err != nil {
		return fmt.Errorf("failed to move message from %s to %s: %w", sourceKey, targetKey, err)
	}
	log.Printf("Moved message to %s", targetKey)
	return nil
}

func (f *Forwarder) markAsFailed(messageId string) {
	err := f.storage.Move(f.config.S3.Incoming.NewPrefix+messageId, f.config.S3.Incoming.FailedPrefix+messageId)
	if err != nil {
		log.Printf("failed to mark message as failed: %v", err)
	}
}

func (f *Forwarder) markAsForwarded(messageId string) error {
	return f.moveMessage(f.config.S3.Incoming.NewPrefix+messageId, f.config.S3.Incoming.ForwardedPrefix+messageId)
}

func (f *Forwarder) markAsSpamVirus(messageId string) error {
	return f.moveMessage(f.config.S3.Incoming.NewPrefix+messageId, f.config.S3.Incoming.SpamVirusPrefix+messageId)
}
