// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkagroupbalancer_test

import (
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/collector/extension"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/kafkagroupbalancer"
)

// Compile-time assertions: GroupBalancer must be a superset of both
// extension.Extension and kgo.GroupBalancer so that implementations satisfy
// the kafkareceiver's runtime type assertions.
var (
	_ extension.Extension = kafkagroupbalancer.GroupBalancer(nil)
	_ kgo.GroupBalancer   = kafkagroupbalancer.GroupBalancer(nil)
)
