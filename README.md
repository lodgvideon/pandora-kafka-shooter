# pandora-kafka-shooter
Kafka json gun for load testing Kafka-services
Written using kafka-go library.
Currents support only:
* json message format

idempotency - is not awaitable, because kafka-go does not support such behavior 
avro - not supported

protobuf - not supported

custom load balancing mechanisms - not supported



****
## Config example:

```pools:
  - gun:
      type: pandora_kafka_shooter

      kafka_brokers: 'localhost:9094'
      # can be: broker_1:9094,broker_2:9094,broker_3:9094
      
      required_acks: -1
      #RequireNone = 0	RequireOne  = 1 RequireAll = -1 default: -1
      

      compression: 0
      #None= 0, Gzip = 1,Snappy = 2,Lz4= 3,Zstd = 4  default: 0
      
      batch_size: 100
      #default: 100
      

      balancer: sarama
      # Kafka message balancing algorinms: roundrobin, sarama, crc32, murmur2, leastbytes default:sarama
      
      max_attempts: 10
      # Maximum attempts to write before failed default: 10

      write_timeout: 10s
      # Timeout for write batch to Kafka - default: 10s
      
      async: False
      #Enable async semantic - not recomend - you wil don't know - is there problem in background? default: False
      

      allow_topic_autocreation: False
      #allow_topic_autocreation - allow producer to create topic default: False
      
      headers:
        - {key: "TEMP_HEADER", value: "TEMP_VALUE"}
        - {key: "TEMP_HEADER_2", Value: "TEMP_VALUE_2"}
    ammo:
      type: kafka_provider
      source:
        path: './examples/ammo.txt'
        type: file
    result:
      type: phout
      destination: ./http_phout.log
    rps-per-instance: true
    rps:
      type: line
      from: 100
      to: 100
      duration: 30s
    startup:
      type: once
      times: 100
```
## Ammo example:

```
{"topic":"kafka_shoot", "key":"1234", "tag":"SPARTA_1", "headers": {"ammo_header":"ammo_value", "ammo_header2":"ammo_value2"}, "message": {"a":[{"v":1,"d":"s"},{"v":1,"d":"fff"}]}}
{"topic":"kafka_shoot_2", "key":"333", "tag":"SPARTA_2", "headers": {"ammo_header":"ammo_value", "ammo_header3":"ammo_value4"}, "message": {"d":"ss"}}
```

## Message balancers:

* roundrobin - standard roundrobin mechanisma
* CRC32Balancer - this partitioner is equivalent to the "consistent_random" setting in librdkafka.  When Consistent is true, this partitioner is equivalent to the "consistent" setting.  The latter will hash empty or nil keys into the same partition.
* sarama - default kafka go-sarama framework balancing algorithm
* murmur2 - murmur2_random" is functionally equivalent to the default Java partitioner.  That's because the
Java partitioner will use a round robin balancer instead of random on nil
keys.  We choose librdkafka's implementation because it arguably has a larger
install base.
* leastbytes - is a Balancer implementation that routes messages to the partition
that has received the least amount of data.