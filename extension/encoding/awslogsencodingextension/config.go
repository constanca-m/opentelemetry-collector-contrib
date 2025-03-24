// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package awslogsencodingextension // import "github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding/awslogsencodingextension"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/confmap/xconfmap"
)

var _ xconfmap.Validator = (*Config)(nil)

const (
	formatCloudWatchLogsSubscriptionFilter = "cloudwatch_logs_subscription_filter"
	formatVPCFlowLog                       = "vpc_flow_log"

	fileFormatPlainText = "plain-text"
	fileFormatParquet   = "parquet"
)

var (
	supportedLogFormats           = []string{formatCloudWatchLogsSubscriptionFilter, formatVPCFlowLog}
	supportedVPCFlowLogFileFormat = []string{fileFormatPlainText, fileFormatParquet}
)

type Config struct {
	// Format defines the AWS logs format.
	//
	// Valid values are defined in supportedLogFormats
	Format string `mapstructure:"format"`

	VPCFlowLogConfig VPCFlowLogConfig `mapstructure:"vpc_flow_log"`
}

type VPCFlowLogConfig struct {
	// VPC flow logs sent to S3 have support
	// for file format in plain text or
	// parquet. Default is plain text.
	//
	// See https://docs.aws.amazon.com/vpc/latest/userguide/flow-logs-s3-path.html.
	FileFormat string `mapstructure:"file_format"`
}

func (cfg *Config) Validate() error {
	var errs []error

	switch cfg.Format {
	case "":
		errs = append(errs, fmt.Errorf("format unspecified, expected one of %q", supportedLogFormats))
	case formatCloudWatchLogsSubscriptionFilter: // valid
	case formatVPCFlowLog: // valid
	default:
		errs = append(errs, fmt.Errorf("unsupported format %q, expected one of %q", cfg.Format, supportedLogFormats))
	}

	switch cfg.VPCFlowLogConfig.FileFormat {
	case fileFormatParquet: // valid
	case fileFormatPlainText: // valid
	default:
		errs = append(errs, fmt.Errorf(
			"unsupported file format %q for VPC flow log, expected one of %q",
			cfg.VPCFlowLogConfig.FileFormat,
			supportedVPCFlowLogFileFormat,
		))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
