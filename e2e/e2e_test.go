//go:build e2e

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/aws/smithy-go"
	"github.com/codezombiech/aws-mail-forwarder/message"
)

type testMailAppResponse struct {
	Result  string                      `json:"result"`
	Message string                      `json:"message"`
	Count   int                         `json:"count"`
	Limit   int                         `json:"limit"`
	Offset  int                         `json:"offset"`
	Emails  []testMailAppEmailsResponse `json:"emails"`
}

type testMailAppEmailsResponse struct {
	DownloadUrl string `json:"downloadUrl"`
	Id          string `json:"id"`
}

func sendTestMail(t *testing.T, config testConfig, timestamp int64) {
	from := fmt.Sprintf("e2e Test Sender <%s>", config.sender)
	to := []string{fmt.Sprintf("e2e Test Recipient <%s>", config.recipient)}
	cc := []string{}
	bcc := []string{}

	input := sesv2.SendEmailInput{
		FromEmailAddress: aws.String(from),
		Destination: &types.Destination{
			ToAddresses:  to,
			CcAddresses:  cc,
			BccAddresses: bcc,
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data: aws.String(fmt.Sprintf("Test subject %d", timestamp)),
				},
				Body: &types.Body{
					Text: &types.Content{
						Data: aws.String("Test body"),
					},
				},
			},
		},
	}

	sesClient := sesv2.NewFromConfig(config.awsConfig)

	_, err := sesClient.SendEmail(context.TODO(), &input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			t.Fatalf("code: %s, message: %s, fault: %s", apiErr.ErrorCode(), apiErr.ErrorMessage(), apiErr.ErrorFault().String())
		} else {
			t.Fatalf(err.Error())
		}
	}
}

func receiveTestMail(t *testing.T, config testConfig, timestamp int64) *mail.Message {
	// Wait for new mail to arrive
	client := http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf(
		"https://api.testmail.app/api/json?apikey=%s&namespace=%s&tag=%s&livequery=true&timestamp_from=%d&pretty=true",
		config.testMailAppApiKey,
		config.testMailAppNamespace,
		config.testMailAppTag,
		timestamp),
	)
	if err != nil {
		t.Fatalf("Failed to receive test mail: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Receiving test mail failed with HTTP status code %v", resp.StatusCode)
	}

	bodyRaw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	var body testMailAppResponse
	if err := json.Unmarshal(bodyRaw, &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Pick first mail
	mailResponse := body.Emails[0]
	t.Logf("Received mail with ID %s", mailResponse.Id)

	// Download mail as EML file
	resp, err = http.Get(mailResponse.DownloadUrl)
	if err != nil {
		t.Fatalf("Failed to download mail: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Retrieving test mail failed with HTTP status code %v", resp.StatusCode)
	}

	message, err := mail.ReadMessage(resp.Body)
	if err != nil {
		t.Fatalf("Failed to parse mail: %v", err)
	}

	return message
}

func validateMail(t *testing.T, config testConfig, mail *mail.Message, timestamp int64) {
	if expected, actual := fmt.Sprintf("FORWARDER: Test subject %d", timestamp), mail.Header.Get(message.SubjectKey); expected != actual {
		t.Errorf("expected %s, got %s", expected, actual)
	}

	if expected, actual := fmt.Sprintf("\"e2e Test Sender at %s\" <%s>", config.sender, config.recipient), mail.Header.Get(message.FromKey); expected != actual {
		t.Errorf("expected %s, got %s", expected, actual)
	}

	if expected, actual := fmt.Sprintf("e2e Test Recipient <%s>", config.recipient), mail.Header.Get(message.ToKey); expected != actual {
		t.Errorf("expected %s, got %s", expected, actual)
	}
}

type testConfig struct {
	sender               string
	recipient            string
	awsConfig            aws.Config
	testMailAppApiKey    string
	testMailAppNamespace string
	testMailAppTag       string
	addresses            testConfigAddresses
}

type testConfigAddresses struct {
	sender    string
	forwarder string
}

func getTestConfig(t *testing.T) testConfig {
	checkEnv(t, "SENDER")
	checkEnv(t, "RECIPIENT")
	checkEnv(t, "AWS_REGION")
	checkEnv(t, "TESTMAILAPP_APIKEY")
	checkEnv(t, "TESTMAILAPP_NAMESPACE")
	checkEnv(t, "TESTMAILAPP_TAG")

	awsConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		t.Fatalf("Failed to load AWS config: %v", err)
	}

	return testConfig{
		sender:               os.Getenv("SENDER"),
		recipient:            os.Getenv("RECIPIENT"),
		awsConfig:            awsConfig,
		testMailAppApiKey:    os.Getenv("TESTMAILAPP_APIKEY"),
		testMailAppNamespace: os.Getenv("TESTMAILAPP_NAMESPACE"),
		testMailAppTag:       os.Getenv("TESTMAILAPP_TAG"),
	}
}

func checkEnv(t *testing.T, envName string) {
	if os.Getenv(envName) == "" {
		t.Fatalf("Environment variable %s must be set", envName)
	}
}

func TestForwarding(t *testing.T) {
	testConfig := getTestConfig(t)

	timestamp := time.Now().UTC().UnixMilli()

	t.Logf("Sending mail...")
	sendTestMail(t, testConfig, timestamp)
	t.Logf("Sending mail completed")

	t.Logf("Receiving mail...")
	message := receiveTestMail(t, testConfig, timestamp)
	t.Logf("Receiving mail completed")

	t.Logf("Validating mail...")
	validateMail(t, testConfig, message, timestamp)
	t.Logf("Validating mail completed")
}
