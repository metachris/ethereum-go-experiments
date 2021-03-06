package core

import (
	"fmt"
	"os"
	"strconv"
)

type PostgresConfig struct {
	User       string
	Password   string
	Host       string
	Name       string
	DisableTLS bool
}

type Config struct {
	EthNode string

	Database PostgresConfig

	WebserverHost string
	WebserverPort int

	NumTopAddresses    int
	NumTopTransactions int

	EthplorerApiKey string // not needed

	// Debug helpers
	Debug                 bool
	HideOutput            bool
	DebugPrintFlashbotsTx bool
	LowApiCallMode        bool
}

func (c Config) String() string {
	return fmt.Sprintf("eth:%s psql:%s@%s/%s, numAddr:%d, numTx:%d, debug=%t, lowApiCall=%t", c.EthNode, c.Database.User, c.Database.Host, c.Database.Name, c.NumTopAddresses, c.NumTopTransactions, c.Debug, c.LowApiCallMode)
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

// var config *Config
var Cfg Config = Config{
	Database: PostgresConfig{
		User:       getEnvStr("DB_USER", ""),
		Password:   getEnvStr("DB_PASS", ""),
		Host:       getEnvStr("DB_HOST", ""),
		Name:       getEnvStr("DB_NAME", ""),
		DisableTLS: len(getEnvStr("DB_DISABLE_TLS", "")) > 0,
	},

	WebserverHost: getEnvStr("WEBSERVER_HOST", ""),
	WebserverPort: getEnvInt("WEBSERVER_PORT", 8090),

	EthNode:         getEnvStr("ETH_NODE", ""),
	EthplorerApiKey: getEnvStr("ETHPLORER_API_KEY", "freekey"),

	NumTopAddresses:    getEnvInt("NUM_TOP_ADDR", 25),
	NumTopTransactions: getEnvInt("NUM_TOP_TX", 20),

	Debug:                 getEnvBool("DEBUG", false),
	HideOutput:            getEnvBool("HIDE_OUTPUT", false),
	DebugPrintFlashbotsTx: getEnvBool("MEV", false),
	LowApiCallMode:        getEnvBool("LOW_API", false),
}

func init() {
	if len(Cfg.EthNode) == 0 {
		panic("ETH_NODE environment variable not found")
	}
}
