//go:build integration

package storage

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
)

type testConfig struct {
	awsConfig  aws.Config
	bucketName string
}

func getTestConfig(t *testing.T) testConfig {
	checkEnv(t, "AWS_PROFILE")
	checkEnv(t, "AWS_REGION")
	checkEnv(t, "BUCKET_NAME")

	awsConfig, err := awsconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		t.Fatalf("Failed to load AWS config: %v", err)
	}

	return testConfig{
		awsConfig:  awsConfig,
		bucketName: os.Getenv("BUCKET_NAME"),
	}
}

func checkEnv(t *testing.T, envName string) {
	if os.Getenv(envName) == "" {
		t.Fatalf("Environment variable %s must be set", envName)
	}
}

func TestGet(t *testing.T) {
	config := getTestConfig(t)

	storage := NewStorage(config.awsConfig, config.bucketName)

	reader, _, err := storage.Get("test/test-file-get.txt")
	if err != nil {
		t.Fatal(err)
	}

	// Read as string
	bytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := "test file content\n", string(bytes); expected != actual {
		t.Errorf("expected '%v', got '%v'", expected, actual)
	}
}

func TestPut(t *testing.T) {
	config := getTestConfig(t)

	storage := NewStorage(config.awsConfig, config.bucketName)

	reader := strings.NewReader("test file content")
	etag, err := storage.Put("test/test-file-put.txt", reader)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(etag)
}

func TestMove(t *testing.T) {
	config := getTestConfig(t)

	storage := NewStorage(config.awsConfig, config.bucketName)

	err := storage.Move("test/test-file-move.txt", "test/moved/test-file-move.txt")
	if err != nil {
		t.Fatal(err)
	}

	// Move it back
	err = storage.Move("test/moved/test-file-move.txt", "test/test-file-move.txt")
	if err != nil {
		t.Fatal(err)
	}
}
