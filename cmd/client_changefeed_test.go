// Copyright 2020 PingCAP, Inc.
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

package cmd

import (
	"context"
	"io/ioutil"
	"path/filepath"

	"github.com/coreos/go-semver/semver"
	"github.com/pingcap/check"
	"github.com/pingcap/ticdc/pkg/util/testleak"
	"github.com/pingcap/ticdc/pkg/version"
	"github.com/spf13/cobra"
)

type clientChangefeedSuite struct{}

var _ = check.Suite(&clientChangefeedSuite{})

func (s *clientChangefeedSuite) TestVerifyChangefeedParams(c *check.C) {
	defer testleak.AfterTest(c)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := &cobra.Command{}

	dir := c.MkDir()
	path := filepath.Join(dir, "config.toml")
	content := `
enable-old-value = false
`
	err := ioutil.WriteFile(path, []byte(content), 0o644)
	c.Assert(err, check.IsNil)

	sinkURI = "blackhole:///?protocol=maxwell"
<<<<<<< HEAD
	info, err := verifyChangefeedParameters(ctx, cmd, false /* isCreate */, nil)
=======
	info, err := verifyChangefeedParameters(ctx, cmd, false /* isCreate */, nil, version.TiCDCClusterVersionUnknown)
>>>>>>> 04fa75b2a (version: prohibit operating TiCDC clusters across major and minor versions (#2481))
	c.Assert(err, check.IsNil)
	c.Assert(info.Config.EnableOldValue, check.IsTrue)
	c.Assert(info.SortDir, check.Equals, "")

	sinkURI = ""
<<<<<<< HEAD
	_, err = verifyChangefeedParameters(ctx, cmd, true /* isCreate */, nil)
	c.Assert(err, check.NotNil)

	c.Assert(info.Config.EnableOldValue, check.IsTrue)
=======
	_, err = verifyChangefeedParameters(ctx, cmd, true /* isCreate */, nil, version.TiCDCClusterVersionUnknown)
	c.Assert(err, check.NotNil)

	sinkURI = "blackhole:///"
	info, err = verifyChangefeedParameters(ctx, cmd, false /* isCreate */, nil, version.TiCDCClusterVersion{Version: semver.New("4.0.0")})
	c.Assert(err, check.IsNil)
	c.Assert(info.Config.EnableOldValue, check.IsFalse)
	c.Assert(info.Engine, check.Equals, model.SortInMemory)
>>>>>>> 04fa75b2a (version: prohibit operating TiCDC clusters across major and minor versions (#2481))

	sortDir = "/tidb/data"
	pdCli = &mockPDClient{}
	disableGCSafePointCheck = true
<<<<<<< HEAD
	_, err = verifyChangefeedParameters(ctx, cmd, false, nil)
	c.Assert(err, check.ErrorMatches, "*Creating changefeed with `--sort-dir`, it's invalid*")
	_, err = verifyChangefeedParameters(ctx, cmd, true, nil)
	c.Assert(err, check.NotNil)

	sortDir = ""
	sinkURI = "blackhole:///"
	_, err = verifyChangefeedParameters(ctx, cmd, false, nil)
=======
	_, err = verifyChangefeedParameters(ctx, cmd, false, nil, version.TiCDCClusterVersionUnknown)
	c.Assert(err, check.ErrorMatches, "*Creating changefeed with `--sort-dir`, it's invalid*")
	_, err = verifyChangefeedParameters(ctx, cmd, true, nil, version.TiCDCClusterVersionUnknown)
	c.Assert(err, check.NotNil)

	sortDir = ""
	_, err = verifyChangefeedParameters(ctx, cmd, false, nil, version.TiCDCClusterVersionUnknown)
>>>>>>> 04fa75b2a (version: prohibit operating TiCDC clusters across major and minor versions (#2481))
	c.Assert(err, check.IsNil)
}
