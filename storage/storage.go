package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

type Storage struct {
	s3Client   *s3.Client
	bucketName string
}

func NewStorage(awsConfig aws.Config, bucketName string) *Storage {
	return &Storage{
		s3Client:   s3.NewFromConfig(awsConfig),
		bucketName: bucketName,
	}
}

func (s *Storage) Get(key string) (io.ReadCloser, int64, error) {
	input := s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	result, err := s.s3Client.GetObject(context.TODO(), &input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			return nil, 0, fmt.Errorf(
				"failed to get object (code: %s, message: %s, fault: %s)",
				apiErr.ErrorCode(), apiErr.ErrorMessage(), apiErr.ErrorFault().String(),
			)
		}
		return nil, 0, fmt.Errorf("failed to get object: %w", err)
	}

	return result.Body, result.ContentLength, nil
}

func (s *Storage) Put(key string, reader io.Reader) (*string, error) {
	input := s3.PutObjectInput{
		Body:   reader,
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	result, err := s.s3Client.PutObject(context.TODO(), &input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			return nil, fmt.Errorf(
				"failed to put object (code: %s, message: %s, fault: %s)",
				apiErr.ErrorCode(), apiErr.ErrorMessage(), apiErr.ErrorFault().String(),
			)
		}
		return nil, fmt.Errorf("failed to put object: %w", err)
	}
	return result.ETag, nil
}

func (s *Storage) Move(sourceKey string, targetKey string) error {
	if _, err := s.copy(sourceKey, targetKey); err != nil {
		return err
	}

	if err := s.delete(sourceKey); err != nil {
		return err
	}

	return nil
}

func (s *Storage) copy(sourceKey string, targetKey string) (*string, error) {
	input := s3.CopyObjectInput{
		Bucket:     aws.String(s.bucketName),
		CopySource: aws.String(s.bucketName + "/" + sourceKey),
		Key:        aws.String(targetKey),
	}

	result, err := s.s3Client.CopyObject(context.TODO(), &input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			return nil, fmt.Errorf(
				"failed to copy object (code: %s, message: %s, fault: %s)",
				apiErr.ErrorCode(), apiErr.ErrorMessage(), apiErr.ErrorFault().String(),
			)
		}
		return nil, fmt.Errorf("failed to copy object: %w", err)
	}

	return result.CopyObjectResult.ETag, nil
}

func (s *Storage) delete(key string) error {
	input := s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	_, err := s.s3Client.DeleteObject(context.TODO(), &input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			return fmt.Errorf(
				"failed to delete object (code: %s, message: %s, fault: %s)",
				apiErr.ErrorCode(), apiErr.ErrorMessage(), apiErr.ErrorFault().String(),
			)
		}
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}
