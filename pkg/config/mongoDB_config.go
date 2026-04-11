package config

type MongoDBConfig struct {
	MongoURI string
	Datatbse string
}

func LoadMongoDBConfig() MongoDBConfig {
	return MongoDBConfig{
		MongoURI: GetEnvWithDefault("MONGO_URI", "mongodb://localhost:27017"),
		Datatbse: GetEnvWithDefault("MONGO_DB", "voute"),
	}
}
