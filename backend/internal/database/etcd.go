package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdDriver implements the Driver interface for etcd v3.
//
// etcd is a key-value store, so the MCP "sql" parameter is interpreted as
// a JSON command descriptor:
//
//	Query:   {"cmd":"get","key":"/mykey"}
//	         {"cmd":"list","prefix":"/myprefix/"}
//	         {"cmd":"list","prefix":"/myprefix/","limit":100}
//
//	Execute: {"cmd":"put","key":"/mykey","value":"myval"}
//	         {"cmd":"put","key":"/mykey","value":"myval","ttl":"1h"}
//	         {"cmd":"delete","key":"/mykey"}
//	         {"cmd":"delete","prefix":"/myprefix/"}
//
//	Schema:  GetTableNames lists keys by top-level prefix segments.
//	         GetTableSchema returns value + version + lease for a key.
type EtcdDriver struct{}

// Name returns the driver name
func (d *EtcdDriver) Name() string {
	return "etcd"
}

// Open connects to etcd. DSN is a comma-separated list of endpoints,
// optionally with username/password:
//
//	http://localhost:2379
//	http://localhost:2379,http://localhost:2380
//	etcd://user:password@localhost:2379
func (d *EtcdDriver) Open(dsn string) (Handle, error) {
	cfg, err := etcdConfigFromDSN(dsn)
	if err != nil {
		return nil, err
	}

	client, err := clientv3.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Status(ctx, cfg.Endpoints[0]); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to reach etcd: %w", err)
	}

	return client, nil
}

// Query executes a read command against etcd.
func (d *EtcdDriver) Query(h Handle, query string) ([]string, [][]interface{}, error) {
	client := h.(*clientv3.Client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var cmd struct {
		Cmd    string `json:"cmd"`
		Key    string `json:"key"`
		Prefix string `json:"prefix"`
		Limit  int64  `json:"limit"`
	}
	if err := json.Unmarshal([]byte(query), &cmd); err != nil {
		// Treat plain string as a key for get
		cmd.Cmd = "get"
		cmd.Key = query
	}
	if cmd.Limit == 0 {
		cmd.Limit = 100
	}

	switch strings.ToLower(cmd.Cmd) {
	case "get":
		resp, err := client.Get(ctx, cmd.Key)
		if err != nil {
			return nil, nil, err
		}
		columns := []string{"key", "value", "version", "mod_revision"}
		rows := make([][]interface{}, len(resp.Kvs))
		for i, kv := range resp.Kvs {
			rows[i] = []interface{}{string(kv.Key), string(kv.Value), kv.Version, kv.ModRevision}
		}
		return columns, rows, nil

	case "list":
		prefix := cmd.Prefix
		if prefix == "" {
			prefix = cmd.Key
		}
		if prefix == "" {
			prefix = "/"
		}
		resp, err := client.Get(ctx, prefix,
			clientv3.WithPrefix(),
			clientv3.WithLimit(cmd.Limit),
		)
		if err != nil {
			return nil, nil, err
		}
		columns := []string{"key", "value", "version", "mod_revision"}
		rows := make([][]interface{}, len(resp.Kvs))
		for i, kv := range resp.Kvs {
			rows[i] = []interface{}{string(kv.Key), string(kv.Value), kv.Version, kv.ModRevision}
		}
		return columns, rows, nil

	default:
		return nil, nil, fmt.Errorf("unknown query cmd %q: use get or list", cmd.Cmd)
	}
}

// Execute runs a write command against etcd.
func (d *EtcdDriver) Execute(h Handle, query string) (int64, error) {
	client := h.(*clientv3.Client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var cmd struct {
		Cmd    string `json:"cmd"`
		Key    string `json:"key"`
		Prefix string `json:"prefix"`
		Value  string `json:"value"`
		TTL    string `json:"ttl"`
	}
	if err := json.Unmarshal([]byte(query), &cmd); err != nil {
		return 0, fmt.Errorf("execute requires JSON: {\"cmd\":\"put|delete\",...}: %w", err)
	}

	switch strings.ToLower(cmd.Cmd) {
	case "put":
		var opts []clientv3.OpOption
		if cmd.TTL != "" {
			ttl, err := time.ParseDuration(cmd.TTL)
			if err != nil {
				return 0, fmt.Errorf("invalid ttl %q: %w", cmd.TTL, err)
			}
			lease, err := client.Grant(ctx, int64(ttl.Seconds()))
			if err != nil {
				return 0, fmt.Errorf("failed to create lease: %w", err)
			}
			opts = append(opts, clientv3.WithLease(lease.ID))
		}
		if _, err := client.Put(ctx, cmd.Key, cmd.Value, opts...); err != nil {
			return 0, err
		}
		return 1, nil

	case "delete":
		key := cmd.Key
		var opts []clientv3.OpOption
		if cmd.Prefix != "" {
			key = cmd.Prefix
			opts = append(opts, clientv3.WithPrefix())
		}
		resp, err := client.Delete(ctx, key, opts...)
		if err != nil {
			return 0, err
		}
		return resp.Deleted, nil

	default:
		return 0, fmt.Errorf("unknown execute cmd %q: use put or delete", cmd.Cmd)
	}
}

// GetTableNames returns top-level key prefixes (first path segment).
func (d *EtcdDriver) GetTableNames(h Handle) ([]string, error) {
	client := h.(*clientv3.Client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Get(ctx, "/", clientv3.WithPrefix(), clientv3.WithKeysOnly(), clientv3.WithLimit(1000))
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	var prefixes []string
	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		// Extract first path segment: /segment/...  -> /segment
		parts := strings.SplitN(strings.TrimPrefix(key, "/"), "/", 2)
		prefix := "/" + parts[0]
		if _, ok := seen[prefix]; !ok {
			seen[prefix] = struct{}{}
			prefixes = append(prefixes, prefix)
		}
	}
	return prefixes, nil
}

// GetTableSchema returns value, version, and lease info for a key.
func (d *EtcdDriver) GetTableSchema(h Handle, key string) ([]map[string]interface{}, error) {
	client := h.(*clientv3.Client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Get(ctx, key, clientv3.WithPrefix(), clientv3.WithLimit(1))
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return []map[string]interface{}{}, nil
	}

	kv := resp.Kvs[0]
	info := map[string]interface{}{
		"key":          string(kv.Key),
		"value":        string(kv.Value),
		"version":      kv.Version,
		"create_rev":   kv.CreateRevision,
		"mod_rev":      kv.ModRevision,
		"lease":        kv.Lease,
	}
	return []map[string]interface{}{info}, nil
}

// Close closes the etcd client.
func (d *EtcdDriver) Close(h Handle) error {
	return h.(*clientv3.Client).Close()
}

// etcdConfigFromDSN parses a DSN string into a clientv3.Config.
// Supported formats:
//
//	http://host:2379
//	https://host:2379
//	http://host:2379,http://host2:2379
//	etcd://user:password@host:2379
func etcdConfigFromDSN(dsn string) (clientv3.Config, error) {
	cfg := clientv3.Config{
		DialTimeout: 5 * time.Second,
	}

	// Handle etcd:// scheme with credentials
	if strings.HasPrefix(dsn, "etcd://") {
		rest := strings.TrimPrefix(dsn, "etcd://")
		if idx := strings.LastIndex(rest, "@"); idx >= 0 {
			creds := rest[:idx]
			host := rest[idx+1:]
			parts := strings.SplitN(creds, ":", 2)
			if len(parts) == 2 {
				cfg.Username = parts[0]
				cfg.Password = parts[1]
			}
			cfg.Endpoints = []string{"http://" + host}
		} else {
			cfg.Endpoints = []string{"http://" + rest}
		}
		return cfg, nil
	}

	// Comma-separated list of http/https endpoints
	endpoints := strings.Split(dsn, ",")
	for i, ep := range endpoints {
		endpoints[i] = strings.TrimSpace(ep)
	}
	cfg.Endpoints = endpoints
	return cfg, nil
}
