package main

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
	client   *mongo.Client
	files    *mongo.Collection
	metadata *mongo.Collection
}

func NewMongoDB(uri, dbName string) (*MongoDB, error) {
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, err
	}

	// ------------------------------------------------------------------------------------------- files
	files := client.Database(dbName).Collection("files")
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
	metadata := client.Database(dbName).Collection("metadata")
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

type Metadata struct {
	Hash     string   `bson:"hash"`
	Size     int64    `bson:"size"`
	Metadata []string `bson:"metadata"`
}

func (m *MongoDB) InsertOne(fileInfo *FileInfo) (err error) {
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

	_, err = m.metadata.InsertOne(context.Background(), Metadata{
		Hash: fileInfo.Hash,
		Size: fileInfo.Size,
		Metadata: fileInfo.Metadata,
	})
	// log.Printf("InsertOne: [%v], [%v]\n", fileInfo.Name, fileInfo.Hash)
	return err
}

func (m *MongoDB) Load(filename string) (*FileInfo, error) {
	var fileInfo FileInfo
	err := m.files.FindOne(context.Background(), bson.M{"name": filename}).Decode(&fileInfo)
	if err != nil {
		// log.Printf("files: [%v], [%v]\n", filename, err)
		return nil, err
	}

	var metadata Metadata
	err = m.metadata.FindOne(context.Background(), bson.M{"hash": fileInfo.Hash}).Decode(&metadata)
	if err != nil {
		// log.Printf("metadata: [%v], [%v]\n", filename, err)
		return nil, err
	}
	fileInfo.Size = metadata.Size
	fileInfo.Metadata = metadata.Metadata
	// log.Printf("Load: [%v], [%v], metadata: %v\n", fileInfo.Name, fileInfo.Hash, len(fileInfo.Metadata))
	return &fileInfo, nil
}


func (m *MongoDB) FindOne(name string, hash ...string) (*FileInfo, error) {
	var fileInfo FileInfo

	// log.Printf("FindOne: [%v], [%+v]\n", name, hash)
	err := m.files.FindOne(context.Background(), bson.M{"name": name}).Decode(&fileInfo)
	if err == nil {
		// log.Printf("FindOne name: [%v] err: [%v]\n", name, err)
		return &fileInfo, nil
	}

	if len(hash) > 0 {
		err = m.metadata.FindOne(context.Background(), bson.M{"hash": hash[0]}).Decode(&fileInfo)
		// log.Printf("FindOne hash: [%v] err: [%v]\n", hash[0], err)
	}
	return &fileInfo, err
}

func (m *MongoDB) Close() {
	m.client.Disconnect(context.Background())
}
