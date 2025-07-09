// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package googlecloudlogentryencodingextension

import (
	stdjson "encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/plogtest"
)

func TestProtoPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		scenario string
		config   Config
		wantFile string
	}{
		{
			scenario: "AsProtobuf",
			config: Config{
				HandleProtoPayloadAs: HandleAsProtobuf,
			},
			wantFile: "testdata/proto_payload/as_protofobuf_expected.yaml",
		},
		{
			scenario: "AsJSON",
			config: Config{
				HandleProtoPayloadAs: HandleAsJSON,
			},
			wantFile: "testdata/proto_payload/as_json_expected.yaml",
		},
		{
			scenario: "AsText",
			config: Config{
				HandleProtoPayloadAs: HandleAsText,
			},
			wantFile: "testdata/proto_payload/as_text_expected.yaml",
		},
	}

	input, err := os.ReadFile("testdata/proto_payload/proto_payload.json")
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			extension := newTestExtension(t, tt.config)

			wantRes, err := golden.ReadLogs(tt.wantFile)
			require.NoError(t, err)

			gotRes, err := extension.UnmarshalLogs(input)
			require.NoError(t, err)

			require.NoError(t, plogtest.CompareLogs(wantRes, gotRes))
		})
	}
}

func TestProtoFieldTypes(t *testing.T) {
	tests := []struct {
		scenario string
		input    string
		expected log
	}{
		{
			"String",
			`{
  "protoPayload": {
    "@type": "type.googleapis.com/google.cloud.audit.AuditLog",
    "serviceName": "OpenTelemetry"
  }
}`,
			log{
				Body: map[string]any{
					"@type":       "type.googleapis.com/google.cloud.audit.AuditLog",
					"serviceName": "OpenTelemetry",
				},
			},
		},
		{
			"Boolean",
			`{
  "protoPayload": {
    "@type": "type.googleapis.com/google.cloud.audit.AuditLog",
    "authorizationInfo": [
      {
        "granted": true
      }
    ]
  }
}`,
			log{
				Body: map[string]any{
					"@type":             "type.googleapis.com/google.cloud.audit.AuditLog",
					"authorizationInfo": []any{map[string]any{"granted": true}},
				},
			},
		},
		{
			"EnumByString",
			`{
  "protoPayload": {
    "@type": "type.googleapis.com/google.cloud.audit.AuditLog",
    "policyViolationInfo": {
      "orgPolicyViolationInfo": {
        "violationInfo": [
          {
            "policyType": "CUSTOM_CONSTRAINT"
          }
        ]
      }
    }
  }
}`,
			log{
				Body: map[string]any{
					"@type":               "type.googleapis.com/google.cloud.audit.AuditLog",
					"policyViolationInfo": map[string]any{"orgPolicyViolationInfo": map[string]any{"violationInfo": []any{map[string]any{"policyType": "CUSTOM_CONSTRAINT"}}}},
				},
			},
		},
		{
			"EnumByNumber",
			`{
  "protoPayload": {
    "@type": "type.googleapis.com/google.cloud.audit.AuditLog",
    "policyViolationInfo": {
      "orgPolicyViolationInfo": {
        "violationInfo": [
          {
            "policyType": 3
          }
        ]
      }
    }
  }
}`,
			log{
				Body: map[string]any{
					"@type":               "type.googleapis.com/google.cloud.audit.AuditLog",
					"policyViolationInfo": map[string]any{"orgPolicyViolationInfo": map[string]any{"violationInfo": []any{map[string]any{"policyType": "CUSTOM_CONSTRAINT"}}}},
				},
			},
		},
		{
			"BestEffortAnyType",
			`{
  "protoPayload": {
    "@type": "type.examples/does.not.Exist",
    "noName": "Foobar"
  }
}`,
			log{
				Body: map[string]any{
					"noName": "Foobar",
				},
			},
		},
	}

	config := Config{
		HandleProtoPayloadAs: HandleAsProtobuf,
	}
	for _, tt := range tests {
		fn := func(t *testing.T, want log) {
			extension := newTestExtension(t, config)

			wantRes, err := generateLog(t, want)
			require.NoError(t, err)

			gotRes, err := extension.translateLogEntry([]byte(tt.input))
			require.NoError(t, err)

			require.NoError(t, plogtest.CompareLogs(wantRes, gotRes))
		}
		t.Run(tt.scenario, func(t *testing.T) {
			fn(t, tt.expected)
		})
	}
}

func TestProtoErrors(t *testing.T) {
	tests := []struct {
		scenario string
		input    string
		error    string
		expected log
	}{
		{
			"UnknownJSONName",
			"{\n  \"protoPayload\": {\n    \"@type\": \"type.googleapis.com/google.cloud.audit.AuditLog\",\n    \"ServiceName\": 42\n  }\n}",
			"google.cloud.audit.AuditLog has no known field with JSON name ServiceName",
			log{
				Attributes: map[string]any{
					"gcp.proto_payload": map[string]any{},
				},
				Body: map[string]any{},
			},
		},
		{
			"EnumTypeError",
			"{\n  \"protoPayload\": {\n    \"@type\": \"type.googleapis.com/google.cloud.audit.AuditLog\",\n    \"policyViolationInfo\": {\n      \"orgPolicyViolationInfo\": {\n        \"violationInfo\": [\n          {\n            \"policyType\": {}\n          }\n        ]\n      }\n    }\n  }\n}",
			"wrong type for enum: object",
			log{
				Body: map[string]any{
					"policyViolationInfo": map[string]any{"orgPolicyViolationInfo": map[string]any{"violationInfo": []any{map[string]any{"policyType": nil}}}},
				},
				Attributes: map[string]any{
					"gcp.proto_payload": map[string]any{
						"policyViolationInfo": map[string]any{"orgPolicyViolationInfo": map[string]any{"violationInfo": []any{map[string]any{"policyType": nil}}}},
					},
				},
			},
		},
	}

	config := Config{
		HandleProtoPayloadAs: HandleAsProtobuf,
	}
	for _, tt := range tests {
		fn := func(t *testing.T, want log) {
			extension := newTestExtension(t, config)

			wantRes, err := generateLog(t, want)
			require.NoError(t, err)

			gotRes, err := extension.translateLogEntry([]byte(tt.input))
			require.ErrorContains(t, err, tt.error)

			require.NoError(t, plogtest.CompareLogs(wantRes, gotRes))
		}
		t.Run(tt.scenario, func(t *testing.T) {
			fn(t, tt.expected)
		})
	}
}

func TestGetTokenType(t *testing.T) {
	tests := []struct {
		scenario string
		input    string
		expected string
	}{
		{
			"Object",
			"{}",
			"object",
		},
		{
			"Object",
			"[]",
			"array",
		},
		{
			"Number",
			"1.1",
			"number",
		},
		{
			"Boolean",
			"true",
			"bool",
		},
		{
			"String",
			"\"\"",
			"string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			j := stdjson.RawMessage{}
			err := j.UnmarshalJSON([]byte(tt.input))
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, getTokenType(j))
		})
	}
}
