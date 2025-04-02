package config

import (
	"fmt"
	"os"
	"strconv"
)

var AppConfig Config = Config{}

type Config struct {
	VMWIZ_SCHEME   string
	VMWIZ_HOSTNAME string
	VMWIZ_PORT     int

	PVE_HOST    string
	PVE_USER    string
	PVE_TOKENID string
	PVE_UUID    string

	SSH_CM_HOST            string
	SSH_CM_USER            string
	SSH_CM_PKEY_PASSPHRASE string

	COMP_NAME                string
	SSH_COMP_HOST            string
	SSH_COMP_USER            string
	SSH_COMP_PKEY_PASSPHRASE string

	NETCENTER_HOST string
	NETCENTER_USER string
	NETCENTER_PWD  string

	KEYCLOAK_ISSUER_URL    string
	KEYCLOAK_CLIENT_ID     string
	KEYCLOAK_CLIENT_SECRET string

	POSTGRES_USER     string
	POSTGRES_PASSWORD string
	POSTGRES_DB       string
}

func (c *Config) Init() error {
	VMWIZ_PORT, err := strconv.Atoi(os.Getenv("VMWIZ_PORT"))
	if err != nil {
		return fmt.Errorf("Failed to parse config: VMWIZ_PORT: %v", err.Error())
	}

	c.VMWIZ_SCHEME = os.Getenv("VMWIZ_SCHEME")
	c.VMWIZ_HOSTNAME = os.Getenv("VMWIZ_HOSTNAME")
	c.VMWIZ_PORT = VMWIZ_PORT

	c.PVE_HOST = os.Getenv("PVE_HOST")
	c.PVE_USER = os.Getenv("PVE_USER")
	c.PVE_TOKENID = os.Getenv("PVE_TOKENID")
	c.PVE_UUID = os.Getenv("PVE_UUID")

	c.SSH_CM_HOST = os.Getenv("SSH_CM_HOST")
	c.SSH_CM_USER = os.Getenv("SSH_CM_USER")
	c.SSH_CM_PKEY_PASSPHRASE = os.Getenv("SSH_CM_PKEY_PASSPHRASE")

	c.COMP_NAME = os.Getenv("COMP_NAME")
	c.SSH_COMP_HOST = os.Getenv("SSH_COMP_HOST")
	c.SSH_COMP_USER = os.Getenv("SSH_COMP_USER")
	c.SSH_COMP_PKEY_PASSPHRASE = os.Getenv("SSH_COMP_PKEY_PASSPHRASE")

	c.NETCENTER_HOST = os.Getenv("NETCENTER_HOST")
	c.NETCENTER_USER = os.Getenv("NETCENTER_USER")
	c.NETCENTER_PWD = os.Getenv("NETCENTER_PWD")

	c.KEYCLOAK_ISSUER_URL = os.Getenv("KEYCLOAK_ISSUER_URL")
	c.KEYCLOAK_CLIENT_ID = os.Getenv("KEYCLOAK_CLIENT_ID")
	c.KEYCLOAK_CLIENT_SECRET = os.Getenv("KEYCLOAK_CLIENT_SECRET")

	c.POSTGRES_USER = os.Getenv("POSTGRES_USER")
	c.POSTGRES_PASSWORD = os.Getenv("POSTGRES_PASSWORD")
	c.POSTGRES_DB = os.Getenv("POSTGRES_DB")

	return nil
}
