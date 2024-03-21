package main

import (
	"context"
	"fmt"
	"github.com/hamba/avro"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sr"
	"golang.org/x/crypto/sha3"
	"log"
	"math/rand"
	"os"
	"time"
)

const (
	BootstrapServers  = "localhost:9092"
	SchemaRegistryUrl = "http://localhost:8081"
	topic             = "ethereum.mainnet.blocks"
	BatchSize         = 10000
)

type BlockHeader struct {
	Number     int64  `json:"number" avro:"number"`
	Hash       string `json:"hash" avro:"hash"`
	ParentHash string `json:"parent_hash" avro:"parent_hash"`
	GasUsed    int64  `json:"gas_used" avro:"gas_used"`
	Timestamp  int64  `json:"timestamp" avro:"timestamp"`
}

func main() {

	kafkaClient, err := kgo.NewClient(kgo.SeedBrokers(BootstrapServers))
	if err != nil {
		log.Panicf("unable to create kafka client %s", err)
	}

	// register the schema & get a serializer/deserializer (serde)
	avroSerde, err := registerSchema[BlockHeader]("block_header.avsc", "ethereum.mainnet.blocks-value")
	if err != nil {
		log.Panic(err)
	}

	prevBlock := BlockHeader{Number: 10000, Timestamp: time.Now().Unix()}
	var batch []*kgo.Record
	for i := 1; i <= 1000000; i++ {

		// generate a new block, given the previous block
		block := generateBlock(prevBlock.Number, prevBlock.Time().Add(time.Duration(30)*time.Second))

		if (i%BatchSize == 0 && len(batch) > 0) || i == 1000000 {

			response := kafkaClient.ProduceSync(context.Background(), batch...)
			if response.FirstErr() != nil {
				log.Panic(response.FirstErr())
			}

			fmt.Printf("Produced %d blocks\n", i)
			// prob better way to do this
			batch = []*kgo.Record{}
		}

		// Note we are using the serde to encode the block to avro
		batch = append(batch, &kgo.Record{
			Topic: topic,
			Key:   []byte(fmt.Sprintf("%d", block.Number)),
			Value: avroSerde.MustEncode(block),
		})

	}
}

func registerSchema[T any](schemaFilePath string, subject string) (*sr.Serde, error) {

	schemaRegistryClient, err := sr.NewClient(sr.URLs(SchemaRegistryUrl))
	if err != nil {
		return nil, fmt.Errorf("unable to create schema registry client %s", err)
	}

	// register the schema and Create Serializer/Deserializer
	schemaTextBytes, err := os.ReadFile(schemaFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read schema file %w", err)
	}

	subjectSchema, err := schemaRegistryClient.CreateSchema(context.Background(), subject, sr.Schema{
		Schema: string(schemaTextBytes),
		Type:   sr.TypeAvro,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create schema %w", err)
	}

	avroSchema, err := avro.Parse(string(schemaTextBytes))
	if err != nil {
		return nil, fmt.Errorf("unable to parse schema %w", err)
	}

	var schemaType T
	var serde sr.Serde
	serde.Register(
		subjectSchema.ID,
		schemaType,
		sr.EncodeFn(func(a any) ([]byte, error) {
			return avro.Marshal(avroSchema, a)
		}),
		sr.DecodeFn(func(bytes []byte, a any) error {
			return avro.Unmarshal(avroSchema, bytes, a)
		}),
	)

	return &serde, nil
}

func generateBlock(blockNumber int64, timestamp time.Time) BlockHeader {

	blockHash := sha3.New256()
	blockHash.Write([]byte(fmt.Sprintf("%d", blockNumber)))

	parentHash := sha3.New256()
	parentHash.Write([]byte(fmt.Sprintf("%d", blockNumber-1)))

	return BlockHeader{
		Number:     blockNumber + 1,
		Hash:       fmt.Sprintf("%x", blockHash.Sum(nil)),
		ParentHash: fmt.Sprintf("%x", parentHash.Sum(nil)),
		GasUsed:    rand.Int63n(10000000),
		Timestamp:  timestamp.Unix(),
	}
}

func (b *BlockHeader) String() string {
	return fmt.Sprintf("Block %d: %s", b.Number, b.Hash)
}

func (b *BlockHeader) Time() time.Time {
	return time.Unix(b.Timestamp, 0)
}
