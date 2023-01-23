package config

import (
	"net/mail"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLoadConfig(t *testing.T) {
	config, err := LoadConfig("testdata/test-load-config.json")
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := "from@example.net", config.FromEmail; expected != actual {
		t.Errorf("FromEmail: expected %v, got %v", expected, actual)
	}
}

func TestSaveConfig(t *testing.T) {
	config := RawConfig{
		FromEmail:     "from@example.net",
		SubjectPrefix: "Prefix: ",
		AllowPlusSign: false,
		S3: S3Config{
			BucketName: "testBucket",
			Incoming: S3IncomingConfig{
				NewPrefix:       "in/new/",
				SpamVirusPrefix: "in/spam-virus/",
				ForwardedPrefix: "in/forwarded/",
				FailedPrefix:    "in/failed/",
			},
			Outgoing: S3OutgoingConfig{
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
				"example.john@example.com",
			},
			"info": {
				"info@example.com",
			},
		},
	}

	err := SaveConfig("testdata/test-safe-config.json", &config)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseConfig(t *testing.T) {
	config := RawConfig{
		FromEmail:     "from@example.net",
		SubjectPrefix: "Prefix: ",
		AllowPlusSign: false,
		S3: S3Config{
			BucketName: "testBucket",
			Incoming: S3IncomingConfig{
				NewPrefix:       "in/new/",
				SpamVirusPrefix: "in/spam-virus/",
				ForwardedPrefix: "in/forwarded/",
				FailedPrefix:    "in/failed/",
			},
			Outgoing: S3OutgoingConfig{
				SentPrefix:   "out/sent/",
				FailedPrefix: "out/failed/",
			},
		},
		ForwardMapping: map[string][]string{
			"single@example.com": {
				"single@example.net",
			},
			"@example.com": {
				"multiple-1@example.net",
				"multiple-2@example.net",
			},
		},
	}

	parsedConfig, err := ParseConfig(&config)
	if err != nil {
		t.Fatal(err)
	}

	compareAddresses := func(mapping map[string][]*mail.Address, key string, want []string) {
		got := make([]string, 0)
		for _, v := range parsedConfig.ForwardMapping[key] {
			got = append(got, v.Address)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("%s: (-want +got):\n%s", key, diff)
		}
	}

	compareAddresses(parsedConfig.ForwardMapping, "single@example.com", []string{"single@example.net"})
	compareAddresses(parsedConfig.ForwardMapping, "@example.com", []string{"multiple-1@example.net", "multiple-2@example.net"})

	if expected, actual := "from@example.net", config.FromEmail; expected != actual {
		t.Errorf("FromEmail: expected %v, got %v", expected, actual)
	}
}

func TestParseConfigError(t *testing.T) {
	config := RawConfig{
		ForwardMapping: map[string][]string{
			"single@example.com": {
				"single@example@net",
			},
		},
	}

	_, err := ParseConfig(&config)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}

	if want, got := "invalid address in mapping: single@example.com => single@example@net, mail: expected single address, got \"@net\"", err.Error(); want != got {
		t.Fatalf("want %s, got %s", want, got)
	}
}
