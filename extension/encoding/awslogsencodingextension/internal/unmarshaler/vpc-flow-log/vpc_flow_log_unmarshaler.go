// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package vpcflowlog // import "github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding/awslogsencodingextension/internal/unmarshaler/vpc-flow-log"

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/klauspost/compress/gzip"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	conventions "go.opentelemetry.io/collector/semconv/v1.27.0"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding/awslogsencodingextension/internal/metadata"
)

const (
	fileFormatPlainText = "plain-text"
	fileFormatParquet   = "parquet"
)

var supportedVPCFlowLogFileFormat = []string{fileFormatPlainText, fileFormatParquet}

type vpcFlowLogUnmarshaler struct {
	// VPC flow logs can be sent in plain text
	// or parquet files to S3.
	//
	// See https://docs.aws.amazon.com/vpc/latest/userguide/flow-logs-s3-path.html.
	fileFormat string

	// Pool the gzip readers, which are expensive to create.
	gzipPool sync.Pool

	buildInfo component.BuildInfo
}

func NewVPCFlowLogUnmarshaler(format string, buildInfo component.BuildInfo) (plog.Unmarshaler, error) {
	switch format {
	case fileFormatParquet:
		// TODO
		return nil, errors.New("still needs to be implemented")
	case fileFormatPlainText: // valid
	default:
		return nil, fmt.Errorf(
			"unsupported file fileFormat %q for VPC flow log, expected one of %q",
			format,
			supportedVPCFlowLogFileFormat,
		)
	}
	return &vpcFlowLogUnmarshaler{
		fileFormat: format,
		gzipPool:   sync.Pool{},
		buildInfo:  buildInfo,
	}, nil
}

func (v *vpcFlowLogUnmarshaler) UnmarshalLogs(content []byte) (plog.Logs, error) {
	var errGzipReader error
	gzipReader, ok := v.gzipPool.Get().(*gzip.Reader)
	if !ok {
		gzipReader, errGzipReader = gzip.NewReader(bytes.NewReader(content))
	} else {
		errGzipReader = gzipReader.Reset(bytes.NewReader(content))
	}
	if errGzipReader != nil {
		if gzipReader != nil {
			v.gzipPool.Put(gzipReader)
		}
		return plog.Logs{}, fmt.Errorf("failed to decompress content: %w", errGzipReader)
	}
	defer func() {
		_ = gzipReader.Close()
		v.gzipPool.Put(gzipReader)
	}()

	switch v.fileFormat {
	case fileFormatPlainText:
		return v.unmarshalPlainTextLogs(gzipReader)
	case fileFormatParquet:
		// TODO
		return plog.Logs{}, errors.New("still needs to be implemented")
	default:
		// not possible, prevent by NewVPCFlowLogUnmarshaler
		return plog.Logs{}, nil
	}
}

type resourceKey struct {
	accountID string
	region    string
}

func (v *vpcFlowLogUnmarshaler) unmarshalPlainTextLogs(reader *gzip.Reader) (plog.Logs, error) {
	scanner := bufio.NewScanner(reader)

	// first line includes the fields
	// TODO Replace with an iterator starting from go 1.24:
	// https://pkg.go.dev/strings#FieldsSeq
	var fields []string
	if scanner.Scan() {
		firstLine := scanner.Text()
		fields = strings.Split(firstLine, " ")
	}

	scopeLogsByResource := map[resourceKey]plog.ScopeLogs{}
	for scanner.Scan() {
		line := scanner.Text()
		if err := v.addToLogs(scopeLogsByResource, fields, line); err != nil {
			return plog.Logs{}, err
		}
	}

	if err := scanner.Err(); err != nil {
		return plog.Logs{}, fmt.Errorf("error reading log line: %w", err)
	}

	return v.createLogs(scopeLogsByResource), nil
}

// createLogs based on the scopeLogsByResource map
func (v *vpcFlowLogUnmarshaler) createLogs(scopeLogsByResource map[resourceKey]plog.ScopeLogs) plog.Logs {
	logs := plog.NewLogs()

	for key, scopeLogs := range scopeLogsByResource {
		rl := logs.ResourceLogs().AppendEmpty()
		attr := rl.Resource().Attributes()
		attr.PutStr(conventions.AttributeCloudProvider, conventions.AttributeCloudProviderAWS)
		if key.accountID != "" {
			attr.PutStr(conventions.AttributeCloudAccountID, key.accountID)
		}
		if key.region != "" {
			attr.PutStr(conventions.AttributeCloudRegion, key.region)
		}
		scopeLogs.MoveTo(rl.ScopeLogs().AppendEmpty())
	}

	return logs
}

// addToLogs parses the log line and creates
// a new record log. The record log is added
// to the scope logs of the resource identified
// by the resourceKey created from the values.
func (v *vpcFlowLogUnmarshaler) addToLogs(
	scopeLogsByResource map[resourceKey]plog.ScopeLogs,
	fields []string,
	logLine string,
) error {
	values := strings.Split(logLine, " ")
	nFields := len(fields)
	nValues := len(values)
	if nFields != nValues {
		return fmt.Errorf("expect %d fields per log line, got log line with %d fields", nFields, nValues)
	}

	// create new key for resource and new
	// log record to add to the scope of logs
	// of the resource
	key := &resourceKey{}
	record := plog.NewLogRecord()

	// range over the fields of the log line
	for i, field := range fields {
		if values[i] == "-" {
			// If a field is not applicable or could not be computed for a
			// specific record, the record displays a '-' symbol for that entry.
			//
			// See https://docs.aws.amazon.com/vpc/latest/userguide/flow-log-records.html.
			continue
		}

		_, err := handleField(field, values[i], record, key)
		if err != nil {
			return err
		}
	}

	scopeLogs := v.getScopeLogs(*key, scopeLogsByResource)
	rScope := scopeLogs.LogRecords().AppendEmpty()
	record.MoveTo(rScope)

	return nil
}

// getScopeLogs for the given key. If it does not exist yet,
// create new scope logs, and add the key to the logs map.
func (v *vpcFlowLogUnmarshaler) getScopeLogs(key resourceKey, logs map[resourceKey]plog.ScopeLogs) plog.ScopeLogs {
	scopeLogs, ok := logs[key]
	if !ok {
		scopeLogs = plog.NewScopeLogs()
		scopeLogs.Scope().SetName(metadata.ScopeName)
		scopeLogs.Scope().SetVersion(v.buildInfo.Version)
		logs[key] = scopeLogs
	}
	return scopeLogs
}

// handleField analyzes the given field and it either
// adds its value to the resourceKey or puts the
// field and its value in the attributes map. If the
// field is not recognized, it returns false.
func handleField(field string, value string, record plog.LogRecord, key *resourceKey) (bool, error) {
	// convert string to number and add the
	// value to an attribute
	addNumber := func(field, str, attrName string) error {
		n, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return fmt.Errorf("%q field in log file is not a number", field)
		}
		record.Attributes().PutInt(attrName, n)
		return nil
	}

	switch field {
	case "account-id":
		key.accountID = value
	case "vpc-id":
		record.Attributes().PutStr("aws.vpc.id", value)
	case "subnet-id":
		record.Attributes().PutStr("aws.subnet.id", value)
	case "instance-id":
		record.Attributes().PutStr("aws.instance.id", value)
	case "az-id":
		record.Attributes().PutStr("aws.az.id", value)
	case "ecs-cluster-arn":
		record.Attributes().PutStr(conventions.AttributeAWSECSClusterARN, value)
	case "ecs-task-arn":
		record.Attributes().PutStr(conventions.AttributeAWSECSTaskARN, value)
	case "interface-id":
		record.Attributes().PutStr("aws.eni.id", value)
	case "pkt-srcaddr":
		record.Attributes().PutStr(conventions.AttributeSourceAddress, value)
	case "pkt-dstaddr":
		record.Attributes().PutStr(conventions.AttributeDestinationAddress, value)
	case "srcport":
		if err := addNumber(field, value, conventions.AttributeSourcePort); err != nil {
			return false, err
		}
	case "dstport":
		if err := addNumber(field, value, conventions.AttributeDestinationPort); err != nil {
			return false, err
		}
	case "protocol":
		if err := addNumber(field, value, "network.protocol.number"); err != nil {
			return false, err
		}
	case "type":
		record.Attributes().PutStr(conventions.AttributeNetworkType, strings.ToLower(value))
	case "region":
		key.region = value
	case "flow-direction":
		record.Attributes().PutStr(conventions.AttributeNetworkIoDirection, value)
	case "ecs-task-id":
		record.Attributes().PutStr(conventions.AttributeAWSECSTaskID, value)
	case "version":
		if err := addNumber(field, value, "aws.flow.log.version"); err != nil {
			return false, err
		}
	case "srcaddr":
		record.Attributes().PutStr("source.layer.address", value)
	case "dstaddr":
		record.Attributes().PutStr("destination.layer.address", value)
	case "packets":
		if err := addNumber(field, value, "aws.flow.log.packets"); err != nil {
			return false, err
		}
	case "bytes":
		if err := addNumber(field, value, "aws.flow.log.bytes"); err != nil {
			return false, err
		}
	case "start":
		if err := addNumber(field, value, "aws.flow.log.start"); err != nil {
			return false, err
		}
	case "end":
		if err := addNumber(field, value, "aws.flow.log.end"); err != nil {
			return false, err
		}
	case "action":
		record.Attributes().PutStr("aws.flow.log.action", value)
	case "log-status":
		record.Attributes().PutStr("aws.flow.log.status", value)
	case "tcp-flags":
		record.Attributes().PutStr("network.tcp.flags", value)
	case "sublocation-type":
		record.Attributes().PutStr("aws.sublocation.type", value)
	case "sublocation-id":
		record.Attributes().PutStr("aws.sublocation.id", value)
	case "pkt-src-aws-service":
		record.Attributes().PutStr("aws.flow.log.source.service", value)
	case "pkt-dst-aws-service":
		record.Attributes().PutStr("aws.flow.log.destination.service", value)
	case "traffic-path":
		record.Attributes().PutStr("aws.flow.log.traffic.path", value)
	case "ecs-cluster-name":
		record.Attributes().PutStr("aws.ecs.cluster.name", value)
	case "ecs-container-instance-arn":
		record.Attributes().PutStr("aws.ecs.container.instance.arn", value)
	case "ecs-container-instance-id":
		record.Attributes().PutStr("aws.ecs.container.instance.id", value)
	case "ecs-container-id":
		record.Attributes().PutStr("aws.ecs.container.id", value)
	case "ecs-second-container-id":
		record.Attributes().PutStr("aws.ecs.second.container.arn", value)
	case "ecs-service-name":
		record.Attributes().PutStr("aws.ecs.service.name", value)
	case "ecs-task-definition-arn":
		record.Attributes().PutStr("aws.ecs.task.definition.arn", value)
	case "reject-reason":
		record.Attributes().PutStr("aws.flow.log.reject_reason", value)
	default:
		return false, nil
	}

	return true, nil
}
