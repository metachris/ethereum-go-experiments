package ethtools

import (
	"fmt"
	"os"
	"strconv"
)

//go:generate stringer -type=Config
type Config struct {
	Database PostgresConfig

	EthNode         string
	EthplorerApiKey string

	WebserverHost string
	WebserverPort int

	NumAddressesByValueSent        int
	NumAddressesByValueReceived    int
	NumAddressesByNumTxSent        int
	NumAddressesByNumTxReceived    int
	NumAddressesByNumTokenTransfer int

	// Debug helpers
	Debug           bool
	CheckTxStatus   bool
	HideOutput      bool
	DebugPrintMevTx bool
}

func (c Config) String() string {
	return fmt.Sprintf("eth:%s psql:%s@%s/%s, numAddr:%d/%d/%d/%d/%d, debug=%t, checkTx=%t", c.EthNode, c.Database.User, c.Database.Host, c.Database.Name, c.NumAddressesByValueSent, c.NumAddressesByValueReceived, c.NumAddressesByNumTxSent, c.NumAddressesByNumTxReceived, c.NumAddressesByNumTokenTransfer, c.Debug, c.CheckTxStatus)
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

func getEnvBool(key string, defaultVal bool) bool {
	val, exists := os.LookupEnv(key)
	if exists && len(val) > 0 {
		return true
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

var config *Config

func GetConfig() *Config {
	if config != nil {
		return config
	}

	dbConfig := PostgresConfig{
		User:       getEnvStr("DB_USER", ""),
		Password:   getEnvStr("DB_PASS", ""),
		Host:       getEnvStr("DB_HOST", ""),
		Name:       getEnvStr("DB_NAME", ""),
		DisableTLS: len(getEnvStr("DB_DISABLE_TLS", "")) > 0,
	}

	if len(dbConfig.Host) == 0 {
		panic("Error: no DB_HOST environment variable set! Please check if you've set all environment variables.")
	}

	config = &Config{
		Database: dbConfig,

		WebserverHost: getEnvStr("WEBSERVER_HOST", ""),
		WebserverPort: getEnvInt("WEBSERVER_PORT", 8090),

		EthNode:         getEnvStr("ETH_NODE", ""),
		EthplorerApiKey: getEnvStr("ETHPLORER_API_KEY", "freekey"),

		NumAddressesByValueSent:        getEnvInt("NUM_ADDR_VALUE_SENT", 25),
		NumAddressesByValueReceived:    getEnvInt("NUM_ADDR_VALUE_RECEIVED", 25),
		NumAddressesByNumTxSent:        getEnvInt("NUM_ADDR_NUM_TX_SENT", 25),
		NumAddressesByNumTxReceived:    getEnvInt("NUM_ADDR_NUM_TX_RECEIVED", 25),
		NumAddressesByNumTokenTransfer: getEnvInt("NUM_ADDR_NUM_TOKEN_TRANSFER", 100),

		Debug:           getEnvBool("DEBUG", false),
		CheckTxStatus:   getEnvBool("CHECK_TX_STATUS", true),
		HideOutput:      getEnvBool("HIDE_OUTPUT", false),
		DebugPrintMevTx: getEnvBool("MEV", false),
	}

	return config
}
