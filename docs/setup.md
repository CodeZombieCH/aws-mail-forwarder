# Setup

This document describes the set up procedure for the AWS Mail Forwarder. Please follow the instruction from top to bottom and read them carefully.

The order of sections in this document was carefully selected to give you the most straightforward experience, relaying on automatism of the AWS Console.


## Step 1: AWS Region

Choose an AWS region (`<region>`) that supports receiving emails:

- us-east-1
- us-west-2
- eu-west-1

See https://docs.aws.amazon.com/ses/latest/dg/regions.html#region-receive-email for more details.


## Step 2: S3 Bucket

Create a S3 bucket (`<bucket>`) in `<region>`:

- *Bucket name*: e.g. `aws-mail-forwarder`
- *Object Ownership*: ACL disabled
- *Block Public Access settings for this bucket*: Block all public access
- *Bucket Versioning*: disabled


## Step 3: S3 Bucket Policy

Navigate to *Permissions* and add the following bucket policy:

```json
{
  "Version":"2012-10-17",
  "Statement":[
    {
      "Sid":"AllowSESPuts",
      "Effect":"Allow",
      "Principal":{
        "Service":"ses.amazonaws.com"
      },
      "Action":"s3:PutObject",
      "Resource":"arn:aws:s3:::<bucket>/*",
      "Condition":{
        "StringEquals":{
          "AWS:SourceAccount":"<your-arn>"
        }
      }
    }
  ]
}
```

See https://docs.aws.amazon.com/ses/latest/dg/receiving-email-permissions.html for more details.


## Step 4: Lambda Function

Create a Lambda function (`<lambda>`) in `<region>` using the "Author from scratch" template:

- *Function name:* e.g. `aws-mail-forwarder`
- *Runtime*: Go 1.x
- *Architecture*: x86_64
- *Permissions*: default (Create a new role with basic Lambda permissions)

After the function has been created, navigate to *Configuration* > *General configuration* and adjust the following settings:

- *Memory*: 256 (recommended, 128 works for mails below 10MB)
- *Ephemeral storage*: 512 (lowest value possible)
- *Timeout*: 15sec  (recommended)


## Step 5: SES

### Verified Identities
In the *Verified identities* tab, add the domain(s) for which you want to receive and forward email.

Configure the MX records of the DNS zones for your domains to use the AWS Email Receiving Endpoint.
See https://docs.aws.amazon.com/ses/latest/dg/regions.html#region-receive-email for more details.

### Email Receiving
1. In the *Email receiving* tab, create a new rule set or modify an existing rule set
1. Select the rule set and add a new rule which opens a wizard with multiple steps.

    Step 1:
    - *Rule name*: e.g. `aws-mail-forwarder`
    - *Spam and virus scanning*: enable

    Step 2:
    - *Recipient conditions*: Add the domains you want to receive emails for

    Step 3:<br>
    Add the following actions:
    1. Deliver to Amazon S3 bucket
        - *S3 bucket*: previously created bucket `<bucket>`
        - *Object key prefix*: `in/new/`
    2. Invoke AWS Lambda function
        - *Lambda function*: previously created lambda function `<lambda>`
        - *Invocation type*: Event invocation

    Step 4:<br>
    Review and confirm the new rule.
    A confirmation dialog will pop up, asking you if you want to add the required permissions (the permission to invoke your lambda function). Confirm by selecting "Add permission".

## Step 6: IAM Policy

The IAM policy assigned to the lambda function needs to be changed to grant access to the previously created AWS resources.

1. Navigate to *Access management* > *Roles*
1. Find the IAM role automatically generated when creating the lambda function. The role name starts with your `<lambda>` function name, e.g. "aws-mail-forwarder-role-jsyybqqy"
1. Select the role and open expand the single attached policy, usually named "AWSLambdaBasicExecutionRole-`<uuid>`"
1. Click on "Edit" and switch to the *JSON* tab<br>
    Add the following statements to your policy and replace the placeholders:

    ```json
    {
        "Effect": "Allow",
        "Action": [
            "ses:SendRawEmail",
            "ses:SendEmail"
        ],
        "Resource": "*"
    },
    {
        "Effect": "Allow",
        "Action": [
            "s3:GetObject",
            "s3:PutObject",
            "s3:DeleteObject"
        ],
        "Resource": "arn:aws:s3:::<bucket>/*"
    }
    ```

    The complete policy should look like this:

    ```json
    {
        "Version": "2012-10-17",
        "Statement": [
            {
                "Effect": "Allow",
                "Action": "logs:CreateLogGroup",
                "Resource": "arn:aws:logs:eu-west-1:111122223333:*"
            },
            {
                "Effect": "Allow",
                "Action": [
                    "logs:CreateLogStream",
                    "logs:PutLogEvents"
                ],
                "Resource": [
                    "arn:aws:logs:eu-west-1:111122223333:log-group:/aws/lambda/aws-mail-forwarder:*"
                ]
            },
            {
                "Effect": "Allow",
                "Action": [
                    "ses:SendRawEmail",
                    "ses:SendEmail"
                ],
                "Resource": "*"
            },
            {
                "Effect": "Allow",
                "Action": [
                    "s3:GetObject",
                    "s3:PutObject",
                    "s3:DeleteObject"
                ],
                "Resource": "arn:aws:s3:::aws-mail-forwarder/*"
            }
        ]
    }
    ```
1. Save the changes


## Step 7: Lambda Function

## Prepare Function

1. Download the `lambda` binary from the [Github Releases page](https://github.com/codezombiech/aws-mail-forwarder/releases/) or compile it yourself.
1. Create a temporary directory (`<temp-dir>`)
1. Copy the `lambda` binary to `<temp-dir>`
1. Copy the example configuration from [build/config.example.json](build/config.example.json) to `<temp-dir>` and rename it to `config.json`
1. Adjust the config to your needs
1. Create a ZIP file (`<zip-archive>`) with the `lambda` binary and the `config.json` configuration file

## Upload Function

1. Open the `<lambda>` function in the AWS Console
1. In the *Code source* region, upload your ZIP file `<zip-archive>`
1. In the *Runtime settings* region click on edit and change the *Handler* to `lambda`.

Done 😫

![exhausted](images/exhausted.png)


## Validation

After you completed the set up instructions you should now have the following AWS resources:

- IAM
    - [x] Role: `aws-mail-forwarder-role-<random-suffix>`
    - [x] Policy: `AWSLambdaBasicExecutionRole-<uuid>`
- S3
    - [x] Bucket: `aws-mail-forwarder`
        - [x] Bucket Policy: inline bucket policy (allows writing from SES)
- SES
    - [x] Verified Identities
        - [x] example.com
- Lambda
    - [x] Function: `aws-mail-forwarder`
        - Resource-based policy statements: inline policy (allows invocation from SES)
