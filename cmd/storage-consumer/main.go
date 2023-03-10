// Copyright 2022 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pingcap/errors"
	"github.com/pingcap/log"
	"github.com/pingcap/tidb/br/pkg/storage"
	"github.com/pingcap/tiflow/cdc/model"
<<<<<<< HEAD
	"github.com/pingcap/tiflow/cdc/sink"
	"github.com/pingcap/tiflow/cdc/sink/codec"
	"github.com/pingcap/tiflow/cdc/sink/codec/canal"
	"github.com/pingcap/tiflow/cdc/sink/codec/common"
	"github.com/pingcap/tiflow/cdc/sink/codec/csv"
	sinkutil "github.com/pingcap/tiflow/cdc/sinkv2/util"
=======
	"github.com/pingcap/tiflow/cdc/sink/ddlsink"
	ddlfactory "github.com/pingcap/tiflow/cdc/sink/ddlsink/factory"
	dmlfactory "github.com/pingcap/tiflow/cdc/sink/dmlsink/factory"
	"github.com/pingcap/tiflow/cdc/sink/tablesink"
	sinkutil "github.com/pingcap/tiflow/cdc/sink/util"
>>>>>>> 855e4e13bd (consumer(ticdc): support handling DDL events in cloud storage consumer (#8278))
	"github.com/pingcap/tiflow/pkg/cmd/util"
	"github.com/pingcap/tiflow/pkg/config"
	"github.com/pingcap/tiflow/pkg/logutil"
	"github.com/pingcap/tiflow/pkg/quotes"
	psink "github.com/pingcap/tiflow/pkg/sink"
	"github.com/pingcap/tiflow/pkg/sink/cloudstorage"
	putil "github.com/pingcap/tiflow/pkg/util"
	"go.uber.org/zap"
)

var (
	upstreamURIStr   string
	upstreamURI      *url.URL
	downstreamURIStr string
	configFile       string
	logFile          string
	logLevel         string
	flushInterval    time.Duration
	enableProfiling  bool
)

<<<<<<< HEAD
const fakePartitionNumForSchemaFile = -1
=======
const (
	defaultChangefeedName         = "storage-consumer"
	defaultFlushWaitDuration      = 200 * time.Millisecond
	fakePartitionNumForSchemaFile = -1
)
>>>>>>> 855e4e13bd (consumer(ticdc): support handling DDL events in cloud storage consumer (#8278))

func init() {
	flag.StringVar(&upstreamURIStr, "upstream-uri", "", "storage uri")
	flag.StringVar(&downstreamURIStr, "downstream-uri", "", "downstream sink uri")
	flag.StringVar(&configFile, "config", "", "changefeed configuration file")
	flag.StringVar(&logFile, "log-file", "", "log file path")
	flag.StringVar(&logLevel, "log-level", "info", "log level")
	flag.DurationVar(&flushInterval, "flush-interval", 10*time.Second, "flush interval")
	flag.BoolVar(&enableProfiling, "enable-profiling", false, "whether to enable profiling")
	flag.Parse()

	err := logutil.InitLogger(&logutil.Config{
		Level: logLevel,
		File:  logFile,
	})
	if err != nil {
		log.Error("init logger failed", zap.Error(err))
		os.Exit(1)
	}

	uri, err := url.Parse(upstreamURIStr)
	if err != nil {
		log.Error("invalid upstream-uri", zap.Error(err))
		os.Exit(1)
	}
	upstreamURI = uri
	scheme := strings.ToLower(upstreamURI.Scheme)
	if !psink.IsStorageScheme(scheme) {
		log.Error("invalid storage scheme, the scheme of upstream-uri must be file/s3/azblob/gcs")
		os.Exit(1)
	}
}

type schemaPathKey struct {
	schema  string
	table   string
	version int64
}

func (s schemaPathKey) generagteSchemaFilePath() string {
	return fmt.Sprintf("%s/%s/%d/schema.json", s.schema, s.table, s.version)
}

func (s *schemaPathKey) parseSchemaFilePath(path string) error {
	str := `(\w+)\/(\w+)\/(\d+)\/schema.json`
	pathRE, err := regexp.Compile(str)
	if err != nil {
		return err
	}

	matches := pathRE.FindStringSubmatch(path)
	if len(matches) != 4 {
		return fmt.Errorf("cannot match schema path pattern for %s", path)
	}

	version, err := strconv.ParseUint(matches[3], 10, 64)
	if err != nil {
		return err
	}

	*s = schemaPathKey{
		schema:  matches[1],
		table:   matches[2],
		version: int64(version),
	}
	return nil
}

type dmlPathKey struct {
	schemaPathKey
	partitionNum int64
	date         string
}

func (d *dmlPathKey) generateDMLFilePath(idx uint64, extension string) string {
	var elems []string

	elems = append(elems, d.schema)
	elems = append(elems, d.table)
	elems = append(elems, fmt.Sprintf("%d", d.version))

	if d.partitionNum != 0 {
		elems = append(elems, fmt.Sprintf("%d", d.partitionNum))
	}
	if len(d.date) != 0 {
		elems = append(elems, d.date)
	}
	elems = append(elems, fmt.Sprintf("CDC%06d%s", idx, extension))

	return strings.Join(elems, "/")
}

// dml file path pattern is as follows:
// {schema}/{table}/{table-version-separator}/{partition-separator}/{date-separator}/CDC{num}.extension
// in this pattern, partition-separator and date-separator could be empty.
func (d *dmlPathKey) parseDMLFilePath(dateSeparator, path string) (uint64, error) {
	var partitionNum int64

	str := `(\w+)\/(\w+)\/(\d+)\/(\d+\/)*`
	switch dateSeparator {
	case config.DateSeparatorNone.String():
		str += `(\d{4})*`
	case config.DateSeparatorYear.String():
		str += `(\d{4})\/`
	case config.DateSeparatorMonth.String():
		str += `(\d{4}-\d{2})\/`
	case config.DateSeparatorDay.String():
		str += `(\d{4}-\d{2}-\d{2})\/`
	}
	str += `CDC(\d+).\w+`
	pathRE, err := regexp.Compile(str)
	if err != nil {
		return 0, err
	}

	matches := pathRE.FindStringSubmatch(path)
	if len(matches) != 7 {
		return 0, fmt.Errorf("cannot match dml path pattern for %s", path)
	}

	version, err := strconv.ParseInt(matches[3], 10, 64)
	if err != nil {
		return 0, err
	}

	if len(matches[4]) > 0 {
		partitionNum, err = strconv.ParseInt(matches[4], 10, 64)
		if err != nil {
			return 0, err
		}
	}
	fileIdx, err := strconv.ParseUint(strings.TrimLeft(matches[6], "0"), 10, 64)
	if err != nil {
		return 0, err
	}

	*d = dmlPathKey{
		schemaPathKey: schemaPathKey{
			schema:  matches[1],
			table:   matches[2],
			version: version,
		},
		partitionNum: partitionNum,
		date:         matches[5],
	}

	return fileIdx, nil
}

// fileIndexRange defines a range of files. eg. CDC000002.csv ~ CDC000005.csv
type fileIndexRange struct {
	start uint64
	end   uint64
}

type consumer struct {
<<<<<<< HEAD
	sink            sink.Sink
=======
	sinkFactory     *dmlfactory.SinkFactory
	ddlSink         ddlsink.Sink
>>>>>>> 855e4e13bd (consumer(ticdc): support handling DDL events in cloud storage consumer (#8278))
	replicationCfg  *config.ReplicaConfig
	codecCfg        *common.Config
	externalStorage storage.ExternalStorage
	fileExtension   string
	// tableDMLIdxMap maintains a map of <dmlPathKey, max file index>
	tableDMLIdxMap map[dmlPathKey]uint64
	// tableTsMap maintains a map of <TableID, max commit ts>
	tableTsMap       map[model.TableID]uint64
	tableIDGenerator *fakeTableIDGenerator
}

func newConsumer(ctx context.Context) (*consumer, error) {
	replicaConfig := config.GetDefaultReplicaConfig()
	if len(configFile) > 0 {
		err := util.StrictDecodeFile(configFile, "storage consumer", replicaConfig)
		if err != nil {
			log.Error("failed to decode config file", zap.Error(err))
			return nil, err
		}
	}

	err := replicaConfig.ValidateAndAdjust(upstreamURI)
	if err != nil {
		log.Error("failed to validate replica config", zap.Error(err))
		return nil, err
	}

	switch replicaConfig.Sink.Protocol {
	case config.ProtocolCsv.String():
	case config.ProtocolCanalJSON.String():
	default:
		return nil, fmt.Errorf("data encoded in protocol %s is not supported yet",
			replicaConfig.Sink.Protocol)
	}

	protocol, err := config.ParseSinkProtocolFromString(replicaConfig.Sink.Protocol)
	if err != nil {
		return nil, err
	}

	codecConfig := common.NewConfig(protocol)
	err = codecConfig.Apply(upstreamURI, replicaConfig)
	if err != nil {
		return nil, err
	}

	extension := sinkutil.GetFileExtension(protocol)

	storage, err := putil.GetExternalStorageFromURI(ctx, upstreamURIStr)
	if err != nil {
		log.Error("failed to create external storage", zap.Error(err))
		return nil, err
	}

	errCh := make(chan error, 1)
<<<<<<< HEAD
	s, err := sink.New(ctx, model.DefaultChangeFeedID("storage-consumer"),
		downstreamURIStr, config.GetDefaultReplicaConfig(), errCh)
=======
	stdCtx := contextutil.PutChangefeedIDInCtx(ctx,
		model.DefaultChangeFeedID(defaultChangefeedName))
	sinkFactory, err := dmlfactory.New(
		stdCtx,
		downstreamURIStr,
		config.GetDefaultReplicaConfig(),
		errCh,
	)
>>>>>>> 855e4e13bd (consumer(ticdc): support handling DDL events in cloud storage consumer (#8278))
	if err != nil {
		log.Error("failed to create event sink factory", zap.Error(err))
		return nil, err
	}

	ddlSink, err := ddlfactory.New(ctx, downstreamURIStr, config.GetDefaultReplicaConfig())
	if err != nil {
		log.Error("failed to create ddl sink", zap.Error(err))
		return nil, err
	}

	return &consumer{
<<<<<<< HEAD
		sink:            s,
=======
		sinkFactory:     sinkFactory,
		ddlSink:         ddlSink,
>>>>>>> 855e4e13bd (consumer(ticdc): support handling DDL events in cloud storage consumer (#8278))
		replicationCfg:  replicaConfig,
		codecCfg:        codecConfig,
		externalStorage: storage,
		fileExtension:   extension,
<<<<<<< HEAD
		tableIdxMap:     make(map[dmlPathKey]uint64),
		tableTsMap:      make(map[model.TableID]uint64),
=======
		errCh:           errCh,
		tableDMLIdxMap:  make(map[dmlPathKey]uint64),
		tableTsMap:      make(map[model.TableID]model.ResolvedTs),
		tableSinkMap:    make(map[model.TableID]tablesink.TableSink),
>>>>>>> 855e4e13bd (consumer(ticdc): support handling DDL events in cloud storage consumer (#8278))
		tableIDGenerator: &fakeTableIDGenerator{
			tableIDs: make(map[string]int64),
		},
	}, nil
}

// map1 - map2
func diffDMLMaps(map1, map2 map[dmlPathKey]uint64) map[dmlPathKey]fileIndexRange {
	resMap := make(map[dmlPathKey]fileIndexRange)
	for k, v := range map1 {
		if _, ok := map2[k]; !ok {
			resMap[k] = fileIndexRange{
				start: 1,
				end:   v,
			}
		} else if v > map2[k] {
			resMap[k] = fileIndexRange{
				start: map2[k] + 1,
				end:   v,
			}
		}
	}

	return resMap
}

// getNewFiles returns newly created dml files in specific ranges
func (c *consumer) getNewFiles(ctx context.Context) (map[dmlPathKey]fileIndexRange, error) {
	tableDMLMap := make(map[dmlPathKey]fileIndexRange)
	opt := &storage.WalkOption{SubDir: ""}

	origDMLIdxMap := make(map[dmlPathKey]uint64, len(c.tableDMLIdxMap))
	for k, v := range c.tableDMLIdxMap {
		origDMLIdxMap[k] = v
	}

	err := c.externalStorage.WalkDir(ctx, opt, func(path string, size int64) error {
		var dmlkey dmlPathKey
		var schemaKey schemaPathKey
		var fileIdx uint64
		var err error
<<<<<<< HEAD
=======

		if strings.HasSuffix(path, "metadata") {
			return nil
		}
>>>>>>> 855e4e13bd (consumer(ticdc): support handling DDL events in cloud storage consumer (#8278))

		if strings.HasSuffix(path, "schema.json") {
			err = schemaKey.parseSchemaFilePath(path)
			if err != nil {
				log.Error("failed to parse schema file path", zap.Error(err))
				// skip handling this file
				return nil
<<<<<<< HEAD
			}
			// fake a dml key for schema.json file, which is useful for putting DDL
			// in front of the DML files when sorting.
			// e.g, for the partitioned table:
			//
			// test/test1/439972354120482843/schema.json					(partitionNum = -1)
			// test/test1/439972354120482843/55/2023-03-09/CDC000001.csv	(partitionNum = 55)
			// test/test1/439972354120482843/66/2023-03-09/CDC000001.csv	(partitionNum = 66)
			//
			// and for the non-partitioned table:
			// test/test2/439972354120482843/schema.json				(partitionNum = -1)
			// test/test2/439972354120482843/2023-03-09/CDC000001.csv	(partitionNum = 0)
			// test/test2/439972354120482843/2023-03-09/CDC000002.csv	(partitionNum = 0)
			//
			// the DDL event recorded in schema.json should be executed first, then the DML events
			// in csv files can be executed.
			dmlkey.schemaPathKey = schemaKey
			dmlkey.partitionNum = fakePartitionNumForSchemaFile
			dmlkey.date = ""
		} else if strings.HasSuffix(path, c.fileExtension) {
			fileIdx, err = dmlkey.parseDMLFilePath(c.replicationCfg.Sink.DateSeparator, path)
			if err != nil {
				log.Error("failed to parse dml file path", zap.Error(err))
				// skip handling this file
				return nil
			}
		} else {
			log.Debug("ignore handling file", zap.String("path", path))
			return nil
		}

		if _, ok := c.tableIdxMap[dmlkey]; !ok || fileIdx >= c.tableIdxMap[dmlkey] {
			c.tableIdxMap[dmlkey] = fileIdx
=======
			}
			// fake a dml key for schema.json file, which is useful for putting DDL
			// in front of the DML files when sorting.
			// e.g, for the partitioned table:
			//
			// test/test1/439972354120482843/schema.json					(partitionNum = -1)
			// test/test1/439972354120482843/55/2023-03-09/CDC000001.csv	(partitionNum = 55)
			// test/test1/439972354120482843/66/2023-03-09/CDC000001.csv	(partitionNum = 66)
			//
			// and for the non-partitioned table:
			// test/test2/439972354120482843/schema.json				(partitionNum = -1)
			// test/test2/439972354120482843/2023-03-09/CDC000001.csv	(partitionNum = 0)
			// test/test2/439972354120482843/2023-03-09/CDC000002.csv	(partitionNum = 0)
			//
			// the DDL event recorded in schema.json should be executed first, then the DML events
			// in csv files can be executed.
			dmlkey.schemaPathKey = schemaKey
			dmlkey.partitionNum = fakePartitionNumForSchemaFile
			dmlkey.date = ""
		} else {
			fileIdx, err = dmlkey.parseDMLFilePath(c.replicationCfg.Sink.DateSeparator, path)
			if err != nil {
				log.Error("failed to parse dml file path", zap.Error(err))
				// skip handling this file
				return nil
			}
		}

		if _, ok := c.tableDMLIdxMap[dmlkey]; !ok || fileIdx >= c.tableDMLIdxMap[dmlkey] {
			c.tableDMLIdxMap[dmlkey] = fileIdx
>>>>>>> 855e4e13bd (consumer(ticdc): support handling DDL events in cloud storage consumer (#8278))
		}

		return nil
	})
	if err != nil {
		return tableDMLMap, err
	}

	tableDMLMap = diffDMLMaps(c.tableDMLIdxMap, origDMLIdxMap)
	return tableDMLMap, err
}

// emitDMLEvents decodes RowChangedEvents from file content and emit them.
func (c *consumer) emitDMLEvents(
	ctx context.Context, tableID int64,
	tableDetail cloudstorage.TableDefinition,
	pathKey dmlPathKey,
	content []byte,
) error {
	var (
<<<<<<< HEAD
		events      []*model.RowChangedEvent
		tableDetail cloudstorage.TableDefinition
		decoder     codec.EventBatchDecoder
		err         error
=======
		decoder codec.EventBatchDecoder
		err     error
>>>>>>> 855e4e13bd (consumer(ticdc): support handling DDL events in cloud storage consumer (#8278))
	)

	tableInfo, err := tableDetail.ToTableInfo()
	if err != nil {
		return errors.Trace(err)
	}

	switch c.codecCfg.Protocol {
	case config.ProtocolCsv:
		decoder, err = csv.NewBatchDecoder(ctx, c.codecCfg, tableInfo, content)
		if err != nil {
			return errors.Trace(err)
		}
	case config.ProtocolCanalJSON:
		decoder = canal.NewBatchDecoder(content, false, c.codecCfg.Terminator)
	}

	cnt := 0
	for {
		tp, hasNext, err := decoder.HasNext()
		if err != nil {
			log.Error("failed to decode message", zap.Error(err))
			return err
		}
		if !hasNext {
			break
		}
		cnt++

		if tp == model.MessageTypeRow {
			row, err := decoder.NextRowChangedEvent()
			if err != nil {
				log.Error("failed to get next row changed event", zap.Error(err))
				return errors.Trace(err)
			}

			if _, ok := c.tableTsMap[tableID]; !ok || row.CommitTs >= c.tableTsMap[tableID] {
				c.tableTsMap[tableID] = row.CommitTs
			} else {
				log.Warn("row changed event commit ts fallback, ignore",
					zap.Uint64("commitTs", row.CommitTs),
					zap.Uint64("tableMaxCommitTs", c.tableTsMap[tableID]),
					zap.Any("row", row),
				)
				continue
			}
			row.Table.TableID = tableID
			events = append(events, row)
		}
	}
	log.Info("decode success", zap.String("schema", pathKey.schema),
		zap.String("table", pathKey.table),
		zap.Int64("version", pathKey.version),
		zap.Int("decodeRowsCnt", cnt),
		zap.Int("filteredRowsCnt", len(events)))

	err = c.sink.EmitRowChangedEvents(ctx, events...)
	return err
}

<<<<<<< HEAD
=======
func (c *consumer) waitTableFlushComplete(ctx context.Context, tableID model.TableID) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-c.errCh:
			return err
		default:
		}

		resolvedTs := c.tableTsMap[tableID]
		err := c.tableSinkMap[tableID].UpdateResolvedTs(resolvedTs)
		if err != nil {
			return errors.Trace(err)
		}
		checkpoint := c.tableSinkMap[tableID].GetCheckpointTs()
		if checkpoint.Equal(resolvedTs) {
			c.tableTsMap[tableID] = resolvedTs.AdvanceBatch()
			return nil
		}
		time.Sleep(defaultFlushWaitDuration)
	}
}

func (c *consumer) syncExecDMLEvents(
	ctx context.Context,
	tableDef cloudstorage.TableDefinition,
	key dmlPathKey,
	fileIdx uint64,
) error {
	filePath := key.generateDMLFilePath(fileIdx, c.fileExtension)
	log.Debug("read from dml file path", zap.String("path", filePath))
	content, err := c.externalStorage.ReadFile(ctx, filePath)
	if err != nil {
		return errors.Trace(err)
	}
	tableID := c.tableIDGenerator.generateFakeTableID(
		key.schema, key.table, key.partitionNum)
	err = c.emitDMLEvents(ctx, tableID, tableDef, key, content)
	if err != nil {
		return errors.Trace(err)
	}

	resolvedTs := c.tableTsMap[tableID]
	err = c.tableSinkMap[tableID].UpdateResolvedTs(resolvedTs)
	if err != nil {
		return errors.Trace(err)
	}
	err = c.waitTableFlushComplete(ctx, tableID)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (c *consumer) getTableDefFromFile(
	ctx context.Context,
	schemaKey schemaPathKey,
) (cloudstorage.TableDefinition, error) {
	var tableDef cloudstorage.TableDefinition

	schemaFilePath := schemaKey.generagteSchemaFilePath()
	schemaContent, err := c.externalStorage.ReadFile(ctx, schemaFilePath)
	if err != nil {
		return tableDef, errors.Trace(err)
	}

	err = json.Unmarshal(schemaContent, &tableDef)
	if err != nil {
		return tableDef, errors.Trace(err)
	}

	return tableDef, nil
}

func (c *consumer) handleNewFiles(
	ctx context.Context,
	dmlFileMap map[dmlPathKey]fileIndexRange,
) error {
	keys := make([]dmlPathKey, 0, len(dmlFileMap))
	for k := range dmlFileMap {
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		log.Info("no new dml files found since last round")
		return nil
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].version != keys[j].version {
			return keys[i].version < keys[j].version
		}
		if keys[i].partitionNum != keys[j].partitionNum {
			return keys[i].partitionNum < keys[j].partitionNum
		}
		if keys[i].date != keys[j].date {
			return keys[i].date < keys[j].date
		}
		if keys[i].schema != keys[j].schema {
			return keys[i].schema < keys[j].schema
		}
		return keys[i].table < keys[j].table
	})

	for _, key := range keys {
		tableDef, err := c.getTableDefFromFile(ctx, key.schemaPathKey)
		if err != nil {
			return err
		}
		// if the key is a fake dml path key which is mainly used for
		// sorting schema.json file before the dml files, then execute the ddl query.
		if key.partitionNum == fakePartitionNumForSchemaFile &&
			len(key.date) == 0 && len(tableDef.Query) > 0 {
			ddlEvent, err := tableDef.ToDDLEvent()
			if err != nil {
				return err
			}
			if err := c.ddlSink.WriteDDLEvent(ctx, ddlEvent); err != nil {
				return errors.Trace(err)
			}
			log.Info("execute ddl event successfully", zap.String("query", tableDef.Query))
			continue
		}

		fileRange := dmlFileMap[key]
		for i := fileRange.start; i <= fileRange.end; i++ {
			if err := c.syncExecDMLEvents(ctx, tableDef, key, i); err != nil {
				return err
			}
		}
	}

	return nil
}

>>>>>>> 855e4e13bd (consumer(ticdc): support handling DDL events in cloud storage consumer (#8278))
func (c *consumer) run(ctx context.Context) error {
	ticker := time.NewTicker(flushInterval)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		dmlFileMap, err := c.getNewFiles(ctx)
		if err != nil {
			return errors.Trace(err)
		}

<<<<<<< HEAD
		keys := make([]dmlPathKey, 0, len(fileMap))
		for k := range fileMap {
			keys = append(keys, k)
		}

		if len(keys) == 0 {
			log.Info("no new files found since last round")
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].schema != keys[j].schema {
				return keys[i].schema < keys[j].schema
			}
			if keys[i].table != keys[j].table {
				return keys[i].table < keys[j].table
			}
			if keys[i].version != keys[j].version {
				return keys[i].version < keys[j].version
			}
			if keys[i].partitionNum != keys[j].partitionNum {
				return keys[i].partitionNum < keys[j].partitionNum
			}
			return keys[i].date < keys[j].date
		})

		for _, k := range keys {
			fileRange := fileMap[k]
			for i := fileRange.start; i <= fileRange.end; i++ {
				filePath := k.generateDMLFilePath(i, c.fileExtension)
				log.Debug("read from dml file path", zap.String("path", filePath))
				content, err := c.externalStorage.ReadFile(ctx, filePath)
				if err != nil {
					return errors.Trace(err)
				}
				tableID := c.tableIDGenerator.generateFakeTableID(
					k.schema, k.table, k.partitionNum)
				err = c.emitDMLEvents(ctx, tableID, k, content)
				if err != nil {
					return errors.Trace(err)
				}

				resolvedTs := model.NewResolvedTs(c.tableTsMap[tableID])
				_, err = c.sink.FlushRowChangedEvents(ctx, tableID, resolvedTs)
				if err != nil {
					return errors.Trace(err)
				}
			}
=======
		err = c.handleNewFiles(ctx, dmlFileMap)
		if err != nil {
			return errors.Trace(err)
>>>>>>> 855e4e13bd (consumer(ticdc): support handling DDL events in cloud storage consumer (#8278))
		}
	}
}

// copied from kafka-consumer
type fakeTableIDGenerator struct {
	tableIDs       map[string]int64
	currentTableID int64
	mu             sync.Mutex
}

func (g *fakeTableIDGenerator) generateFakeTableID(schema, table string, partition int64) int64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	key := quotes.QuoteSchema(schema, table)
	if partition != 0 {
		key = fmt.Sprintf("%s.`%d`", key, partition)
	}
	if tableID, ok := g.tableIDs[key]; ok {
		return tableID
	}
	g.currentTableID++
	g.tableIDs[key] = g.currentTableID
	return g.currentTableID
}

func main() {
<<<<<<< HEAD
=======
	var consumer *consumer
	var err error

	if enableProfiling {
		go func() {
			server := &http.Server{
				Addr:              ":6060",
				ReadHeaderTimeout: 5 * time.Second,
			}

			if err := server.ListenAndServe(); err != nil {
				log.Fatal("http pprof", zap.Error(err))
			}
		}()
	}

>>>>>>> 855e4e13bd (consumer(ticdc): support handling DDL events in cloud storage consumer (#8278))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	consumer, err := newConsumer(ctx)
	if err != nil {
		log.Error("failed to create storage consumer", zap.Error(err))
		os.Exit(1)
	}

	if err := consumer.run(ctx); err != nil && errors.Cause(err) != context.Canceled {
		log.Error("error occurred while running consumer", zap.Error(err))
		os.Exit(1)
	}
	os.Exit(0)
}
