package config

type Config struct {
	Hosts struct {
		Etcd struct {
			Address          []string
			RegisterTTL      int
			RegisterInterval int
		}
		Sqlite struct {
			DBPath    string
			Password  string
			IfShowSql bool
			IfSyncDB  bool
		}
	}
	Project     string
	ServiceName string
	Env         string
	Version     string
	HealthCheck bool
}
