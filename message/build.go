package message

import (
	"bytes"
	"fmt"
	"strings"
)

func BuildMail(message *BufferedMessage) ([]byte, error) {
	// Prepare header
	var header strings.Builder
	for k, v := range message.Header {
		header.WriteString(k + ": " + strings.Join(v, ",") + RFC5322LineDelimiter)
	}

	var buffer bytes.Buffer
	var err error

	// Write header
	_, err = buffer.Write([]byte(header.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to write header: %w", err)
	}

	// Separate header from body
	// See https://www.rfc-editor.org/rfc/rfc5322#section-2.1
	_, err = buffer.WriteString(RFC5322LineDelimiter + RFC5322LineDelimiter)
	if err != nil {
		return nil, fmt.Errorf("failed to write separator: %w", err)
	}

	// Write body
	_, err = buffer.Write(message.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to write body: %w", err)
	}

	// Write EOL
	_, err = buffer.WriteString(RFC5322LineDelimiter)
	if err != nil {
		return nil, fmt.Errorf("failed to write EOL: %w", err)
	}

	return buffer.Bytes(), nil
}
