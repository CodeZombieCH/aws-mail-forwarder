//go:build integration

package sender

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
)

type testConfig struct {
	awsConfig      aws.Config
	verifiedDomain string
}

func getTestConfig(t *testing.T) testConfig {
	checkEnv(t, "AWS_REGION")
	checkEnv(t, "AWS_PROFILE")
	checkEnv(t, "VERIFIED_DOMAIN")

	awsConfig, err := awsconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		t.Fatalf("Failed to load AWS config: %v", err)
	}

	return testConfig{
		awsConfig:      awsConfig,
		verifiedDomain: os.Getenv("VERIFIED_DOMAIN"),
	}
}

func checkEnv(t *testing.T, envName string) {
	if os.Getenv(envName) == "" {
		t.Fatalf("Environment variable %s must be set", envName)
	}
}

func TestSendMessage(t *testing.T) {
	testConfig := getTestConfig(t)

	sender := fmt.Sprintf("sender%s", testConfig.verifiedDomain)
	recipient := fmt.Sprintf("recipient%s", testConfig.verifiedDomain)
	message := `Foo: bar
From: <<<sender>>>
To: <<<recipient>>>
Subject: Test email (contains an attachment)
MIME-Version: 1.0
Content-type: Multipart/Mixed; boundary="NextPart"


--NextPart
Content-Type: text/plain

This is the message body.

--NextPart
Content-Type: text/plain;
Content-Disposition: attachment; filename="attachment.txt"

This is the text in the attachment.

--NextPart--
`
	message = strings.Replace(message, "<<<sender>>>", sender, -1)
	message = strings.Replace(message, "<<<recipient>>>", recipient, -1)
	// Replace with line delimiter according to RFC5322
	message = strings.Replace(message, "\n", "\r\n", -1)

	s := NewSender(testConfig.awsConfig)
	messageId, err := s.SendMessage(sender, []string{recipient}, []byte(message))
	if err != nil {
		t.Fatal(err)
	}

	if messageId == nil {
		t.Fatalf("Message ID should not be nil")
	}
}
