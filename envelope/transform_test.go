package envelope

import (
	"net/mail"
	"testing"

	"github.com/codezombiech/aws-mail-forwarder/config"
	"github.com/google/go-cmp/cmp"
)

func getRawConfig() *config.RawConfig {
	return &config.RawConfig{
		FromEmail:     "noreply@example.com",
		SubjectPrefix: "Prefix: ",
		AllowPlusSign: true,
		S3: config.S3Config{
			BucketName: "s3-bucket-name",
			Incoming: config.S3IncomingConfig{
				NewPrefix:       "in/new/",
				SpamVirusPrefix: "in/spam-virus/",
				ForwardedPrefix: "in/forwarded/",
				FailedPrefix:    "in/failed/",
			},
			Outgoing: config.S3OutgoingConfig{
				SentPrefix:   "out/sent/",
				FailedPrefix: "out/failed/",
			},
		},
		ForwardMapping: map[string][]string{
			"info@example.com": {
				"example.john@example.com",
				"example.jen@example.com",
			},
			"abuse@example.com": {
				"example.jim@example.com",
			},
			"@example.com": {
				"domain-match@example.com",
			},
			"info": {
				"local-name-match@example.com",
			},
		},
	}
}

func getConfig() *config.ParsedConfig {
	parsedConfig, _ := config.ParseConfig(getRawConfig())
	return parsedConfig
}

func parseAddresses(recipients []string, t *testing.T) []*mail.Address {
	parsedRecipients := make([]*mail.Address, 0)
	for _, r := range recipients {
		parsed, err := mail.ParseAddress(r)
		if err != nil {
			t.Fatalf("Failed to parse address %v\n", r)
		}
		parsedRecipients = append(parsedRecipients, parsed)
	}
	return parsedRecipients
}

func parseAddress(recipient string, t *testing.T) *mail.Address {
	parsed, err := mail.ParseAddress(recipient)
	if err != nil {
		t.Fatalf("Failed to parse address %v\n", recipient)
	}
	return parsed
}

func toStringAddresses(recipients []*mail.Address) []string {
	recipientsAsString := make([]string, 0)
	for i := 0; i < len(recipients); i++ {
		recipientsAsString = append(recipientsAsString, recipients[i].Address)
	}
	return recipientsAsString
}

func TestTransformRecipientsExactMatch(t *testing.T) {
	config := getConfig()

	transformed, err := TransformRecipients(config, []string{
		"info@example.com",
	})
	if err != nil {
		t.Fatalf("transformation failed: %v\n", err)
	}

	t.Log(transformed)
	if expected, actual := 2, len(transformed[0].Transformed); expected != actual {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func TestTransformRecipientsDomainMatch(t *testing.T) {
	config := getConfig()

	transformed, err := TransformRecipients(config, []string{
		"domain-match@example.com",
	})
	if err != nil {
		t.Fatalf("transformation failed: %v\n", err)
	}

	t.Log(transformed)
	if expected, actual := 1, len(transformed[0].Transformed); expected != actual {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func TestTransformRecipientsLocalNameMatch(t *testing.T) {
	config := getConfig()

	transformed, err := TransformRecipients(config, []string{
		"info@foo.bar",
	})
	if err != nil {
		t.Fatalf("transformation failed: %v\n", err)
	}

	t.Log(transformed)
	if expected, actual := 1, len(transformed[0].Transformed); expected != actual {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func TestTransformRecipientsPlusSign(t *testing.T) {
	config := getConfig()

	transformed, err := TransformRecipients(config, []string{
		"abuse+me@example.com",
	})
	if err != nil {
		t.Fatalf("transformation failed: %v\n", err)
	}

	if expected, actual := []string{"example.jim@example.com"}, toStringAddresses(transformed[0].Transformed); !cmp.Equal(expected, actual) {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func TestTransformRecipientsPriorities(t *testing.T) {
	rawConfig := config.RawConfig{
		ForwardMapping: map[string][]string{
			"info@example.com": {
				"full-match@example.com",
			},
			"@example.com": {
				"domain-match@example.com",
			},
			"info": {
				"local-name-match@example.com",
			},
			"@": {
				"wildcard-match@example.com",
			},
		},
	}

	config, err := config.ParseConfig(&rawConfig)
	if err != nil {
		t.Fatal(err)
	}

	type TestCases struct {
		input    string
		expected []string
	}

	tests := map[string]TestCases{
		"full match":       {input: "info@example.com", expected: []string{"full-match@example.com"}},
		"domain match":     {input: "test@example.com", expected: []string{"domain-match@example.com"}},
		"local name match": {input: "info@bar.net", expected: []string{"local-name-match@example.com"}},
		"wildcard match":   {input: "foo@bar.com", expected: []string{"wildcard-match@example.com"}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			transformed, err := TransformRecipients(config, []string{tc.input})
			if err != nil {
				t.Fatalf("transformation failed: %v\n", err)
			}

			expected, actual := parseAddresses(tc.expected, t), transformed[0].Transformed
			if len(expected) != len(actual) {
				t.Errorf("expected %v elements, got %v elements\n", len(expected), len(actual))
			}

			for i := 0; i < len(expected); i++ {
				if expected, actual := expected[i], actual[i]; expected.Address != actual.Address {
					t.Errorf("expected %v, got %v\n", expected, actual)
				}
			}
		})
	}
}

func TestTransformSendersWithoutConfig(t *testing.T) {
	config := config.ParsedConfig{}

	tests := map[string]struct {
		sender                   string
		recipients               []string
		recipientTransformations []TransformationResult
		want                     string
	}{
		"address only": {
			sender:     "sender@example.com",
			recipients: []string{"public@example.com"},
			recipientTransformations: []TransformationResult{
				{
					Source:      parseAddress("public@example.com", t),
					Transformed: parseAddresses([]string{"private@example.com"}, t),
				},
			},
			want: "\"sender@example.com\" <public@example.com>",
		},
		"name and address": {
			sender:     "\"John Doe\" <sender@example.com>",
			recipients: []string{"public@example.com"},
			recipientTransformations: []TransformationResult{
				{
					Source:      parseAddress("public@example.com", t),
					Transformed: parseAddresses([]string{"private@example.com"}, t),
				},
			},
			want: "\"John Doe at sender@example.com\" <public@example.com>",
		},
		"name and address, multiple recipients": {
			sender:     "\"John Doe\" <sender@example.com>",
			recipients: []string{"public-1@example.com", "public-2@example.com"},
			recipientTransformations: []TransformationResult{
				{
					Source:      parseAddress("public-1@example.com", t),
					Transformed: parseAddresses([]string{"private@example.com"}, t),
				},
			},
			want: "\"John Doe at sender@example.com\" <public-1@example.com>",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			newSender, err := TransformSenders(&config, []string{tc.sender}, tc.recipientTransformations)
			if err != nil {
				t.Fatal(err)
			}

			var want, got string
			want, got = tc.want, newSender.String()
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("new sender (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTransformSendersWithConfig(t *testing.T) {
	config := config.ParsedConfig{}
	config.FromEmail = "forwarder@example.com"

	tests := map[string]struct {
		sender                   string
		recipients               []string
		recipientTransformations []TransformationResult
		want                     string
	}{
		"address only": {
			sender:     "sender@example.com",
			recipients: []string{"public@example.com"},
			recipientTransformations: []TransformationResult{
				{
					Source:      parseAddress("public@example.com", t),
					Transformed: parseAddresses([]string{"private@example.com"}, t),
				},
			},
			want: "\"sender@example.com\" <forwarder@example.com>",
		},
		"name and address": {
			sender:     "\"John Doe\" <sender@example.com>",
			recipients: []string{"public@example.com"},
			recipientTransformations: []TransformationResult{
				{
					Source:      parseAddress("public@example.com", t),
					Transformed: parseAddresses([]string{"private@example.com"}, t),
				},
			},
			want: "\"John Doe at sender@example.com\" <forwarder@example.com>",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			newSender, err := TransformSenders(&config, []string{tc.sender}, tc.recipientTransformations)
			if err != nil {
				t.Fatal(err)
			}

			var want, got string
			want, got = tc.want, newSender.String()
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("new sender (-want +got):\n%s", diff)
			}
		})
	}
}
