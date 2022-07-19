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

package entry

import (
	"github.com/pingcap/check"
	ticonfig "github.com/pingcap/tidb/config"
	tiddl "github.com/pingcap/tidb/ddl"
	"github.com/pingcap/tidb/domain"
	"github.com/pingcap/tidb/kv"
	timeta "github.com/pingcap/tidb/meta"
	timodel "github.com/pingcap/tidb/parser/model"
	"github.com/pingcap/tidb/session"
	"github.com/pingcap/tidb/store/mockstore"
	"github.com/pingcap/tidb/util/testkit"
	"github.com/tikv/client-go/v2/oracle"
)

// SchemaTestHelper is a test helper for schema which creates an internal tidb instance to generate DDL jobs with meta information
type SchemaTestHelper struct {
	c       *check.C
	tk      *testkit.TestKit
	storage kv.Storage
	domain  *domain.Domain
}

// NewSchemaTestHelper creates a SchemaTestHelper
func NewSchemaTestHelper(c *check.C) *SchemaTestHelper {
	store, err := mockstore.NewMockStore()
	c.Assert(err, check.IsNil)
	ticonfig.UpdateGlobal(func(conf *ticonfig.Config) {
		conf.AlterPrimaryKey = true
	})
	session.SetSchemaLease(0)
	session.DisableStats4Test()
	domain, err := session.BootstrapSession(store)
	c.Assert(err, check.IsNil)
	domain.SetStatsUpdating(true)
	tk := testkit.NewTestKit(c, store)
	return &SchemaTestHelper{
		c:       c,
		tk:      tk,
		storage: store,
		domain:  domain,
	}
}

// DDL2Job executes the DDL stmt and returns the DDL job
func (s *SchemaTestHelper) DDL2Job(ddl string) *timodel.Job {
	s.tk.MustExec(ddl)
<<<<<<< HEAD
	jobs, err := s.GetCurrentMeta().GetLastNHistoryDDLJobs(1)
	s.c.Assert(err, check.IsNil)
	s.c.Assert(jobs, check.HasLen, 1)
	return jobs[0]
}

// Storage return the tikv storage
=======
	jobs, err := tiddl.GetLastNHistoryDDLJobs(s.GetCurrentMeta(), 1)
	require.Nil(s.t, err)
	require.Len(s.t, jobs, 1)
	return jobs[0]
}

// DDL2Jobs executes the DDL statement and return the corresponding DDL jobs.
// It is mainly used for "DROP TABLE" and "DROP VIEW" statement because
// multiple jobs will be generated after executing these two types of
// DDL statements.
func (s *SchemaTestHelper) DDL2Jobs(ddl string, jobCnt int) []*timodel.Job {
	s.tk.MustExec(ddl)
	jobs, err := tiddl.GetLastNHistoryDDLJobs(s.GetCurrentMeta(), jobCnt)
	require.Nil(s.t, err)
	require.Len(s.t, jobs, jobCnt)
	return jobs
}

// Storage returns the tikv storage
>>>>>>> b4bbb8623 (db(dm): use `net.JoinHostPort` to generate host-port part of URI (#6248))
func (s *SchemaTestHelper) Storage() kv.Storage {
	return s.storage
}

// GetCurrentMeta return the current meta snapshot
func (s *SchemaTestHelper) GetCurrentMeta() *timeta.Meta {
	ver, err := s.storage.CurrentVersion(oracle.GlobalTxnScope)
	s.c.Assert(err, check.IsNil)
	return timeta.NewSnapshotMeta(s.storage.GetSnapshot(ver))
}

// Close closes the helper
func (s *SchemaTestHelper) Close() {
	s.domain.Close()
	s.storage.Close() //nolint:errcheck
}
