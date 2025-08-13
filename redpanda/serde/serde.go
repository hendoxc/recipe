package serde

import (
	"encoding/binary"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// toMessageIndexesRecursive builds the index path from the descriptor up to the file root.
// The path is built in parent-to-child order, which is what the wire format requires.
func toMessageIndexesRecursive(descriptor protoreflect.Descriptor) []int {
	index := descriptor.Index()
	parent := descriptor.Parent()

	// Base case: parent is the file descriptor
	if _, ok := parent.(protoreflect.FileDescriptor); ok {
		return []int{index}
	}

	// Recursive case: parent must be another message descriptor
	parentMsgDesc, ok := parent.(protoreflect.MessageDescriptor) // Explicitly check for MessageDescriptor
	if !ok {
		// This indicates an unexpected structure or a broken parent link.
		return []int{index}
	}
	// Get the parent's path first, then append the current index.
	parentIndexes := toMessageIndexesRecursive(parentMsgDesc) // Recursive call with the MessageDescriptor
	return append(parentIndexes, index)
}

// toMessageIndexBytes calculates the Confluent Wire Format message index bytes.
func ToMessageIndexBytes(descriptor protoreflect.Descriptor) []byte {

	// Optimization: If it's the first message defined directly in the file (index 0, parent is FileDescriptor),
	// the wire format uses a single byte {0}.
	if descriptor.Index() == 0 {
		if _, ok := descriptor.Parent().(protoreflect.FileDescriptor); ok {
			return []byte{0}
		}
	}

	// Get the index path (e.g., [0, 1] for Outer.Inner) using the recursive helper.
	msgIndexes := toMessageIndexesRecursive(descriptor)

	// Allocate a buffer large enough for the worst-case varint encoding.
	// Use MaxVarintLen64 as it's the max length for both signed and unsigned.
	buf := make([]byte, (1+len(msgIndexes))*binary.MaxVarintLen64)

	// Encode the number of indexes (length of the path) as an unsigned varint.
	// Use PutUvarint for unsigned integers.
	offset := binary.PutUvarint(buf, uint64(len(msgIndexes)))

	// Encode each index in the path as an unsigned varint.
	for _, element := range msgIndexes {
		// Use PutUvarint for unsigned integers.
		offset += binary.PutUvarint(buf[offset:], uint64(element))
	}

	// Return only the portion of the buffer that was actually used.
	return buf[:offset]
}
