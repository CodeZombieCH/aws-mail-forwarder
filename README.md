# AWS Mail Forwarder

A mail forwarder powered by Amazon Web Services (AWS) written in Go.

This project started as a port of the amazing [aws-lambda-ses-forwarder](https://github.com/arithmetric/aws-lambda-ses-forwarder)
from JavaScript to Go but soon turned into an almost complete rewrite with many new features.


## Features

- Forwards mails based on a mapping definition
- Supports mails up to 40MB in total size (including headers, body, attachments)
- Stores mails in S3 according to their forwarding state (sent, failed, ...)
- Skips mails marked as SPAM/virus but stores them for later inspection
- Uses packages from the Go standard library to parse mails and mail addresses


## Use Cases

- Forward mails sent to your domain(s) to another mail account (e.g. Gmail, Outlook, ProtonMail) without the need to run your own mail server or pay for a managed mailbox.
- Hide your mail addresses by using a public proxy domain and forward mails to your private mail address.


## Cost

AWS Mail Forwarder is based on Amazon Web Services and runs at almost no cost. But as always, there is a price to pay: setting it up takes some time and knowledge of AWS and DNS.


## Forwarding Process

Mails sent to your domain and received by AWS SES are forwarded to other mail address(es) by rewriting mail headers and sending the rewritten mail using AWS SES.

Example:

|              | Original Message                | Forwarded Message                                            |
|--------------|---------------------------------|--------------------------------------------------------------|
| Sender       | "John Doe" \<sender@example.com\> | "John Doe at sender@example.com" \<forwarder@your-domain.tld\> |
| Recipient(s) | info@your-domain.tld            | your-mail@some-mail-provider.com                             |


## Limitations

### Limited Region Availability

Receiving emails is currently only supported in the following regions:
- us-east-1
- us-west-2
- eu-west-1

Make sure to set up your AWS objects (S3, Lambda, SES) in one of this regions!

See https://docs.aws.amazon.com/ses/latest/dg/regions.html#region-receive-email for more details

### No Bounce

Messages are only processed and never bounce

### Limited Support for Multiple From

[RFC 5322](https://www.rfc-editor.org/rfc/rfc5322) allows multiple `From` headers, although this is seldomly used in the real world. AWS Mail Forwarder has limited support for multiple `From` headers in that it rewrites the message to forward by only taking the first `From` header.


## Installation

See [docs/setup.md](docs/setup.md)


## Configuration

See example config file [`config.example.json`](build/config.example.json)


## Debugging

### Headers

The following debugging headers are added to the message sent:

- `X-Forwarder-Message-Id`:<br>
  The unique ID assigned to the email by Amazon SES
  The message ID is used:
  - by SES as the key of the message stored in the S3 bucket after receiving a message
  - by the lambda function as the key of the message stored in the S3 bucket after sending a forwarded message
- `X-Forwarder-Original-From`:<br>
  The original `From` header


## History

### Port

This project started because I thought it would be a nice exercise to port [aws-lambda-ses-forwarder](https://github.com/arithmetric/aws-lambda-ses-forwarder) to `Go` while having the following improvements in mind:

- Type safety
- Real mail parsing instead of regex
- Reduced memory consumption
- Usage of the newer SES v2 API to send mails

The port turned out to be a great success, with memory consumption reduced by 50%.

### Continuation
After the successful port I decided to continue working on this project by ditching the goal to be compatible with the original Node.js application and adding new features that I had in mind for quite a while.
