package ethtools

import (
	"fmt"
	"os"
	"strconv"
)

//go:generate stringer -type=Config
type Config struct {
	Database PostgresConfig

	EthplorerApiKey string
	WebserverPort   int

	NumAddressesByValueSent      int
	NumAddressesByValueReceived  int
	NumAddressesByNumTxSent      int
	NumAddressesByNumTxReceived  int
	NumAddressesByTokenTransfers int
}

func (c Config) String() string {
	return fmt.Sprintf("psql:%s@%s/%s, numAddr:%d/%d/%d/%d/%d", c.Database.User, c.Database.Host, c.Database.Name, c.NumAddressesByValueSent, c.NumAddressesByValueReceived, c.NumAddressesByNumTxSent, c.NumAddressesByNumTxReceived, c.NumAddressesByTokenTransfers)
}

type PostgresConfig struct {
	User       string
	Password   string
	Host       string
	Name       string
	DisableTLS bool
}

func getEnvStr(key string, defaultVal string) string {
	val, exists := os.LookupEnv(key)
	if exists {
		return val
	} else {
		return defaultVal
	}
}

func getEnvInt(key string, defaultVal int) int {
	val, exists := os.LookupEnv(key)
	if exists {
		intVal, err := strconv.Atoi(val)
		if err != nil {
			panic(fmt.Sprintf("Invalid value for key %s = %s", key, val))
		}
		return intVal
	} else {
		return defaultVal
	}
}

func GetConfig() Config {
	dbConfig := PostgresConfig{
		User:       getEnvStr("DB_USER", ""),
		Password:   getEnvStr("DB_PASS", ""),
		Host:       getEnvStr("DB_HOST", ""),
		Name:       getEnvStr("DB_NAME", ""),
		DisableTLS: len(getEnvStr("DB_DISABLE_TLS", "")) > 0,
	}

	return Config{
		Database: dbConfig,

		EthplorerApiKey: getEnvStr("ETHPLORER_API_KEY", "freekey"),
		WebserverPort:   getEnvInt("WEBSERVER_PORT", 8090),

		NumAddressesByValueSent:      getEnvInt("NUM_ADDR_VALUE_SENT", 25),
		NumAddressesByValueReceived:  getEnvInt("NUM_ADDR_VALUE_RECEIVED", 25),
		NumAddressesByNumTxSent:      getEnvInt("NUM_ADDR_NUM_TX_SENT", 25),
		NumAddressesByNumTxReceived:  getEnvInt("NUM_ADDR_NUM_TX_RECEIVED", 25),
		NumAddressesByTokenTransfers: getEnvInt("NUM_ADDR_TOKEN_TRANSFERS", 100),
	}
}
