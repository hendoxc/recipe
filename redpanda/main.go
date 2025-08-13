package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
	"recipe/pb"
	"recipe/serde"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sr"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	srSubject  = "my.subject-reversed"
	topic      = "my.data"
	schemaPath = "message_reversed.proto"
)

func main() {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	kafkaClient, err := kgo.NewClient(kgo.SeedBrokers("localhost:19092"))
	if err != nil {
		log.Fatalf("Failed to create kafka client: %v", err)
	}
	defer kafkaClient.Close()

	srClient, err := sr.NewClient(sr.URLs("http://localhost:18081"))
	if err != nil {
		log.Fatalf("Failed to create schema registry client: %v", err)
	}

	initTopics(ctx, kafkaClient, topic)
	schemaID := initSchema(ctx, srClient, srSubject, schemaPath)

	msgTwo := &pb.MessageTwo{
		Value: "Hello, World!",
	}

	//
	// Lets produce MessageTwo which is index 0 in the proto schema
	// you can correctly view this in the console and also consume it with the connect consumer
	///

	protoBytes, err := proto.Marshal(msgTwo)
	if err != nil {
		log.Fatalf("Failed to marshal message: %v", err)
	}

	produceMessage(ctx, kafkaClient, protoBytes, schemaID, topic, msgTwo.ProtoReflect().Descriptor())

	//
	// Now lets produce MessageOne which is index 1 in the proto schema
	// this is where the issues happen
	// the console isn't able to decode this message
	// and the connect consumer even panics!
	// you  can comment this out to see the console and connect consumer work with Just MessageTwo

	msgOne := &pb.MessageOne{
		Value: "Hello, World! One",
	}

	protoBytes, err = proto.Marshal(msgOne)
	if err != nil {
		log.Fatalf("Failed to marshal message: %v", err)
	}

	produceMessage(ctx, kafkaClient, protoBytes, schemaID, topic, msgOne.ProtoReflect().Descriptor())

}

func produceMessage(ctx context.Context, kafkaClient *kgo.Client, protoBytes []byte, schemaID uint32, topic string, descriptor protoreflect.MessageDescriptor) {

	// this is where we figure out the index of the message
	msgIndexBytes := serde.ToMessageIndexBytes(descriptor)

	// Prepare the Schema Registry format
	schemaIDBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(schemaIDBytes, schemaID)

	// Concatenate: Magic Byte (1 byte) + Schema ID (4 bytes) + index(can vary) + Protobuf message
	encodedMessage := []byte{0} // 0 is the magic byte for Confluent serialization
	encodedMessage = append(encodedMessage, schemaIDBytes...)
	encodedMessage = append(encodedMessage, msgIndexBytes...)
	encodedMessage = append(encodedMessage, protoBytes...)

	r, err := kafkaClient.ProduceSync(ctx, &kgo.Record{
		Topic: topic,
		Value: encodedMessage,
	}).First()
	if err != nil {
		log.Fatalf("Failed to produce message: %v", err)
	}

	fmt.Printf("Message produced to partition %d\n", r.Partition)
	fmt.Printf("Message produced to offset %d\n", r.Offset)
	fmt.Printf("Message produced to timestamp %v\n", r.Timestamp)
}

func initTopics(ctx context.Context, kafkaClient *kgo.Client, topic string) {
	kadmin := kadm.NewClient(kafkaClient)
	_, err := kadmin.CreateTopic(ctx, 1, 1, nil, topic)
	if err != nil {
		if errors.Is(err, kerr.TopicAlreadyExists) {
			fmt.Printf("Topic %s already exists\n", topic)
			return
		}
		log.Fatalf("Failed to create topic: %v", err)
	}
	fmt.Printf("Topic %s created\n", topic)
}

func initSchema(ctx context.Context, srClient *sr.Client, schemaSubject string, schemaFilePath string) uint32 {
	ss, err := srClient.CreateSchema(ctx, schemaSubject, sr.Schema{
		Schema: readSchemaContents(schemaFilePath),
		Type:   sr.TypeProtobuf,
	})
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}
	fmt.Printf("Schema %s created with ID %d\n", schemaSubject, ss.ID)
	return uint32(ss.ID)
}

func readSchemaContents(schemaFilePath string) string {
	schemaText, err := os.ReadFile(schemaFilePath)
	if err != nil {
		log.Fatalf("Failed to read schema file: %v", err)
	}
	return string(schemaText)
}
