package common

import (
	"context"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator"
	"github.com/yandex/pandora/core/coreutil"
	"log"
	"strconv"
	"time"
)

type ClickhouseAggregator struct {
	aggregator.Reporter
	ClickhouseConfig ClickhouseConfig
	Conn             driver.Conn
	currentSize      uint32
	batch            driver.Batch
}

func NewClickhouseAggregator(clickhouseConfig ClickhouseConfig) *ClickhouseAggregator {

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{clickhouseConfig.Address},
		Auth: clickhouse.Auth{
			Database: clickhouseConfig.Database,
			Username: clickhouseConfig.Username,
			Password: clickhouseConfig.Password,
		},
		MaxOpenConns: clickhouseConfig.MaxOpenConns,
		//TLS: &tls.Config{
		//	InsecureSkipVerify: true,
		//},
		Settings: clickhouse.Settings{
			"max_execution_time": 120,
		},
		DialTimeout: 5 * time.Second,
		Compression: &clickhouse.Compression{
			clickhouse.CompressionLZ4,
		},
		Debug: true,
	})
	batch, err := conn.PrepareBatch(context.Background(), "INSERT INTO pandora_results.pandora_results")
	err = conn.Ping(context.Background())

	if err != nil {
		log.Panic("Unable to create Clickhouse Connection")

	}

	return &ClickhouseAggregator{ClickhouseConfig: clickhouseConfig, Conn: conn, batch: batch, Reporter: *aggregator.NewReporter(clickhouseConfig.ReporterConfig)}
}

func (c *ClickhouseAggregator) Run(ctx context.Context, deps core.AggregatorDeps) (err error) {
HandleLoop:
	for {
		select {
		case sample := <-c.Incomming:
			err := c.handleSample(sample)
			if err != nil {
				log.Panic(err.Error())
				return err
			}
		case <-ctx.Done():
			break HandleLoop // Still need to handle all queued samples.
		}
	}
	return
}

func (c *ClickhouseAggregator) handleSample(sample core.Sample) (err error) {
	ctx := context.Background()

	if c.batch == nil {
		batch, err := c.Conn.PrepareBatch(ctx, "INSERT INTO pandora_results.pandora_results")
		if err != nil {
			log.Panic("Error during creating batch", err)
		}
		c.batch = batch
	}
	var errorCount = 0
	s := sample.(*Sample)
	if s.error != nil {
		errorCount = 1
	}

	tags := s.Tags()
	err = c.batch.Append(
		s.timeStamp,
		uint64(s.timeStamp.UnixMilli()),
		c.ClickhouseConfig.ProfileName,
		c.ClickhouseConfig.RunId,
		c.ClickhouseConfig.Hostname,
		tags,
		uint64(1),
		uint64(errorCount),
		float64(s.GetDuration()),
		strconv.Itoa(s.ProtoCode()),
		s.Err().Error(),
	)
	if err != nil {
		log.Panic("Error during appending to Batch")
	}
	c.currentSize++

	if c.currentSize >= c.ClickhouseConfig.BatchSize {
		err := c.batch.Send()
		if err != nil {
			log.Panic("Error during sending batch")
		}
		c.batch = nil
		c.currentSize = 0
	}

	coreutil.ReturnSampleIfBorrowed(sample)
	return
}

type ClickhouseConfig struct {
	Address         string                    `config:"address" validate:"required"`
	Database        string                    `config:"database" validate:"required"`
	Username        string                    `config:"username" validate:"required"`
	Password        string                    `config:"password" validate:"required"`
	BatchSize       uint32                    `config:"batch_size" validate:"required"`
	MaxOpenConns    int                       `config:"max_connections"`
	MaxIdleConns    uint32                    `config:"max_iddle"`
	ConnMaxLifetime time.Duration             `config:"conn_lifetime" validate:"required"`
	ReporterConfig  aggregator.ReporterConfig `config:",squash"`
	ProfileName     string                    `config:"profile" validate:"required"`
	RunId           string                    `config:"run_id" validate:"required"`
	Hostname        string                    `config:"hostname" validate:"required"`
}
