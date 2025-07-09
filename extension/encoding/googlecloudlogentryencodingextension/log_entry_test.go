// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package googlecloudlogentryencodingextension

import (
	stdjson "encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

type log struct {
	Timestamp          string
	ObservedTimestamp  string
	Body               any
	SeverityText       string
	SeverityNumber     plog.SeverityNumber
	Attributes         map[string]any
	ResourceAttributes map[string]any
	SpanID             string
	TraceID            string
}

func generateLog(t *testing.T, log log) (plog.Logs, error) {
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()

	res := rl.Resource()
	require.NoError(t, res.Attributes().FromRaw(log.ResourceAttributes))

	lr := rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	err := lr.Attributes().FromRaw(log.Attributes)
	if err != nil {
		return logs, err
	}
	if log.Timestamp != "" {
		ts, err := time.Parse(time.RFC3339, log.Timestamp)
		if err != nil {
			return logs, err
		}
		lr.SetTimestamp(pcommon.NewTimestampFromTime(ts))
	}

	if log.ObservedTimestamp != "" {
		ots, err := time.Parse(time.RFC3339, log.ObservedTimestamp)
		if err != nil {
			return logs, err
		}
		lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(ots))
	}

	require.NoError(t, lr.Body().FromRaw(log.Body))
	lr.SetSeverityText(log.SeverityText)
	lr.SetSeverityNumber(log.SeverityNumber)

	spanID, _ := spanIDStrToSpanIDBytes(log.SpanID)
	lr.SetSpanID(spanID)
	traceID, _ := traceIDStrToTraceIDBytes(log.TraceID)
	lr.SetTraceID(traceID)

	return logs, nil
}

func TestCloudLoggingTraceToTraceIDBytes(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		scenario,
		in string
		out         [16]byte
		expectedErr string
	}{
		{
			scenario: "valid_trace",
			in:       "projects/my-gcp-project/traces/1dbe317eb73eb6e3bbb51a2bc3a41e09",
			out:      [16]uint8{0x1d, 0xbe, 0x31, 0x7e, 0xb7, 0x3e, 0xb6, 0xe3, 0xbb, 0xb5, 0x1a, 0x2b, 0xc3, 0xa4, 0x1e, 0x9},
		},
		{
			scenario:    "invalid_trace_format",
			in:          "1dbe317eb73eb6e3bbb51a2bc3a41e09",
			expectedErr: "invalid trace format",
		},
		{
			scenario:    "invalid_hex_trace",
			in:          "projects/my-gcp-project/traces/xyze317eb73eb6e3bbb51a2bc3a41e09",
			expectedErr: "failed to decode trace id to hexadecimal string",
		},
		{
			scenario:    "invalid_trace_length",
			in:          "projects/my-gcp-project/traces/1dbe317eb73eb6e3bbb51a2bc3a41e",
			expectedErr: "expected trace ID hex length to be 16",
		},
	} {
		t.Run(test.scenario, func(t *testing.T) {
			out, err := cloudLoggingTraceToTraceIDBytes(test.in)
			if err != nil {
				require.ErrorContains(t, err, test.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.out, out)
			}
		})
	}
}

func TestSpanIDStrToSpanIDBytes(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		scenario,
		in string
		out        [8]byte
		expectsErr string
	}{
		{
			scenario: "valid_span",
			in:       "3e3a5741b18f0710",
			out:      [8]uint8{0x3e, 0x3a, 0x57, 0x41, 0xb1, 0x8f, 0x7, 0x10},
		},
		{
			scenario:   "invalid_hex_span",
			in:         "123/123",
			expectsErr: "failed to decode span id to hexadecimal string",
		},
		{
			scenario:   "invalid_span_length",
			in:         "3e3a5741b18f0710ab",
			expectsErr: "expected span ID hex length to be 8",
		},
	} {
		t.Run(test.scenario, func(t *testing.T) {
			out, err := spanIDStrToSpanIDBytes(test.in)

			if err != nil {
				require.ErrorContains(t, err, test.expectsErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.out, out)
			}
		})
	}
}

func TestCloudLoggingSeverityToNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		severity string
		expected plog.SeverityNumber
	}{
		{
			severity: "DEBUG",
			expected: plog.SeverityNumberDebug,
		},
		{
			severity: "INFO",
			expected: plog.SeverityNumberInfo,
		},
		{
			severity: "NOTICE",
			expected: plog.SeverityNumberInfo2,
		},
		{
			severity: "WARNING",
			expected: plog.SeverityNumberWarn,
		},
		{
			severity: "ERROR",
			expected: plog.SeverityNumberError,
		},
		{
			severity: "CRITICAL",
			expected: plog.SeverityNumberFatal,
		},
		{
			severity: "ALERT",
			expected: plog.SeverityNumberFatal2,
		},
		{
			severity: "EMERGENCY",
			expected: plog.SeverityNumberFatal4,
		},
		{
			severity: "DEFAULT",
			expected: plog.SeverityNumberUnspecified,
		},
		{
			severity: "UNKNOWN",
			expected: plog.SeverityNumberUnspecified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			res := cloudLoggingSeverityToNumber(tt.severity)
			require.Equal(t, tt.expected, res)
		})
	}
}

func TestHandleTextPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		textPayload stdjson.RawMessage
		expectsBody string
		expectsErr  string
	}{
		{
			name: "valid_text_payload",
			textPayload: func() stdjson.RawMessage {
				raw, err := json.Marshal("valid")
				require.NoError(t, err)
				return raw
			}(),
			expectsBody: "valid",
		},
		{
			name:        "invalid_text_payload",
			textPayload: []byte("invalid"),
			expectsErr:  "failed to unmarshal text payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lr := plog.NewLogRecord()

			err := handleTextPayload(pcommon.Map{}, lr, pcommon.Map{}, "", tt.textPayload, Config{})
			if tt.expectsErr != "" {
				require.ErrorContains(t, err, tt.expectsErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, lr.Body().Str(), tt.expectsBody)
		})
	}
}

func TestHandleJSONPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		jsonPayload stdjson.RawMessage
		expectsBody any
		cfg         Config
		expectsErr  string
	}{
		{
			name: "valid_json_payload_as_json",
			cfg: Config{
				HandleJSONPayloadAs: HandleAsJSON,
			},
			jsonPayload: []byte(`{"test": "test"}`),
			expectsBody: map[string]any{
				"test": "test",
			},
		},
		{
			name: "invalid_json_payload_as_json",
			cfg: Config{
				HandleJSONPayloadAs: HandleAsJSON,
			},
			jsonPayload: []byte("invalid"),
			expectsErr:  "failed to unmarshal JSON payload",
		},
		{
			name: "valid_json_payload_as_text",
			cfg: Config{
				HandleJSONPayloadAs: HandleAsText,
			},
			jsonPayload: []byte(`{"test": "test"}`),
			expectsBody: `{"test": "test"}`,
		},
		{
			name: "invalid_json_payload_unknown",
			cfg: Config{
				HandleJSONPayloadAs: "unknown",
			},
			jsonPayload: []byte("does-not-matter"),
			expectsErr:  "unrecognized JSON payload type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lr := plog.NewLogRecord()

			err := handleJSONPayload(pcommon.Map{}, lr, pcommon.Map{}, "", tt.jsonPayload, tt.cfg)
			if tt.expectsErr != "" {
				require.ErrorContains(t, err, tt.expectsErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, lr.Body().AsRaw(), tt.expectsBody)
		})
	}
}

func TestHandleProtoPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		protoPayload stdjson.RawMessage
		expectsBody  any
		cfg          Config
		expectsErr   string
	}{
		{
			name:         "valid_proto_payload_as_proto",
			protoPayload: []byte(`{"@type":"test"}`),
			cfg: Config{
				HandleProtoPayloadAs: HandleAsProtobuf,
			},
			expectsBody: map[string]any{},
		},
		{
			name:         "invalid_proto_payload_as_proto",
			protoPayload: []byte("invalid"),
			cfg: Config{
				HandleProtoPayloadAs: HandleAsProtobuf,
			},
			expectsErr: "failed to set body from proto payload",
		},
		{
			name:         "valid_proto_payload_as_json",
			protoPayload: []byte(`{"@type":"test"}`),
			cfg: Config{
				HandleProtoPayloadAs: HandleAsJSON,
			},
			expectsBody: map[string]any{
				"@type": "test",
			},
		},
		{
			name:         "invalid_proto_payload_as_json",
			protoPayload: []byte("invalid"),
			cfg: Config{
				HandleProtoPayloadAs: HandleAsJSON,
			},
			expectsErr: "failed to unmarshal JSON payload",
		},
		{
			name:         "valid_proto_payload_as_text",
			protoPayload: []byte(`{"@type":"test"}`),
			cfg: Config{
				HandleProtoPayloadAs: HandleAsText,
			},
			expectsBody: `{"@type":"test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lr := plog.NewLogRecord()

			err := handleProtoPayload(pcommon.Map{}, lr, pcommon.Map{}, "", tt.protoPayload, tt.cfg)
			if tt.expectsErr != "" {
				require.ErrorContains(t, err, tt.expectsErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, lr.Body().AsRaw(), tt.expectsBody)
		})
	}
}
