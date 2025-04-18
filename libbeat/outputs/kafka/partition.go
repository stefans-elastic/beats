// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package kafka

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"math/rand/v2"
	"strconv"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/sarama"
)

type partitionBuilder func(*logp.Logger, *config.C) (func() partitioner, error)

type partitioner func(*message, int32) (int32, error)

// stablePartitioner re-uses last configured partition in case of event being
// repartitioned (on retry from libbeat).
type messagePartitioner struct {
	p          partitioner
	reachable  bool
	partitions int32 // number of partitions seen last
}

func makePartitioner(
	log *logp.Logger,
	partition map[string]*config.C,
) (sarama.PartitionerConstructor, error) {
	mkStrategy, reachable, err := initPartitionStrategy(log, partition)
	if err != nil {
		return nil, err
	}

	return func(topic string) sarama.Partitioner {
		return &messagePartitioner{
			p:         mkStrategy(),
			reachable: reachable,
		}
	}, nil
}

var partitioners = map[string]partitionBuilder{
	"random":      cfgRandomPartitioner,
	"round_robin": cfgRoundRobinPartitioner,
	"hash":        cfgHashPartitioner,
}

func initPartitionStrategy(
	log *logp.Logger,
	partition map[string]*config.C,
) (func() partitioner, bool, error) {
	if len(partition) == 0 {
		// default use `hash` partitioner + all partitions (block if unreachable)
		return makeHashPartitioner, false, nil
	}

	if len(partition) > 1 {
		return nil, false, errors.New("Too many partitioners")
	}

	// extract partitioner from config
	var name string
	var config *config.C
	for n, c := range partition {
		name, config = n, c
	}

	// instantiate partitioner strategy
	mk := partitioners[name]
	if mk == nil {
		return nil, false, fmt.Errorf("unknown kafka partition mode %v", name)
	}
	constr, err := mk(log, config)
	if err != nil {
		return nil, false, err
	}

	// parse shared config
	cfg := struct {
		Reachable bool `config:"reachable_only"`
	}{
		Reachable: false,
	}
	err = config.Unpack(&cfg)
	if err != nil {
		return nil, false, err
	}

	return constr, cfg.Reachable, nil
}

func (p *messagePartitioner) RequiresConsistency() bool { return !p.reachable }
func (p *messagePartitioner) Partition(
	libMsg *sarama.ProducerMessage,
	numPartitions int32,
) (int32, error) {
	msg, ok := libMsg.Metadata.(*message)
	if !ok {
		return 0, fmt.Errorf("failed to assert libMsg.Metadata to *message")
	}

	if numPartitions == p.partitions { // if reachable is false, this is always true
		if 0 <= msg.partition && msg.partition < numPartitions {
			return msg.partition, nil
		}
	}

	partition, err := p.p(msg, numPartitions)
	if err != nil {
		return 0, nil //nolint:nilerr //ignoring this error
	}

	msg.partition = partition

	if _, err := msg.data.Cache.Put("partition", partition); err != nil {
		return 0, fmt.Errorf("setting kafka partition in publisher event failed: %w", err)
	}

	p.partitions = numPartitions
	return msg.partition, nil
}

func cfgRandomPartitioner(_ *logp.Logger, config *config.C) (func() partitioner, error) {
	cfg := struct {
		GroupEvents int `config:"group_events" validate:"min=1"`
	}{
		GroupEvents: 1,
	}
	if err := config.Unpack(&cfg); err != nil {
		return nil, err
	}

	return func() partitioner {
		N := cfg.GroupEvents
		count := cfg.GroupEvents
		partition := int32(0)

		return func(_ *message, numPartitions int32) (int32, error) {
			if N == count {
				count = 0
				partition = rand.Int32N(numPartitions)
			}
			count++
			return partition, nil
		}
	}, nil
}

func cfgRoundRobinPartitioner(_ *logp.Logger, config *config.C) (func() partitioner, error) {
	cfg := struct {
		GroupEvents int `config:"group_events" validate:"min=1"`
	}{
		GroupEvents: 1,
	}
	if err := config.Unpack(&cfg); err != nil {
		return nil, err
	}

	return func() partitioner {
		N := cfg.GroupEvents
		count := N
		partition := rand.Int32()

		return func(_ *message, numPartitions int32) (int32, error) {
			if N == count {
				count = 0
				if partition++; partition >= numPartitions {
					partition = 0
				}
			}
			count++
			return partition, nil
		}
	}, nil
}

func cfgHashPartitioner(log *logp.Logger, config *config.C) (func() partitioner, error) {
	cfg := struct {
		Hash   []string `config:"hash"`
		Random bool     `config:"random"`
	}{
		Random: true,
	}
	if err := config.Unpack(&cfg); err != nil {
		return nil, err
	}

	if len(cfg.Hash) == 0 {
		return makeHashPartitioner, nil
	}

	return func() partitioner {
		return makeFieldsHashPartitioner(log, cfg.Hash, !cfg.Random)
	}, nil
}

func makeHashPartitioner() partitioner {
	hasher := fnv.New32a()

	return func(msg *message, numPartitions int32) (int32, error) {
		if msg.key == nil {
			return rand.Int32N(numPartitions), nil
		}

		hash := msg.hash
		if hash == 0 {
			hasher.Reset()
			if _, err := hasher.Write(msg.key); err != nil {
				return -1, err
			}
			msg.hash = hasher.Sum32()
			hash = msg.hash
		}

		// create positive hash value
		return hash2Partition(hash, numPartitions)
	}
}

func makeFieldsHashPartitioner(log *logp.Logger, fields []string, dropFail bool) partitioner {
	hasher := fnv.New32a()

	return func(msg *message, numPartitions int32) (int32, error) {
		hash := msg.hash
		if hash == 0 {
			hasher.Reset()

			var err error
			for _, field := range fields {
				err = hashFieldValue(hasher, msg.data.Content.Fields, field)
				if err != nil {
					break
				}
			}

			if err != nil {
				if dropFail {
					log.Errorf("Hashing partition key failed: %+v", err)
					return -1, err
				}

				msg.hash = rand.Uint32()
			} else {
				msg.hash = hasher.Sum32()
			}
			hash = msg.hash
		}

		return hash2Partition(hash, numPartitions)
	}
}

func hash2Partition(hash uint32, numPartitions int32) (int32, error) {
	p := int32(hash) & 0x7FFFFFFF
	return p % numPartitions, nil
}

func hashFieldValue(h hash.Hash32, event mapstr.M, field string) error {
	type stringer interface {
		String() string
	}

	type hashable interface {
		Hash32(h hash.Hash32) error
	}

	v, err := event.GetValue(field)
	if err != nil {
		return err
	}

	switch s := v.(type) {
	case hashable:
		err = s.Hash32(h)
	case string:
		_, err = h.Write([]byte(s))
	case []byte:
		_, err = h.Write(s)
	case stringer:
		_, err = h.Write([]byte(s.String()))
	case int8, int16, int32, int64, int,
		uint8, uint16, uint32, uint64, uint:
		err = binary.Write(h, binary.LittleEndian, v)
	case float32:
		tmp := strconv.FormatFloat(float64(s), 'g', -1, 32)
		_, err = h.Write([]byte(tmp))
	case float64:
		tmp := strconv.FormatFloat(s, 'g', -1, 32)
		_, err = h.Write([]byte(tmp))
	default:
		// try to hash using reflection:
		err = binary.Write(h, binary.LittleEndian, v)
		if err != nil {
			err = fmt.Errorf("can not hash key '%v' of unknown type", field)
		}
	}
	return err
}
