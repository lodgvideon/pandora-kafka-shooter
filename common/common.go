package common

type KafkaConfig struct {
	KafkaBrokers           string `validate:"required" config:"kafka_brokers"` // localhost:9094,localhost:9094, will fail, without target defined
	Compression            int    `config:"compression"`                       //None= 0, Gzip = 1,Snappy = 2,Lz4= 3,Zstd = 4
	BatchSize              int    `config:"batch_size"`                        //default=100
	MaxAttempts            int    `config:"max_attempts"`                      // default=10
	BatchBytes             int64  `config:"batch_bytes"`                       //default 1048576
	BatchTimeout           string `config:"batch_timeout"`                     //1s
	Balancer               string `config:"balancer"`                          // default: sarama
	WriteTimeout           string `config:"write_timeout"`                     //default: 10s
	Async                  bool   `config:"async"`                             //default: false
	AllowAutoTopicCreation bool   `config:"allow_topic_autocreation"`          //default:false
	RequiredAcks           int    `config:"required_acks" validate:"required"` //RequireNone = 0	RequireOne  = 1 RequireAll = -1
	Consistent             bool   `config:"consistent"`
}
