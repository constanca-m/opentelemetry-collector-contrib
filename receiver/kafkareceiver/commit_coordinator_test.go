// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestCommitCoordinatorReport(t *testing.T) {
	cases := []struct {
		name     string
		existing *kgo.Record
		report   *kgo.Record
		want     *kgo.Record
	}{
		{
			name:   "stores first record",
			report: commitRecord("t", 0, 5),
			want:   commitRecord("t", 0, 5),
		},
		{
			name:     "keeps later offset",
			existing: commitRecord("t", 0, 5),
			report:   commitRecord("t", 0, 8),
			want:     commitRecord("t", 0, 8),
		},
		{
			name:     "keeps existing when incoming is older",
			existing: commitRecord("t", 0, 8),
			report:   commitRecord("t", 0, 5),
			want:     commitRecord("t", 0, 8),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			coordinator := newCommitCoordinator(t.Context())
			tp := topicPartition{topic: "t", partition: 0}
			if tc.existing != nil {
				coordinator.report(tp, tc.existing)
			}
			coordinator.report(tp, tc.report)
			require.Equal(t, []*kgo.Record{tc.want}, coordinator.takeAll())
		})
	}
}

func TestCommitCoordinatorTakeAndRestore(t *testing.T) {
	t.Run("take removes only requested partitions", func(t *testing.T) {
		coordinator := newCommitCoordinator(t.Context())
		coordinator.report(topicPartition{topic: "t", partition: 0}, commitRecord("t", 0, 1))
		coordinator.report(topicPartition{topic: "t", partition: 1}, commitRecord("t", 1, 2))

		got := coordinator.take([]topicPartition{{topic: "t", partition: 0}})
		require.Equal(t, []*kgo.Record{commitRecord("t", 0, 1)}, got)
		require.Equal(t, []*kgo.Record{commitRecord("t", 1, 2)}, coordinator.takeAll())
	})

	t.Run("take of empty list removes nothing", func(t *testing.T) {
		coordinator := newCommitCoordinator(t.Context())
		coordinator.report(topicPartition{topic: "t", partition: 0}, commitRecord("t", 0, 1))
		require.Empty(t, coordinator.take(nil))
		require.Equal(t, []*kgo.Record{commitRecord("t", 0, 1)}, coordinator.takeAll())
	})

	t.Run("restore keeps a newer report", func(t *testing.T) {
		coordinator := newCommitCoordinator(t.Context())
		coordinator.restore([]*kgo.Record{commitRecord("t", 0, 1)})
		coordinator.report(topicPartition{topic: "t", partition: 0}, commitRecord("t", 0, 4))
		require.Equal(t, []*kgo.Record{commitRecord("t", 0, 4)}, coordinator.takeAll())
	})

	t.Run("restore stores failed record when nothing newer exists", func(t *testing.T) {
		coordinator := newCommitCoordinator(t.Context())
		failed := commitRecord("t", 0, 1)
		coordinator.restore([]*kgo.Record{failed})
		require.Equal(t, []*kgo.Record{failed}, coordinator.takeAll())
	})

	t.Run("restore does not notify", func(t *testing.T) {
		coordinator := newCommitCoordinator(t.Context())
		tp := topicPartition{topic: "t", partition: 0}
		coordinator.report(tp, commitRecord("t", 0, 1))
		<-coordinator.notify
		coordinator.restore(coordinator.takeAll())
		select {
		case <-coordinator.notify:
			t.Fatal("restore must not notify")
		default:
		}
	})
}

func TestCommitCoordinatorPause(t *testing.T) {
	coordinator := newCommitCoordinator(t.Context())
	coordinator.pause()
	ok, err := coordinator.withAttempt(time.Second, func(context.Context) error {
		t.Fatal("attempt must not start while paused")
		return nil
	})
	require.NoError(t, err)
	require.False(t, ok)
}

func commitRecord(topic string, partition int32, offset int64) *kgo.Record {
	return &kgo.Record{
		Topic:     topic,
		Partition: partition,
		Offset:    offset,
	}
}
