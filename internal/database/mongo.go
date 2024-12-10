package database

import (
	"context"
	"dcloud/internal/file"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	filesCollection    = "files"
	metadataCollection = "metadata"
	timeout = 5 * time.Second
)

type MongoDB struct {
	client   *mongo.Client
	files    *mongo.Collection
	metadata *mongo.Collection
}

// Connect connects to the MongoDB and returns a new MongoDB instance.
func Connect(uri string) (*MongoDB, error) {
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid MongoDB URI: %v", err)
	}

	dbName := strings.Trim(parsedURI.Path, "/")
	if dbName == "" {
		return nil, fmt.Errorf("invalid database name in MongoDB URI [%v]", uri)
	}

	parsedURI.Path = ""
	uri = parsedURI.String()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
			return nil, err
	}

	// ------------------------------------------------------------------------------------------- files
	files := client.Database(dbName).Collection(filesCollection)
	indexModel := []mongo.IndexModel{
		{
			Keys:    bson.M{"name": 1},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.M{"hash": 1},
		},
	}

	if _, err := files.Indexes().CreateMany(context.Background(), indexModel); err != nil {
		return nil, err
	}
	// ------------------------------------------------------------------------------------------- /files

	// ------------------------------------------------------------------------------------------- metadata
	metadata := client.Database(dbName).Collection(metadataCollection)
	indexModel = []mongo.IndexModel{
		{
			Keys:    bson.M{"hash": 1},
			Options: options.Index().SetUnique(true),
		},
	}

	if _, err := metadata.Indexes().CreateMany(context.Background(), indexModel); err != nil {
			return nil, err
	}
	// ------------------------------------------------------------------------------------------- /metadata

	return &MongoDB{
		client:   client,
		files:    files,
		metadata: metadata,
	}, nil
}

// Store stores the file info and metadata in the MongoDB.
func (m *MongoDB) Store(fileInfo *file.Info) (err error) {
    // Check if the file already exists in the 'files' collection
    filter := bson.M{"name": fileInfo.Name} //, "hash": file.Hash}
    count, err := m.files.CountDocuments(context.Background(), filter)
    if err != nil {
        return err
    }

    if count > 0 {
        return fmt.Errorf("file with name '%s' and hash '%s' already exists", fileInfo.Name, fileInfo.Hash)
    }

    _, err = m.files.InsertOne(context.Background(), struct{
        Name string `bson:"name"`
        Hash string `bson:"hash"`
    }{
        Name: fileInfo.Name,
        Hash: fileInfo.Hash,
    })

    if err != nil {
        return err
    }

    if len(fileInfo.Metadata) == 0 {
        return nil
    }

    metadataFilter := bson.M{"hash": fileInfo.Hash}
    count, err = m.metadata.CountDocuments(context.Background(), metadataFilter)
    if err != nil {
        return err
    }

    if count > 0 {
        return fmt.Errorf("metadata with hash '%s' already exists", fileInfo.Hash)
    }

    _, err = m.metadata.InsertOne(context.Background(), file.Meta{
        Hash: fileInfo.Hash,
        Size: fileInfo.Size,
        Metadata: fileInfo.Metadata,
    })
    return err
}

// Load loads the file info from the MongoDB by name or hash.
func (m *MongoDB) Load(name string, hash ...string) (*file.Info, error) {
    var fileInfo file.Info

    // Try to find by name first
    err := m.files.FindOne(context.Background(), bson.M{"name": name}).Decode(&fileInfo)
    if err == nil {
        // Load metadata if file found
        var metadata file.Meta
        err = m.metadata.FindOne(context.Background(), bson.M{"hash": fileInfo.Hash}).Decode(&metadata)
        if err == nil {
            fileInfo.Size = metadata.Size
            fileInfo.Metadata = metadata.Metadata
        }
        return &fileInfo, nil
    }

    // If not found by name and hash is provided, try finding by hash
    if len(hash) > 0 {
        var metadata file.Meta
        err = m.metadata.FindOne(context.Background(), bson.M{"hash": hash[0]}).Decode(&metadata)
        if err == nil {
            fileInfo.Hash = hash[0]
            fileInfo.Size = metadata.Size
            fileInfo.Metadata = metadata.Metadata
            return &fileInfo, nil
        }
    }
    return nil, err
}
