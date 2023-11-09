// Copyright 2021 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"

	"github.com/pingcap/tidb/br/pkg/storage"
	cerror "github.com/pingcap/tiflow/pkg/errors"
	"github.com/pingcap/tiflow/pkg/redo"
	"github.com/pingcap/tiflow/pkg/util"
)

// ConsistentConfig represents replication consistency config for a changefeed.
type ConsistentConfig struct {
<<<<<<< HEAD
	Level             string `toml:"level" json:"level"`
	MaxLogSize        int64  `toml:"max-log-size" json:"max-log-size"`
	FlushIntervalInMs int64  `toml:"flush-interval" json:"flush-interval"`
	Storage           string `toml:"storage" json:"storage"`
	UseFileBackend    bool   `toml:"use-file-backend" json:"use-file-backend"`
=======
	Level                 string `toml:"level" json:"level"`
	MaxLogSize            int64  `toml:"max-log-size" json:"max-log-size"`
	FlushIntervalInMs     int64  `toml:"flush-interval" json:"flush-interval"`
	MetaFlushIntervalInMs int64  `toml:"meta-flush-interval" json:"meta-flush-interval"`
	EncodingWorkerNum     int    `toml:"encoding-worker-num" json:"encoding-worker-num"`
	FlushWorkerNum        int    `toml:"flush-worker-num" json:"flush-worker-num"`
	Storage               string `toml:"storage" json:"storage"`
	UseFileBackend        bool   `toml:"use-file-backend" json:"use-file-backend"`
>>>>>>> 028b3afe2e (redo(ticdc): add writer related parameter (#10025))
}

// ValidateAndAdjust validates the consistency config and adjusts it if necessary.
func (c *ConsistentConfig) ValidateAndAdjust() error {
	if !redo.IsConsistentEnabled(c.Level) {
		return nil
	}

	if c.MaxLogSize == 0 {
		c.MaxLogSize = redo.DefaultMaxLogSize
	}

	if c.FlushIntervalInMs == 0 {
		c.FlushIntervalInMs = redo.DefaultFlushIntervalInMs
	}
	if c.FlushIntervalInMs < redo.MinFlushIntervalInMs {
		return cerror.ErrInvalidReplicaConfig.FastGenByArgs(
			fmt.Sprintf("The consistent.flush-interval:%d must be equal or greater than %d",
				c.FlushIntervalInMs, redo.MinFlushIntervalInMs))
	}

<<<<<<< HEAD
=======
	if c.MetaFlushIntervalInMs == 0 {
		c.MetaFlushIntervalInMs = redo.DefaultMetaFlushIntervalInMs
	}
	if c.MetaFlushIntervalInMs < redo.MinFlushIntervalInMs {
		return cerror.ErrInvalidReplicaConfig.FastGenByArgs(
			fmt.Sprintf("The consistent.meta-flush-interval:%d must be equal or greater than %d",
				c.MetaFlushIntervalInMs, redo.MinFlushIntervalInMs))
	}

	if c.EncodingWorkerNum == 0 {
		c.EncodingWorkerNum = redo.DefaultEncodingWorkerNum
	}
	if c.FlushWorkerNum == 0 {
		c.FlushWorkerNum = redo.DefaultFlushWorkerNum
	}

>>>>>>> 028b3afe2e (redo(ticdc): add writer related parameter (#10025))
	uri, err := storage.ParseRawURL(c.Storage)
	if err != nil {
		return cerror.ErrInvalidReplicaConfig.GenWithStackByArgs(
			fmt.Sprintf("invalid storage uri: %s", c.Storage))
	}
	return redo.ValidateStorage(uri)
}

// MaskSensitiveData masks sensitive data in ConsistentConfig
func (c *ConsistentConfig) MaskSensitiveData() {
	c.Storage = util.MaskSensitiveDataInURI(c.Storage)
}
