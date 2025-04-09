// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package vpcflowlog // import "github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding/awslogsencodingextension/internal/unmarshaler/vpc-flow-log"

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/parquet-go/parquet-go"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/compress/gzip"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	conventions "go.opentelemetry.io/collector/semconv/v1.27.0"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding/awslogsencodingextension/internal/metadata"
)

const (
	fileFormatPlainText = "plain-text"
	fileFormatParquet   = "parquet"

	attributeAWSVPCID                     = "aws.vpc.id"
	attributeAWSVPCSubnetID               = "aws.vpc.subnet.id"
	attributeAWSAZID                      = "aws.az.id"
	attributeAWSVPCFlowLogVersion         = "aws.vpc.flow.log.version"
	attributeAWSVPCFlowPackets            = "aws.vpc.flow.packets"
	attributeAWSVPCFlowBytes              = "aws.vpc.flow.bytes"
	attributeAWSVPCFlowStart              = "aws.vpc.flow.start"
	attributeAWSVPCFlowAction             = "aws.vpc.flow.action"
	attributeAWSVPCFlowStatus             = "aws.vpc.flow.status"
	attributeNetworkTCPFlags              = "network.tcp.flags"
	attributeAWSSublocationType           = "aws.sublocation.type"
	attributeAWSSublocationID             = "aws.sublocation.id"
	attributeAWSVPCFlowSourceService      = "aws.vpc.flow.source.service"
	attributeAWSVPCFlowDestinationService = "aws.vpc.flow.destination.service"
	attributeAWSVPCFlowTrafficPath        = "aws.vpc.flow.traffic_path"
	attributeAWSVPCFlowRejectReason       = "aws.vpc.flow.reject_reason"

	// TODO Remove once it gets available in conventions
	attributeNetworkInterfaceName = "network.interface.name"

	unknownValue = "-"
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
	logger    *zap.Logger
}

func NewVPCFlowLogUnmarshaler(format string, buildInfo component.BuildInfo, logger *zap.Logger) (plog.Unmarshaler, error) {
	switch format {
	case fileFormatParquet:
		// TODO
		//return nil, errors.New("still needs to be implemented")
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
		logger:     logger,
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
		return v.unmarshalParquetLogs(gzipReader)
	default:
		// not possible, prevent by NewVPCFlowLogUnmarshaler
		return plog.Logs{}, nil
	}
}

// resourceKey stores the account id and region
// of the flow logs. All log lines inside the
// same S3 file come from the same account and
// region.
//
// See https://docs.aws.amazon.com/vpc/latest/userguide/flow-logs-s3-path.html.
type resourceKey struct {
	accountID string
	region    string
}

func (v *vpcFlowLogUnmarshaler) unmarshalPlainTextLogs(reader io.Reader) (plog.Logs, error) {
	scanner := bufio.NewScanner(reader)

	var fields []string
	if scanner.Scan() {
		firstLine := scanner.Text()
		fields = strings.Split(firstLine, " ")
	}

	logs, resourceLogs, scopeLogs := v.createLogs()
	key := &resourceKey{}
	for scanner.Scan() {
		line := scanner.Text()
		if err := v.addToLogs(key, scopeLogs, fields, line); err != nil {
			return plog.Logs{}, err
		}
	}

	if err := scanner.Err(); err != nil {
		return plog.Logs{}, fmt.Errorf("error reading log line: %w", err)
	}

	v.setResourceAttributes(key, resourceLogs)
	return logs, nil
}

// createLogs with the expected fields for the scope logs
func (v *vpcFlowLogUnmarshaler) createLogs() (plog.Logs, plog.ResourceLogs, plog.ScopeLogs) {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	scopeLogs.Scope().SetName(metadata.ScopeName)
	scopeLogs.Scope().SetVersion(v.buildInfo.Version)
	return logs, resourceLogs, scopeLogs
}

// setResourceAttributes based on the resourceKey
func (v *vpcFlowLogUnmarshaler) setResourceAttributes(key *resourceKey, logs plog.ResourceLogs) {
	attr := logs.Resource().Attributes()
	attr.PutStr(conventions.AttributeCloudProvider, conventions.AttributeCloudProviderAWS)
	if key.accountID != "" {
		attr.PutStr(conventions.AttributeCloudAccountID, key.accountID)
	}
	if key.region != "" {
		attr.PutStr(conventions.AttributeCloudRegion, key.region)
	}
}

// address stores the four fields related to the address
// of a VPC flow log: srcaddr, pkt-srcaddr, dstaddr, and
// pkt-dstaddr. We save these fields in a struct, so we
// can use the right naming conventions in the end.
//
// See https://docs.aws.amazon.com/vpc/latest/userguide/flow-logs-records-examples.html#flow-log-example-nat.
type address struct {
	source         string
	pktSource      string
	destination    string
	pktDestination string
}

// addToLogs parses the log line and creates
// a new record log. The record log is added
// to the scope logs of the resource identified
// by the resourceKey created from the values.
func (v *vpcFlowLogUnmarshaler) addToLogs(
	key *resourceKey,
	scopeLogs plog.ScopeLogs,
	fields []string,
	logLine string,
) error {
	record := plog.NewLogRecord()

	addr := &address{}
	for _, field := range fields {
		if logLine == "" {
			return errors.New("log line has less fields than the ones expected")
		}
		var value string
		value, logLine, _ = strings.Cut(logLine, " ")

		if value == unknownValue {
			// If a field is not applicable or could not be computed for a
			// specific record, the record displays a '-' symbol for that entry.
			//
			// See https://docs.aws.amazon.com/vpc/latest/userguide/flow-log-records.html.
			continue
		}

		if strings.HasPrefix(field, "ecs-") {
			v.logger.Warn("currently there is no support for ECS fields")
			continue
		}

		found, err := handleField(field, value, record, addr, key)
		if err != nil {
			return err
		}
		if !found {
			v.logger.Warn("field is not an available field for a flow log record",
				zap.String("field", field),
				zap.String("documentation", "https://docs.aws.amazon.com/vpc/latest/userguide/flow-log-records.html"),
			)
		}
	}

	if logLine != "" {
		return errors.New("log line has more fields than the ones expected")
	}

	// add the address fields with the correct conventions
	// to the log record
	v.handleAddresses(addr, record)
	rScope := scopeLogs.LogRecords().AppendEmpty()
	record.MoveTo(rScope)

	return nil
}

// handleAddresses creates adds the addresses to the log record
func (v *vpcFlowLogUnmarshaler) handleAddresses(addr *address, record plog.LogRecord) {
	localAddrSet := false
	// see example in
	// https://docs.aws.amazon.com/vpc/latest/userguide/flow-logs-records-examples.html#flow-log-example-nat
	if addr.pktSource == "" && addr.source != "" {
		// there is no middle layer, assume "srcaddr" field
		// corresponds to the original source address.
		record.Attributes().PutStr(conventions.AttributeSourceAddress, addr.source)
	} else if addr.pktSource != "" && addr.source != "" {
		record.Attributes().PutStr(conventions.AttributeSourceAddress, addr.pktSource)
		if addr.pktSource != addr.source {
			// srcaddr is the middle layer
			record.Attributes().PutStr(conventions.AttributeNetworkLocalAddress, addr.source)
			localAddrSet = true
		}
	}

	if addr.pktDestination == "" && addr.destination != "" {
		// there is no middle layer, assume "dstaddr" field
		// corresponds to the original destination address.
		record.Attributes().PutStr(conventions.AttributeDestinationAddress, addr.destination)
	} else if addr.pktDestination != "" && addr.destination != "" {
		record.Attributes().PutStr(conventions.AttributeDestinationAddress, addr.pktDestination)
		if addr.pktDestination != addr.destination {
			if localAddrSet {
				v.logger.Warn("unexpected: srcaddr, dstaddr, pkt-srcaddr and pkt-dstaddr are all different")
			}
			// dstaddr is the middle layer
			record.Attributes().PutStr(conventions.AttributeNetworkLocalAddress, addr.destination)
		}
	}
}

// handleField analyzes the given field and it either
// adds its value to the resourceKey or puts the
// field and its value in the attributes map. If the
// field is not recognized, it returns false.
func handleField(
	field string,
	value string,
	record plog.LogRecord,
	addr *address,
	key *resourceKey,
) (bool, error) {
	// convert string to number
	getNumber := func(value string) (int64, error) {
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return -1, fmt.Errorf("%q field in log file is not a number", field)
		}
		return n, nil
	}

	// convert string to number and add the
	// value to an attribute
	addNumber := func(field, value, attrName string) error {
		n, err := getNumber(value)
		if err != nil {
			return fmt.Errorf("%q field in log file is not a number", field)
		}
		record.Attributes().PutInt(attrName, n)
		return nil
	}

	switch field {
	// TODO Add support for ECS fields
	case "srcaddr":
		// handled later
		addr.source = value
	case "pkt-srcaddr":
		// handled later
		addr.pktSource = value
	case "dstaddr":
		// handled later
		addr.destination = value
	case "pkt-dstaddr":
		// handled later
		addr.pktDestination = value

	case "account-id":
		key.accountID = value
	case "vpc-id":
		record.Attributes().PutStr(attributeAWSVPCID, value)
	case "subnet-id":
		record.Attributes().PutStr(attributeAWSVPCSubnetID, value)
	case "instance-id":
		record.Attributes().PutStr(conventions.AttributeHostID, value)
	case "az-id":
		record.Attributes().PutStr(attributeAWSAZID, value)
	case "interface-id":
		record.Attributes().PutStr(attributeNetworkInterfaceName, value)
	case "srcport":
		if err := addNumber(field, value, conventions.AttributeSourcePort); err != nil {
			return false, err
		}
	case "dstport":
		if err := addNumber(field, value, conventions.AttributeDestinationPort); err != nil {
			return false, err
		}
	case "protocol":
		n, err := getNumber(value)
		if err != nil {
			return false, err
		}
		protocolNumber := int(n)
		if protocolNumber < 0 || protocolNumber >= len(protocolNames) {
			return false, fmt.Errorf("protocol number %d does not have a protocol name", protocolNumber)
		}
		record.Attributes().PutStr(conventions.AttributeNetworkProtocolName, protocolNames[protocolNumber])
	case "type":
		record.Attributes().PutStr(conventions.AttributeNetworkType, strings.ToLower(value))
	case "region":
		key.region = value
	case "flow-direction":
		switch value {
		case "ingress":
			record.Attributes().PutStr(conventions.AttributeNetworkIoDirection, "receive")
		case "egress":
			record.Attributes().PutStr(conventions.AttributeNetworkIoDirection, "transmit")
		default:
			return true, fmt.Errorf("value %s not valid for field %s", value, field)
		}
	case "version":
		if err := addNumber(field, value, attributeAWSVPCFlowLogVersion); err != nil {
			return false, err
		}
	case "packets":
		if err := addNumber(field, value, attributeAWSVPCFlowPackets); err != nil {
			return false, err
		}
	case "bytes":
		if err := addNumber(field, value, attributeAWSVPCFlowBytes); err != nil {
			return false, err
		}
	case "start":
		if err := addNumber(field, value, attributeAWSVPCFlowStart); err != nil {
			return false, err
		}
	case "end":
		unixSeconds, err := getNumber(value)
		if err != nil {
			return true, fmt.Errorf("value %s for field %s does not correspond to a valid timestamp", value, field)
		}
		timestamp := time.Unix(unixSeconds, 0)
		record.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
	case "action":
		record.Attributes().PutStr(attributeAWSVPCFlowAction, value)
	case "log-status":
		record.Attributes().PutStr(attributeAWSVPCFlowStatus, value)
	case "tcp-flags":
		record.Attributes().PutStr(attributeNetworkTCPFlags, value)
	case "sublocation-type":
		record.Attributes().PutStr(attributeAWSSublocationType, value)
	case "sublocation-id":
		record.Attributes().PutStr(attributeAWSSublocationID, value)
	case "pkt-src-aws-service":
		record.Attributes().PutStr(attributeAWSVPCFlowSourceService, value)
	case "pkt-dst-aws-service":
		record.Attributes().PutStr(attributeAWSVPCFlowDestinationService, value)
	case "traffic-path":
		record.Attributes().PutStr(attributeAWSVPCFlowTrafficPath, value)
	case "reject-reason":
		record.Attributes().PutStr(attributeAWSVPCFlowRejectReason, value)
	default:
		return false, nil
	}

	return true, nil
}

//go:generate parquetgen -input unmarshaler.go -type vpcFlowLogRecord -package vpcflowlog

// vpcFlowLogRecord represents a flow log.
// See https://docs.aws.amazon.com/vpc/latest/userguide/flow-log-records.html.
type vpcFlowLogRecord struct {
	Version          *int32  `parquet:"version"`
	AccountID        *string `parquet:"account_id"`
	InterfaceID      *string `parquet:"interface_id"`
	SrcAddr          *string `parquet:"srcaddr"`
	DstAddr          *string `parquet:"dstaddr"`
	SrcPort          *int32  `parquet:"srcport"`
	DstPort          *int32  `parquet:"dstport"`
	Protocol         *int32  `parquet:"protocol"`
	Packets          *int64  `parquet:"packets"`
	Bytes            *int64  `parquet:"bytes"`
	Start            *int64  `parquet:"start"`
	End              *int64  `parquet:"end"`
	Action           *string `parquet:"action"`
	LogStatus        *string `parquet:"log_status"`
	VpcID            *string `parquet:"vpc_id"`
	SubnetID         *string `parquet:"subnet_id"`
	InstanceID       *string `parquet:"instance_id"`
	TcpFlags         *int32  `parquet:"tcp_flags"`
	Type             *string `parquet:"type"`
	PktSrcAddr       *string `parquet:"pkt_srcaddr"`
	PktDstAddr       *string `parquet:"pkt_dstaddr"`
	Region           *string `parquet:"region"`
	AzID             *string `parquet:"az_id"`
	SublocationType  *string `parquet:"sublocation_type"`
	SublocationID    *string `parquet:"sublocation_id"`
	PktSrcAwsService *string `parquet:"pkt_src_aws_service"`
	PktDstAwsService *string `parquet:"pkt_dst_aws_service"`
	FlowDirection    *string `parquet:"flow_direction"`
	TrafficPath      *int32  `parquet:"traffic_path"`

	// TODO Handle ECS fields
	ECSClusterArn           *string `parquet:"ecs_cluster_arn"`
	ECSClusterName          *string `parquet:"ecs_cluster_name"`
	ECSContainerInstanceARN *string `parquet:"ecs_container_instance_arn"`
	ECSContainerInstanceID  *string `parquet:"ecs_container_instance_id"`
	ECSSecondContainerID    *string `parquet:"ecs_second_container_id"`
	ECSServiceName          *string `parquet:"ecs_service_name"`
	ECSTaskDefinitionARN    *string `parquet:"ecs_task_definition_arn"`
	ECSTaskARN              *string `parquet:"ecs_task_arn"`
	ECSTaskID               *string `parquet:"ecs_task_id"`
	RejectReason            *string `parquet:"reject_reason"`
}

func (v *vpcFlowLogUnmarshaler) unmarshalParquetLogs(reader io.Reader) (plog.Logs, error) {
	buf := bytes.NewBuffer([]byte{})
	if _, err := io.Copy(buf, reader); err != nil {
		return plog.Logs{}, fmt.Errorf("copy error: %w", err)
	}

	// add optimized settings
	file, err := parquet.OpenFile(
		bytes.NewReader(buf.Bytes()),
		int64(buf.Len()),
		parquet.SkipBloomFilters(true),
		parquet.SkipPageIndex(true),
		parquet.ReadBufferSize(128*1024),
	)
	if err != nil {
		return plog.Logs{}, fmt.Errorf("open file error: %w", err)
	}

	pReader := parquet.NewReader(file)
	defer pReader.Close()

	logs, resourceLogs, scopeLogs := v.createLogs()
	key := &resourceKey{}

	//var emptyRecord vpcFlowLogRecord
	var record vpcFlowLogRecord
	for {
		record = vpcFlowLogRecord{}
		err = pReader.Read(&record)
		if err == io.EOF {
			break
		}
		if err != nil {
			return plog.Logs{}, fmt.Errorf("failed to read row: %w", err)
		}

		if err = v.addToLogsParquet(scopeLogs, record); err != nil {
			return plog.Logs{}, err
		}
	}

	v.setResourceAttributes(key, resourceLogs)
	return logs, nil
}

func putAttributeStr(field string, value *string, record plog.LogRecord) {
	if value == nil {
		return
	}
	if *value == "-" || *value == "" {
		return
	}
	record.Attributes().PutStr(field, *value)
}

func putAttributeNumber(field string, value *int64, record plog.LogRecord) {
	if value == nil {
		return
	}
	record.Attributes().PutInt(field, *value)
}

// addToLogs parses the log line and creates
// a new record log. The record log is added
// to the scope logs of the resource identified
// by the resourceKey created from the values.
func (v *vpcFlowLogUnmarshaler) addToLogsParquet(
	scopeLogs plog.ScopeLogs,
	log vpcFlowLogRecord,
) error {
	record := plog.NewLogRecord()

	putAttributeStr(attributeAWSVPCID, log.VpcID, record)
	putAttributeStr(attributeAWSVPCSubnetID, log.SubnetID, record)
	putAttributeStr(conventions.AttributeHostID, log.InstanceID, record)
	putAttributeStr(attributeAWSAZID, log.AzID, record)
	putAttributeStr(attributeNetworkInterfaceName, log.InterfaceID, record)
	putAttributeStr(attributeAWSVPCFlowAction, log.Action, record)
	putAttributeStr(attributeAWSVPCFlowStatus, log.LogStatus, record)
	putAttributeStr(attributeAWSSublocationType, log.SublocationType, record)
	putAttributeStr(attributeAWSSublocationID, log.SublocationID, record)
	putAttributeStr(attributeAWSVPCFlowSourceService, log.PktSrcAddr, record)
	putAttributeStr(attributeAWSVPCFlowDestinationService, log.PktDstAddr, record)
	putAttributeStr(attributeAWSVPCFlowRejectReason, log.RejectReason, record)
	// TODO Handle address

	/*
			case "tcp-flags":
				record.Attributes().PutStr(attributeNetworkTCPFlags, value) // TODO This seems like a int


			case "traffic-path":
				record.Attributes().PutStr(attributeAWSVPCFlowTrafficPath, value) // TODO Int again


		case "srcport":
			if err := addNumber(field, value, conventions.AttributeSourcePort); err != nil {
				return false, err
			}
		case "dstport":
			if err := addNumber(field, value, conventions.AttributeDestinationPort); err != nil {
				return false, err
			}
		case "protocol":
			n, err := getNumber(value)
			if err != nil {
				return false, err
			}
			protocolNumber := int(n)
			if protocolNumber < 0 || protocolNumber >= len(protocolNames) {
				return false, fmt.Errorf("protocol number %d does not have a protocol name", protocolNumber)
			}
			record.Attributes().PutStr(conventions.AttributeNetworkProtocolName, protocolNames[protocolNumber])
		case "type":
			record.Attributes().PutStr(conventions.AttributeNetworkType, strings.ToLower(value))
		case "region":
			key.region = value
		case "flow-direction":
			switch value {
			case "ingress":
				record.Attributes().PutStr(conventions.AttributeNetworkIoDirection, "receive")
			case "egress":
				record.Attributes().PutStr(conventions.AttributeNetworkIoDirection, "transmit")
			default:
				return true, fmt.Errorf("value %s not valid for field %s", value, field)
			}
		case "version":
			if err := addNumber(field, value, attributeAWSVPCFlowLogVersion); err != nil {
				return false, err
			}
		case "packets":
			if err := addNumber(field, value, attributeAWSVPCFlowPackets); err != nil {
				return false, err
			}
		case "bytes":
			if err := addNumber(field, value, attributeAWSVPCFlowBytes); err != nil {
				return false, err
			}
		case "start":
			if err := addNumber(field, value, attributeAWSVPCFlowStart); err != nil {
				return false, err
			}
		case "end":
			unixSeconds, err := getNumber(value)
			if err != nil {
				return true, fmt.Errorf("value %s for field %s does not correspond to a valid timestamp", value, field)
			}
			timestamp := time.Unix(unixSeconds, 0)
			record.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
		default:
			return false, nil
		}*/

	// add the address fields with the correct conventions
	// to the log record

	getStr := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}

	addr := &address{
		source:         getStr(log.SrcAddr),
		destination:    getStr(log.DstAddr),
		pktSource:      getStr(log.PktSrcAddr),
		pktDestination: getStr(log.PktDstAddr),
	}

	v.handleAddresses(addr, record)
	rScope := scopeLogs.LogRecords().AppendEmpty()
	record.MoveTo(rScope)

	return nil
}
