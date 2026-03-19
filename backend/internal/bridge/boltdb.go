package bridge

import (
	"fmt"
	"strings"

	bolt "go.etcd.io/bbolt"
)

// BoltDBBridge implements a bridge to a local BoltDB file
type BoltDBBridge struct {
	filePath string
	db       *bolt.DB
}

// NewBoltDBBridge creates a new BoltDB bridge
func NewBoltDBBridge(filePath string) (*BoltDBBridge, error) {
	db, err := bolt.Open(filePath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open BoltDB file: %w", err)
	}

	return &BoltDBBridge{
		filePath: filePath,
		db:       db,
	}, nil
}

// Close closes the bridge connection
func (b *BoltDBBridge) Close() error {
	if b.db != nil {
		return b.db.Close()
	}
	return nil
}

// GetBuckets returns all top-level bucket names
func (b *BoltDBBridge) GetBuckets() ([]string, error) {
	var buckets []string
	err := b.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
			buckets = append(buckets, string(name))
			return nil
		})
	})
	return buckets, err
}

// Get retrieves a value by bucket and key
func (b *BoltDBBridge) Get(bucket, key string) (string, error) {
	var value string
	err := b.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(bucket))
		if bkt == nil {
			return fmt.Errorf("bucket %q not found", bucket)
		}
		v := bkt.Get([]byte(key))
		if v == nil {
			return fmt.Errorf("key %q not found in bucket %q", key, bucket)
		}
		value = string(v)
		return nil
	})
	return value, err
}

// Put sets a key-value pair in a bucket (creates bucket if needed)
func (b *BoltDBBridge) Put(bucket, key, value string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return fmt.Errorf("failed to create bucket %q: %w", bucket, err)
		}
		return bkt.Put([]byte(key), []byte(value))
	})
}

// Delete removes a key from a bucket
func (b *BoltDBBridge) Delete(bucket, key string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(bucket))
		if bkt == nil {
			return fmt.Errorf("bucket %q not found", bucket)
		}
		return bkt.Delete([]byte(key))
	})
}

// Scan returns all key-value pairs in a bucket, optionally filtered by prefix
func (b *BoltDBBridge) Scan(bucket, prefix string) (map[string]string, error) {
	result := make(map[string]string)
	err := b.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(bucket))
		if bkt == nil {
			return fmt.Errorf("bucket %q not found", bucket)
		}
		return bkt.ForEach(func(k, v []byte) error {
			key := string(k)
			if prefix == "" || strings.HasPrefix(key, prefix) {
				result[key] = string(v)
			}
			return nil
		})
	})
	return result, err
}
