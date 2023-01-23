package config

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"os"
)

// Forwarder configuration
type RawConfig struct {
	FromEmail      string              `json:"fromEmail"`      // Email address the From header will be overwritten to (if specified)
	ToEmail        string              `json:"toEmail"`        // Email address the To header will be overwritten to (if specified)
	SubjectPrefix  string              `json:"subjectPrefix"`  // A prefix that will be added to the Subject header (if specified)
	AllowPlusSign  bool                `json:"allowPlusSign"`  // Allow "+" (plus) sign in recipient addresses (part after "+" will be removed)
	ForwardMapping map[string][]string `json:"forwardMapping"` // Mapping of incoming recipients to forwarded recipients
	S3             S3Config            `json:"s3"`
}

// AWS S3 configuration
type S3Config struct {
	BucketName string           `json:"bucketName"` // Name of the S3 bucket
	Incoming   S3IncomingConfig `json:"incoming"`
	Outgoing   S3OutgoingConfig `json:"outgoing"`
}

// AWS S3 configuration for storing incoming messages according to their states
type S3IncomingConfig struct {
	NewPrefix       string `json:"newPrefix"`       // Prefix (directory) where new messages received by SES are expected to be stored
	SpamVirusPrefix string `json:"spamVirusPrefix"` // Prefix (directory) for messages that were flagged as spam or/and virus
	ForwardedPrefix string `json:"forwardedPrefix"` // Prefix (directory) for messages that were successfully forwarded
	FailedPrefix    string `json:"failedPrefix"`    // Prefix (directory) for messages that failed to be forwarded
}

// AWS S3 configuration for storing outgoing messages according to their states
type S3OutgoingConfig struct {
	SentPrefix   string `json:"sentPrefix"`   // Prefix (directory) for messages that were successfully sent
	FailedPrefix string `json:"failedPrefix"` // Prefix (directory) for messages that failed to be sent
}

type ParsedConfig struct {
	RawConfig
	ForwardMapping map[string][]*mail.Address
}

func LoadAndParseConfig(path string) (*ParsedConfig, error) {
	rawConfig, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}

	config, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func ParseConfig(config *RawConfig) (*ParsedConfig, error) {
	parsedMapping := make(map[string][]*mail.Address, 0)

	for key, mapping := range config.ForwardMapping {
		parsedMappingRecipients := make([]*mail.Address, 0)
		for _, mappingRecipient := range mapping {
			parsedMappingRecipient, err := mail.ParseAddress(mappingRecipient)
			if err != nil {
				return nil, fmt.Errorf("invalid address in mapping: %s => %s, %w", key, mappingRecipient, err)
			}
			parsedMappingRecipients = append(parsedMappingRecipients, parsedMappingRecipient)
		}
		parsedMapping[key] = parsedMappingRecipients
	}

	return &ParsedConfig{RawConfig: *config, ForwardMapping: parsedMapping}, nil
}

func LoadConfig(path string) (*RawConfig, error) {
	config := RawConfig{}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize config file: %w", err)
	}

	return &config, nil
}

func SaveConfig(path string, config *RawConfig) error {
	bytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to serialize config file: %w", err)
	}
	err = os.WriteFile(path, bytes, 0666)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}
