package envelope

import (
	"fmt"
	"log"
	"net/mail"
	"regexp"
	"strings"

	"github.com/codezombiech/aws-mail-forwarder/config"
)

type TransformationResult struct {
	Source      *mail.Address
	Transformed []*mail.Address
}

// Transform the original senders to a single new sender for the new message to send
func TransformSenders(config *config.ParsedConfig, senders []string, transformations []TransformationResult) (*mail.Address, error) {
	// SES does not allow sending messages from an unverified address,
	// so change the sender of the forwarded message to the original
	// recipient (which is a verified domain)

	// Parse sender addresses
	senderAddresses := make([]*mail.Address, 0)
	for _, sender := range senders {
		senderAddress, err := mail.ParseAddress(sender)
		if err != nil {
			return nil, fmt.Errorf("invalid sender address %v: %w", sender, err)
		}
		senderAddresses = append(senderAddresses, senderAddress)
	}

	// Calculate address part
	var addressPart string
	if len(config.FromEmail) > 0 {
		addressPart = config.FromEmail
	} else {
		// There might me multiple original recipients
		// For the sake of simplicity, we take the first one
		addressPart = transformations[0].Source.Address
	}

	// Calculate name part
	// There might me multiple original senders
	// For the sake of simplicity, we take the first one
	var namePart string
	fromAddress := senderAddresses[0]
	if fromAddress.Name != "" {
		namePart = fmt.Sprintf("%s at %s", fromAddress.Name, fromAddress.Address)
	} else {
		namePart = fromAddress.Address
	}

	return &mail.Address{
		Name:    namePart,
		Address: addressPart,
	}, nil
}

// Transform the original recipients to new recipients based on the configured mapping
func TransformRecipients(config *config.ParsedConfig, recipients []string) ([]TransformationResult, error) {
	// Parse recipient addresses
	recipientAddresses := make([]*mail.Address, 0)
	for _, recipient := range recipients {
		recipientAddress, err := mail.ParseAddress(recipient)
		if err != nil {
			return nil, fmt.Errorf("invalid recipient address %v: %w", recipient, err)
		}
		recipientAddresses = append(recipientAddresses, recipientAddress)
	}

	transformations := make([]TransformationResult, 0)

	for _, recipient := range recipientAddresses {
		mappingsForRecipient := make([]*mail.Address, 0)

		// TODO: Check if it is smart to be case insensitive => At least document it!
		// According to specs user part can be case sensitive:
		// - https://stackoverflow.com/a/9808332/548020
		// - https://www.rfc-editor.org/rfc/rfc5321#section-2.3.11
		recipientAddress := strings.ToLower(recipient.Address)

		if config.AllowPlusSign {
			log.Printf("Replacing + sign from recipient %v", recipientAddress)
			var re = regexp.MustCompile(`\+.*?@`)
			recipientAddress = re.ReplaceAllString(recipientAddress, `@`)
			log.Printf("Replaced + sign to %v", recipientAddress)
		}

		if mapping, ok := config.ForwardMapping[recipientAddress]; ok {
			// Exact match
			mappingsForRecipient = append(mappingsForRecipient, mapping...)
		} else {
			// Test for partial matches

			// TODO: Check if it would be better to replace the matching strategy by regex?

			localPart, domain, err := splitAddress(recipientAddress)
			if err != nil {
				return nil, err
			}

			if mapping, ok := config.ForwardMapping["@"+domain]; ok {
				// domain match, e.g. "@example.com"
				mappingsForRecipient = append(mappingsForRecipient, mapping...)
			} else if mapping, ok := config.ForwardMapping[localPart]; ok {
				// local part match, e.g. "info"
				mappingsForRecipient = append(mappingsForRecipient, mapping...)
			} else if mapping, ok := config.ForwardMapping["@"]; ok {
				// Wildcard match, "@"
				mappingsForRecipient = append(mappingsForRecipient, mapping...)
			}
		}

		transformations = append(transformations, TransformationResult{
			Source:      recipient,
			Transformed: mappingsForRecipient,
		})
	}

	return transformations, nil
}

func splitAddress(address string) (string, string, error) {
	at := strings.LastIndex(address, "@")
	if at >= 0 {
		localPart, domain := address[:at], address[at+1:]
		return localPart, domain, nil
	} else {
		log.Printf("Invalid address %s", address)
		return "", "", fmt.Errorf("failed to split address %s", address)
	}
}
