package vpcflowlog

import (
	"bytes"
	"fmt"
	"github.com/parquet-go/parquet-go"
	"io"
)

type vpcFlowLogRecord struct {
	Version     *int32  `parquet:"version"`
	AccountID   *string `parquet:"account_id"`
	InterfaceID *string `parquet:"interface_id"`
	SrcAddr     *string `parquet:"srcaddr"`
	DstAddr     *string `parquet:"dstaddr"`
	SrcPort     *int32  `parquet:"srcport"`
	DstPort     *int32  `parquet:"dstport"`
	Protocol    *int32  `parquet:"protocol"`
	Packets     *int64  `parquet:"packets"`
	Bytes       *int64  `parquet:"bytes"`
	Start       *int64  `parquet:"start"`
	End         *int64  `parquet:"end"`
	Action      *string `parquet:"action"`
	LogStatus   *string `parquet:"log_status"`
	//VpcID                   *string `parquet:"vpc_id"`
	//SubnetID                *string `parquet:"subnet_id"`
	//InstanceID              *string `parquet:"instance_id"`
	//TcpFlags                *int32  `parquet:"tcp_flags"`
	//Type                    *string `parquet:"type"`
	//PktSrcAddr              *string `parquet:"pkt_srcaddr"`
	//PktDstAddr              *string `parquet:"pkt_dstaddr"`
	//Region                  *string `parquet:"region"`
	//AzID                    *string `parquet:"az_id"`
	//SublocationType         *string `parquet:"sublocation_type"`
	//SublocationID           *string `parquet:"sublocation_id"`
	//PktSrcAwsService        *string `parquet:"pkt_src_aws_service"`
	//PktDstAwsService        *string `parquet:"pkt_dst_aws_service"`
	//FlowDirection           *string `parquet:"flow_direction"`
	//TrafficPath             *int32  `parquet:"traffic_path"`
	//ECSClusterArn           *string `parquet:"ecs_cluster_arn"`
	//ECSClusterName          *string `parquet:"ecs_cluster_name"`
	//ECSContainerInstanceARN *string `parquet:"ecs_container_instance_arn"`
	//ECSContainerInstanceID  *string `parquet:"ecs_container_instance_id"`
	//ECSSecondContainerID    *string `parquet:"ecs_second_container_id"`
	//ECSServiceName          *string `parquet:"ecs_service_name"`
	//ECSTaskDefinitionARN    *string `parquet:"ecs_task_definition_arn"`
	//ECSTaskARN              *string `parquet:"ecs_task_arn"`
	//ECSTaskID               *string `parquet:"ecs_task_id"`
	//RejectReason            *string `parquet:"reject_reason"`
}

/*type vpcFlowLogRecord struct {
	Version     *int32  `parquet:"version"`
	AccountID   *string `parquet:"account_id"`
	InterfaceID string  `parquet:"interface_id"`
	SrcAddr     string  `parquet:"srcaddr"`
	DstAddr     string  `parquet:"dstaddr"`
	SrcPort     *int32  `parquet:"srcport,optional"` // Optional fields as pointers
	DstPort     *int32  `parquet:"dstport,optional"`
	Protocol    *int32  `parquet:"protocol,optional"`
	Packets     *int64  `parquet:"packets,optional"`
	Bytes       *int64  `parquet:"bytes,optional"`
	Start       int64   `parquet:"start"`
	End         int64   `parquet:"end"`
	Action      string  `parquet:"action"`
	LogStatus   string  `parquet:"log_status"`
}*/

func readParquetContent(raw []byte) error {
	// 1. First verify the file magic number
	if len(raw) < 4 || string(raw[:4]) != "PAR1" {
		return fmt.Errorf("not a valid parquet file")
	}

	mapReader := parquet.NewReader(bytes.NewReader(raw))
	defer mapReader.Close()

	for {
		row := make(map[string]any)
		err := mapReader.Read(&row)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("map read failed: %w", err)
		}
		fmt.Printf("%+v\n", row)
	}

	structReader := parquet.NewReader(bytes.NewReader(raw))
	defer structReader.Close()

	for {
		var record vpcFlowLogRecord
		err := structReader.Read(&record)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("struct read failed: %w", err)
		}

		fmt.Printf("Version: %d\n", *record.Version)
		fmt.Printf("AccountID: %s\n", *record.AccountID)
		fmt.Printf("InterfaceID: %s\n", *record.InterfaceID)
		fmt.Printf("SrcAddr: %s\n", *record.SrcAddr)
		fmt.Println("-----")
	}

	return nil
}
