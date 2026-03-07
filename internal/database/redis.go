package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisDriver implements the Driver interface for Redis.
//
// Since Redis is a key-value store, the MCP "sql" parameter is reinterpreted
// as a JSON command descriptor:
//
//	Query:   {"cmd":"get","key":"mykey"}
//	         {"cmd":"hgetall","key":"myhash"}
//	         {"cmd":"lrange","key":"mylist","start":0,"stop":-1}
//	         {"cmd":"smembers","key":"myset"}
//	         {"cmd":"zrange","key":"myzset","start":0,"stop":-1}
//	         {"cmd":"keys","pattern":"prefix:*"}
//	         {"cmd":"scan","pattern":"prefix:*","count":100}
//
//	Execute: {"cmd":"set","key":"mykey","value":"myval"}
//	         {"cmd":"set","key":"mykey","value":"myval","ttl":"1h"}
//	         {"cmd":"del","key":"mykey"}
//	         {"cmd":"hset","key":"myhash","field":"f","value":"v"}
//	         {"cmd":"hdel","key":"myhash","field":"f"}
//	         {"cmd":"expire","key":"mykey","ttl":"1h"}
//
//	Schema:  GetTableNames lists keys matching a pattern (default "*").
//	         GetTableSchema returns type + TTL + value for a key.
type RedisDriver struct{}

// Name returns the driver name
func (d *RedisDriver) Name() string {
	return "redis"
}

// Open connects to Redis. DSN is a Redis URL: redis://[:password@]host:port[/db]
func (d *RedisDriver) Open(dsn string) (Handle, error) {
	opts, err := redis.ParseURL(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid Redis URL: %w", err)
	}

	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return client, nil
}

// Query executes a read command against Redis.
func (d *RedisDriver) Query(h Handle, query string) ([]string, [][]interface{}, error) {
	client := h.(*redis.Client)
	ctx := context.Background()

	var cmd struct {
		Cmd     string `json:"cmd"`
		Key     string `json:"key"`
		Pattern string `json:"pattern"`
		Start   int64  `json:"start"`
		Stop    int64  `json:"stop"`
		Count   int64  `json:"count"`
	}
	if err := json.Unmarshal([]byte(query), &cmd); err != nil {
		// Treat plain string as a key for GET
		cmd.Cmd = "get"
		cmd.Key = query
	}
	if cmd.Stop == 0 {
		cmd.Stop = -1
	}
	if cmd.Count == 0 {
		cmd.Count = 100
	}

	switch strings.ToLower(cmd.Cmd) {
	case "get":
		val, err := client.Get(ctx, cmd.Key).Result()
		if err == redis.Nil {
			return []string{"key", "value"}, [][]interface{}{{cmd.Key, nil}}, nil
		}
		if err != nil {
			return nil, nil, err
		}
		return []string{"key", "value"}, [][]interface{}{{cmd.Key, val}}, nil

	case "hgetall":
		m, err := client.HGetAll(ctx, cmd.Key).Result()
		if err != nil {
			return nil, nil, err
		}
		columns := []string{"field", "value"}
		rows := make([][]interface{}, 0, len(m))
		for f, v := range m {
			rows = append(rows, []interface{}{f, v})
		}
		return columns, rows, nil

	case "lrange":
		vals, err := client.LRange(ctx, cmd.Key, cmd.Start, cmd.Stop).Result()
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]interface{}, len(vals))
		for i, v := range vals {
			rows[i] = []interface{}{int64(i), v}
		}
		return []string{"index", "value"}, rows, nil

	case "smembers":
		vals, err := client.SMembers(ctx, cmd.Key).Result()
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]interface{}, len(vals))
		for i, v := range vals {
			rows[i] = []interface{}{v}
		}
		return []string{"member"}, rows, nil

	case "zrange":
		vals, err := client.ZRangeWithScores(ctx, cmd.Key, cmd.Start, cmd.Stop).Result()
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]interface{}, len(vals))
		for i, z := range vals {
			rows[i] = []interface{}{z.Member, z.Score}
		}
		return []string{"member", "score"}, rows, nil

	case "keys":
		pattern := cmd.Pattern
		if pattern == "" {
			pattern = cmd.Key
		}
		if pattern == "" {
			pattern = "*"
		}
		keys, err := client.Keys(ctx, pattern).Result()
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]interface{}, len(keys))
		for i, k := range keys {
			rows[i] = []interface{}{k}
		}
		return []string{"key"}, rows, nil

	case "scan":
		pattern := cmd.Pattern
		if pattern == "" {
			pattern = "*"
		}
		var keys []string
		var cursor uint64
		for {
			k, c, err := client.Scan(ctx, cursor, pattern, cmd.Count).Result()
			if err != nil {
				return nil, nil, err
			}
			keys = append(keys, k...)
			cursor = c
			if cursor == 0 {
				break
			}
		}
		rows := make([][]interface{}, len(keys))
		for i, k := range keys {
			rows[i] = []interface{}{k}
		}
		return []string{"key"}, rows, nil

	default:
		return nil, nil, fmt.Errorf("unknown query cmd %q: use get, hgetall, lrange, smembers, zrange, keys, scan", cmd.Cmd)
	}
}

// Execute runs a write command against Redis.
func (d *RedisDriver) Execute(h Handle, query string) (int64, error) {
	client := h.(*redis.Client)
	ctx := context.Background()

	var cmd struct {
		Cmd   string `json:"cmd"`
		Key   string `json:"key"`
		Field string `json:"field"`
		Value string `json:"value"`
		TTL   string `json:"ttl"`
	}
	if err := json.Unmarshal([]byte(query), &cmd); err != nil {
		return 0, fmt.Errorf("execute requires JSON: {\"cmd\":\"set|del|hset|hdel|expire\",...}: %w", err)
	}

	var ttl time.Duration
	if cmd.TTL != "" {
		var err error
		ttl, err = time.ParseDuration(cmd.TTL)
		if err != nil {
			return 0, fmt.Errorf("invalid ttl %q: %w", cmd.TTL, err)
		}
	}

	switch strings.ToLower(cmd.Cmd) {
	case "set":
		if err := client.Set(ctx, cmd.Key, cmd.Value, ttl).Err(); err != nil {
			return 0, err
		}
		return 1, nil

	case "del":
		n, err := client.Del(ctx, cmd.Key).Result()
		if err != nil {
			return 0, err
		}
		return n, nil

	case "hset":
		n, err := client.HSet(ctx, cmd.Key, cmd.Field, cmd.Value).Result()
		if err != nil {
			return 0, err
		}
		return n, nil

	case "hdel":
		n, err := client.HDel(ctx, cmd.Key, cmd.Field).Result()
		if err != nil {
			return 0, err
		}
		return n, nil

	case "expire":
		if ttl == 0 {
			return 0, fmt.Errorf("expire requires a ttl field, e.g. \"1h\"")
		}
		ok, err := client.Expire(ctx, cmd.Key, ttl).Result()
		if err != nil {
			return 0, err
		}
		if ok {
			return 1, nil
		}
		return 0, nil

	default:
		return 0, fmt.Errorf("unknown execute cmd %q: use set, del, hset, hdel, expire", cmd.Cmd)
	}
}

// GetTableNames returns keys matching "*" via SCAN (safe for production).
func (d *RedisDriver) GetTableNames(h Handle) ([]string, error) {
	client := h.(*redis.Client)
	ctx := context.Background()

	var keys []string
	var cursor uint64
	for {
		k, c, err := client.Scan(ctx, cursor, "*", 100).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, k...)
		cursor = c
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

// GetTableSchema returns type, TTL, and value summary for a key.
func (d *RedisDriver) GetTableSchema(h Handle, key string) ([]map[string]interface{}, error) {
	client := h.(*redis.Client)
	ctx := context.Background()

	keyType, err := client.Type(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	ttl, err := client.TTL(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	info := map[string]interface{}{
		"key":  key,
		"type": keyType,
		"ttl":  ttl.String(),
	}

	switch keyType {
	case "string":
		val, err := client.Get(ctx, key).Result()
		if err == nil {
			info["value"] = val
		}
	case "hash":
		info["fields"], _ = client.HLen(ctx, key).Result()
	case "list":
		info["length"], _ = client.LLen(ctx, key).Result()
	case "set":
		info["members"], _ = client.SCard(ctx, key).Result()
	case "zset":
		info["members"], _ = client.ZCard(ctx, key).Result()
	}

	return []map[string]interface{}{info}, nil
}

// Close closes the Redis client.
func (d *RedisDriver) Close(h Handle) error {
	return h.(*redis.Client).Close()
}
