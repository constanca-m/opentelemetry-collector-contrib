// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package kafkagroupbalancer defines the GroupBalancer extension interface,
// which allows custom Kafka consumer-group partition assignment strategies to
// be plugged into kafkareceiver.
package kafkagroupbalancer // import "github.com/open-telemetry/opentelemetry-collector-contrib/extension/kafkagroupbalancer"

import (
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/collector/extension"
)

// GroupBalancer is an extension that supplies a custom Kafka consumer-group
// partition assignment strategy. Implementations must satisfy both
// extension.Extension (Start/Shutdown lifecycle) and kgo.GroupBalancer
// (the franz-go partition-assignment interface).
type GroupBalancer interface {
	extension.Extension
	kgo.GroupBalancer
}
