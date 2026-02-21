package db

type PostgresConnectionConfig struct {
	Host   string
	Port   int
	User   string
	Passwd string
	DBName string
}
