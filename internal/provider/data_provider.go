package provider

type DataProviderInfo interface {
	GetStorageLocationInfo() (string, error)

	// FullyCommitted indicates whether the state of this provider is fully
	// committed, i.e., whether it exactly matches the committed state of the
	// provider.
	// For a filesystem-backed provider, this would be true if all data on disk
	// matches the potentially cached in-memory state. For a database-backed
	// provider which always retrieves data via queries this would potentially
	// always be true. Etc.
	FullyCommitted() (bool, error)
}
