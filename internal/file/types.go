package file

// Info represents the file information.
type Info struct {
	Name     string   `json:"name"`
	Hash     string   `json:"hash"`
	Size     int64    `json:"size,omitempty"`
	Metadata []string `json:"metadata,omitempty"`
}

// Meta represents the file metadata.
type Meta struct {
	Hash     string   `bson:"hash"`
	Size     int64    `bson:"size"`
	Metadata []string `bson:"metadata"`
}

