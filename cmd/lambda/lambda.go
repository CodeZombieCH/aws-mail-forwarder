package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/codezombiech/aws-mail-forwarder/config"
	"github.com/codezombiech/aws-mail-forwarder/forwarder"
)

const (
	ConfigInvalidOrMissingExitCode = -1
	LoadingAwsConfigFailedExitCode = -2
)

var f *forwarder.Forwarder

func HandleRequest(ctx context.Context, sesEvent events.SimpleEmailEvent) error {
	for _, record := range sesEvent.Records {
		ses := record.SES

		// Print event as JSON for debugging
		eventJson, err := json.Marshal(record)
		if err == nil {
			log.Print(string(eventJson))
		}

		err = f.Forward(ses)
		if err != nil {
			log.Print(err)
			return err
		}
	}

	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Load config expected to be present at config(.<env>).json
	configFile := "config.json"
	env := os.Getenv("ENVIRONMENT")
	if env != "" {
		configFile = fmt.Sprintf("config.%s.json", env)
	}

	log.Printf("Loading config file %s", configFile)
	config, err := config.LoadAndParseConfig(configFile)
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		os.Exit(ConfigInvalidOrMissingExitCode)
	}

	awsConfig, err := awsconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Printf("Failed to load AWS config: %v", err)
		os.Exit(LoadingAwsConfigFailedExitCode)
	}

	f = forwarder.NewForwarder(config, awsConfig)

	lambda.Start(HandleRequest)
}
