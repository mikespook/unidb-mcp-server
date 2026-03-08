package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// mongoHandle wraps a mongo client and the target database name
type mongoHandle struct {
	client *mongo.Client
	dbName string
}

// MongoDriver implements the Driver interface for MongoDB
type MongoDriver struct{}

// Name returns the driver name
func (d *MongoDriver) Name() string {
	return "mongodb"
}

// Open opens a MongoDB connection. DSN is a standard MongoDB URI.
// The database name is extracted from the URI path (e.g. mongodb://host/mydb).
func (d *MongoDriver) Open(dsn string) (Handle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Client().ApplyURI(dsn)
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		client.Disconnect(context.Background())
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	dbName := mongoDatabaseFromURI(dsn)

	return &mongoHandle{client: client, dbName: dbName}, nil
}

// Query executes a MongoDB query expressed as JSON.
// Format: {"collection":"name","filter":{...},"projection":{...},"limit":100}
// For compatibility with the MCP "sql" parameter, plain collection names are also accepted
// as a shorthand for a full collection scan: find({}).
func (d *MongoDriver) Query(h Handle, query string) ([]string, [][]interface{}, error) {
	mh := h.(*mongoHandle)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Parse the query as JSON
	var q struct {
		Collection string          `json:"collection"`
		Filter     json.RawMessage `json:"filter"`
		Projection json.RawMessage `json:"projection"`
		Limit      int64           `json:"limit"`
	}
	if err := json.Unmarshal([]byte(query), &q); err != nil {
		// Treat the whole string as a collection name for a simple find
		q.Collection = query
	}
	if q.Collection == "" {
		return nil, nil, fmt.Errorf("query must specify a collection name")
	}
	if q.Limit == 0 {
		q.Limit = 100
	}

	var filter bson.D
	if len(q.Filter) > 0 {
		if err := bson.UnmarshalExtJSON(q.Filter, true, &filter); err != nil {
			return nil, nil, fmt.Errorf("invalid filter: %w", err)
		}
	}

	findOpts := options.Find().SetLimit(q.Limit)
	if len(q.Projection) > 0 {
		var proj bson.D
		if err := bson.UnmarshalExtJSON(q.Projection, true, &proj); err != nil {
			return nil, nil, fmt.Errorf("invalid projection: %w", err)
		}
		findOpts.SetProjection(proj)
	}

	coll := mh.client.Database(mh.dbName).Collection(q.Collection)
	cursor, err := coll.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("find failed: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, nil, err
	}

	if len(docs) == 0 {
		return []string{}, [][]interface{}{}, nil
	}

	// Collect all unique keys as columns
	colSet := make(map[string]struct{})
	for _, doc := range docs {
		for k := range doc {
			colSet[k] = struct{}{}
		}
	}
	columns := make([]string, 0, len(colSet))
	for k := range colSet {
		columns = append(columns, k)
	}

	rows := make([][]interface{}, len(docs))
	for i, doc := range docs {
		row := make([]interface{}, len(columns))
		for j, col := range columns {
			v := doc[col]
			// Serialize complex values to JSON strings
			switch val := v.(type) {
			case bson.M, bson.D, bson.A:
				b, _ := json.Marshal(val)
				row[j] = string(b)
			default:
				row[j] = val
			}
		}
		rows[i] = row
	}

	return columns, rows, nil
}

// Execute runs a MongoDB command expressed as JSON.
// Format: {"collection":"name","op":"insert|update|delete","document":{...},"filter":{...}}
func (d *MongoDriver) Execute(h Handle, query string) (int64, error) {
	mh := h.(*mongoHandle)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var cmd struct {
		Collection string          `json:"collection"`
		Op         string          `json:"op"`
		Document   json.RawMessage `json:"document"`
		Filter     json.RawMessage `json:"filter"`
	}
	if err := json.Unmarshal([]byte(query), &cmd); err != nil {
		return 0, fmt.Errorf("execute requires JSON: {\"collection\":\"...\",\"op\":\"insert|update|delete\",...}: %w", err)
	}

	coll := mh.client.Database(mh.dbName).Collection(cmd.Collection)

	switch cmd.Op {
	case "insert":
		var doc bson.D
		if err := bson.UnmarshalExtJSON(cmd.Document, true, &doc); err != nil {
			return 0, fmt.Errorf("invalid document: %w", err)
		}
		_, err := coll.InsertOne(ctx, doc)
		if err != nil {
			return 0, err
		}
		return 1, nil

	case "update":
		var filter, update bson.D
		if err := bson.UnmarshalExtJSON(cmd.Filter, true, &filter); err != nil {
			return 0, fmt.Errorf("invalid filter: %w", err)
		}
		if err := bson.UnmarshalExtJSON(cmd.Document, true, &update); err != nil {
			return 0, fmt.Errorf("invalid update document: %w", err)
		}
		res, err := coll.UpdateMany(ctx, filter, update)
		if err != nil {
			return 0, err
		}
		return res.ModifiedCount, nil

	case "delete":
		var filter bson.D
		if err := bson.UnmarshalExtJSON(cmd.Filter, true, &filter); err != nil {
			return 0, fmt.Errorf("invalid filter: %w", err)
		}
		res, err := coll.DeleteMany(ctx, filter)
		if err != nil {
			return 0, err
		}
		return res.DeletedCount, nil

	default:
		return 0, fmt.Errorf("unknown op %q: use insert, update, or delete", cmd.Op)
	}
}

// GetTableNames lists all collections in the database
func (d *MongoDriver) GetTableNames(h Handle) ([]string, error) {
	mh := h.(*mongoHandle)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return mh.client.Database(mh.dbName).ListCollectionNames(ctx, bson.D{})
}

// GetTableSchema returns a sample document structure for a collection
func (d *MongoDriver) GetTableSchema(h Handle, collection string) ([]map[string]interface{}, error) {
	mh := h.(*mongoHandle)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := mh.client.Database(mh.dbName).Collection(collection)
	cursor, err := coll.Find(ctx, bson.D{}, options.Find().SetLimit(1))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return []map[string]interface{}{}, nil
	}

	schema := make([]map[string]interface{}, 0, len(docs[0]))
	for k, v := range docs[0] {
		schema = append(schema, map[string]interface{}{
			"field": k,
			"type":  fmt.Sprintf("%T", v),
		})
	}
	return schema, nil
}

// Close disconnects the MongoDB client
func (d *MongoDriver) Close(h Handle) error {
	return h.(*mongoHandle).client.Disconnect(context.Background())
}

// mongoDatabaseFromURI extracts the database name from a MongoDB URI.
// e.g. mongodb://host:27017/mydb -> "mydb"
func mongoDatabaseFromURI(uri string) string {
	// Find the path after the host
	// mongodb://[user:pass@]host[:port]/database[?options]
	for i := len("mongodb://"); i < len(uri); i++ {
		if uri[i] == '/' {
			rest := uri[i+1:]
			// Strip query string
			for j, c := range rest {
				if c == '?' {
					rest = rest[:j]
					break
				}
			}
			if rest != "" {
				return rest
			}
			break
		}
	}
	return "test"
}
