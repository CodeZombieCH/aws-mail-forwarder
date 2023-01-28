module github.com/codezombiech/aws-mail-forwarder

go 1.19

require (
	github.com/aws/aws-lambda-go v1.37.0
	github.com/aws/aws-sdk-go-v2 v1.17.3
	github.com/aws/aws-sdk-go-v2/config v1.18.10
	github.com/aws/aws-sdk-go-v2/service/s3 v1.30.1
	github.com/aws/aws-sdk-go-v2/service/sesv2 v1.16.0
	github.com/aws/smithy-go v1.13.5
	github.com/google/go-cmp v0.5.9
	github.com/jordan-wright/email v4.0.1-0.20210109023952-943e75fe5223+incompatible
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.10 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.13.10 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.27 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.28 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.0.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.13.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.12.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.14.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.18.2 // indirect
)
