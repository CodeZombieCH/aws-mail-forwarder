package sender

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/aws/smithy-go"
)

type Sender struct {
	sesClient *sesv2.Client
}

func NewSender(awsConfig aws.Config) *Sender {
	return &Sender{
		sesClient: sesv2.NewFromConfig(awsConfig),
	}
}

func (s *Sender) SendMessage(source string, destinations []string, data []byte) (*string, error) {
	input := sesv2.SendEmailInput{
		FromEmailAddress: aws.String(source),
		Destination: &types.Destination{
			ToAddresses: destinations,
		},
		Content: &types.EmailContent{
			Raw: &types.RawMessage{
				Data: data,
			},
		},
	}

	output, err := s.sesClient.SendEmail(context.TODO(), &input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			log.Printf("code: %s, message: %s, fault: %s\n", apiErr.ErrorCode(), apiErr.ErrorMessage(), apiErr.ErrorFault().String())
			return nil, fmt.Errorf(
				"failed to send message (code: %s, message: %s, fault: %s)",
				apiErr.ErrorCode(), apiErr.ErrorMessage(), apiErr.ErrorFault().String(),
			)
		}
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return output.MessageId, nil
}
