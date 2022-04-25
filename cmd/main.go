package main

// import some pandora stuff
// and stuff you need for your scenario
// and protobuf contracts for your grpc service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/protocol"
	"github.com/spf13/afero"
	"github.com/yandex/pandora/cli"
	"github.com/yandex/pandora/components/phttp/import"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/core/import"
	"github.com/yandex/pandora/core/register"
	"os"
	"strings"
	"time"
)

type Ammo struct {
	Topic   string          `json:"topic,omitempty"`
	Key     string          `json:"key,omitempty"`
	Tag     string          `json:"tag,omitempty"`
	Message json.RawMessage `json:"message,omitempty"`
}

type GunConfig struct {
	KafkaBrokers           string         `validate:"required" config:"kafka_brokers"` // localhost:9094,localhost:9094, will fail, without target defined
	Compression            int            `config:"compression"`                       //None= 0, Gzip = 1,Snappy = 2,Lz4= 3,Zstd = 4
	BatchSize              int            `config:"batch_size"`                        //default=100
	MaxAttempts            int            `config:"max_attempts"`                      // default=10
	BatchBytes             int64          `config:"batch_bytes"`                       //default 1048576
	BatchTimeout           string         `config:"batch_timeout"`                     //1s
	Balancer               string         `config:"balancer"`                          // default: sarama
	WriteTimeout           string         `config:"write_timeout"`                     //default: 10s
	Async                  bool           `config:"async"`                             //default: false
	AllowAutoTopicCreation bool           `config:"allow_topic_autocreation"`          //default:false
	RequiredAcks           int            `config:"required_acks" validate:"required"` //RequireNone = 0	RequireOne  = 1 RequireAll = -1
	Headers                []HeaderConfig `config:"headers"`                           // Additional headers for each message
}
type HeaderConfig struct {
	Key   string `validate:"required" config:"key"`
	Value string `validate:"required" config:"value"`
}

type Gun struct {
	// Configured on construction.
	client *kafka.Writer
	conf   GunConfig
	// Configured on Bind, before shooting
	aggr core.Aggregator // May be your custom Aggregator.
	core.GunDeps
	ShootId string
}

func NewGun(conf GunConfig) *Gun {
	return &Gun{conf: conf}
}

func (g *Gun) Bind(aggr core.Aggregator, deps core.GunDeps) error {
	g.ShootId = os.Getenv("SHOOT_ID")
	// create gRPC stub at gun initialization
	brokers := g.conf.KafkaBrokers
	brokersSlice := strings.Split(brokers, ",")
	batchTimeOut, err := time.ParseDuration(g.conf.BatchTimeout)
	if err != nil {
		g.Log.Info("Duration was not set, accepted types are: 1s,1h,100ms")
		batchTimeOut, _ = time.ParseDuration("1s")
	}

	writeTimeout, err := time.ParseDuration(g.conf.WriteTimeout)
	if err != nil {
		g.Log.Info("Duration was not set, accepted types are: 1s,1h,100ms")
		writeTimeout, _ = time.ParseDuration("10s")
	}
	var balancer kafka.Balancer
	switch g.conf.Balancer {
	case "roundrobin":
		balancer = &kafka.RoundRobin{}
	case "crc32":
		balancer = &kafka.CRC32Balancer{}
	case "sarama":
		balancer = &kafka.Hash{}
	case "murmur2":
		balancer = &kafka.Murmur2Balancer{}
	case "leastbytes":
		balancer = &kafka.LeastBytes{}
	}

	g.client = &kafka.Writer{
		Addr: kafka.TCP(brokersSlice...),
		// NOTE: When Topic is not defined here, each Message must define it instead.
		Balancer:               balancer,
		MaxAttempts:            g.conf.MaxAttempts,
		BatchSize:              g.conf.BatchSize,
		BatchBytes:             g.conf.BatchBytes,
		BatchTimeout:           batchTimeOut,
		WriteTimeout:           writeTimeout,
		RequiredAcks:           kafka.RequiredAcks(g.conf.RequiredAcks),
		Async:                  g.conf.Async,
		Compression:            kafka.Compression(g.conf.Compression),
		Logger:                 kafka.LoggerFunc(g.logf),
		ErrorLogger:            kafka.LoggerFunc(g.logError),
		AllowAutoTopicCreation: g.conf.AllowAutoTopicCreation,
	}
	g.aggr = aggr
	g.GunDeps = deps
	return nil
}
func (g *Gun) logf(msg string, params ...interface{}) {
	sprintf := fmt.Sprintf(msg, params...)
	g.Log.Debug(sprintf)
}

func (g *Gun) logError(msg string, params ...interface{}) {
	g.Log.Error(fmt.Sprintf(msg, params...))
}

func (g *Gun) Shoot(ammo core.Ammo) {
	customAmmo := ammo.(*Ammo)
	g.shoot(customAmmo)
}

func (g *Gun) shoot(ammo *Ammo) {
	sample := netsample.Acquire(ammo.Topic + "_" + ammo.Tag)
	defer func() {
		g.aggr.Report(sample)
	}()

	var headers []kafka.Header
	headers = append(headers, protocol.Header{Key: "SHOOT_ID", Value: []byte(g.ShootId)})
	for _, header := range g.conf.Headers {
		headers = append(headers, protocol.Header{Key: header.Key, Value: []byte(header.Value)})
	}

	err := g.client.WriteMessages(context.Background(),
		kafka.Message{
			Topic: ammo.Topic,
			Key:   []byte(ammo.Key), Value: []byte(ammo.Message),
			Headers: headers})
	if err != nil {
		sample.SetProtoCode(500)
		return
	}
	sample.SetProtoCode(200)

}

func main() {
	//debug.SetGCPercent(-1)
	// Standard imports.
	fs := afero.NewOsFs()
	coreimport.Import(fs)
	// May not be imported, if you don't need http guns and etc.
	phttp.Import(fs)

	// Custom imports. Integrate your custom types into configuration system.
	coreimport.RegisterCustomJSONProvider("kafka_provider", func() core.Ammo { return &Ammo{} })

	register.Gun("pandora_kafka_shooter", NewGun, func() GunConfig {
		return GunConfig{
			KafkaBrokers:           "localhost:9094",
			Compression:            0,
			BatchSize:              100,
			MaxAttempts:            10,
			BatchBytes:             1048576,
			BatchTimeout:           "1s",
			WriteTimeout:           "10s",
			Balancer:               "sarama",
			RequiredAcks:           -1, //requred all acks
			Async:                  false,
			AllowAutoTopicCreation: false,
			Headers:                []HeaderConfig{},
		}
	})

	cli.Run()
}
