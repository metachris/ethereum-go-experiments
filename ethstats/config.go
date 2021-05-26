package ethstats

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

	NumTopAddresses      int
	NumTopAddressesLarge int
	NumTopTransactions   int

	// Debug helpers
	Debug                 bool
	HideOutput            bool
	DebugPrintFlashbotsTx bool
}

func (c Config) String() string {
	return fmt.Sprintf("eth:%s psql:%s@%s/%s, numAddr:%d/%d, debug=%t", c.EthNode, c.Database.User, c.Database.Host, c.Database.Name, c.NumTopAddresses, c.NumTopAddressesLarge, c.Debug)
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

	config = &Config{
		Database: dbConfig,

		WebserverHost: getEnvStr("WEBSERVER_HOST", ""),
		WebserverPort: getEnvInt("WEBSERVER_PORT", 8090),

		EthNode:         getEnvStr("ETH_NODE", ""),
		EthplorerApiKey: getEnvStr("ETHPLORER_API_KEY", "freekey"),

		NumTopAddresses:      getEnvInt("NUM_TOP_ADDR", 25),
		NumTopAddressesLarge: getEnvInt("NUM_TOP_ADDR_L", 100),
		NumTopTransactions:   getEnvInt("NUM_TOP_TX", 20),

		Debug:                 getEnvBool("DEBUG", false),
		HideOutput:            getEnvBool("HIDE_OUTPUT", false),
		DebugPrintFlashbotsTx: getEnvBool("MEV", false),
	}

	if len(config.EthNode) == 0 {
		panic("ETH_NODE environment variable not found")
		// panic("Error: no DB_HOST environment variable set! Please check if you've set all environment variables.")
	}

	return config
}
