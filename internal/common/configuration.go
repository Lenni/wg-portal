package common

import (
	"errors"
	"os"
	"reflect"
	"runtime"

	"github.com/h44z/wg-portal/internal/ldap"
	"github.com/h44z/wg-portal/internal/users"
	"github.com/h44z/wg-portal/internal/wireguard"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var ErrInvalidSpecification = errors.New("specification must be a struct pointer")

// LoadConfigFile parses yaml files. It uses to yaml annotation to store the data in a struct.
func loadConfigFile(cfg interface{}, filename string) error {
	s := reflect.ValueOf(cfg)

	if s.Kind() != reflect.Ptr {
		return ErrInvalidSpecification
	}
	s = s.Elem()
	if s.Kind() != reflect.Struct {
		return ErrInvalidSpecification
	}

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		return err
	}

	return nil
}

// LoadConfigEnv processes envconfig annotations and loads environment variables to the given configuration struct.
func loadConfigEnv(cfg interface{}) error {
	err := envconfig.Process("", cfg)
	if err != nil {
		return err
	}

	return nil
}

type Config struct {
	Core struct {
		ListeningAddress  string `yaml:"listeningAddress" envconfig:"LISTENING_ADDRESS"`
		ExternalUrl       string `yaml:"externalUrl" envconfig:"EXTERNAL_URL"`
		Title             string `yaml:"title" envconfig:"WEBSITE_TITLE"`
		CompanyName       string `yaml:"company" envconfig:"COMPANY_NAME"`
		MailFrom          string `yaml:"mailFrom" envconfig:"MAIL_FROM"`
		AdminUser         string `yaml:"adminUser" envconfig:"ADMIN_USER"`
		AdminPassword     string `yaml:"adminPass" envconfig:"ADMIN_PASS"`
		EditableKeys      bool   `yaml:"editableKeys" envconfig:"EDITABLE_KEYS"`
		CreateDefaultPeer bool   `yaml:"createDefaultPeer" envconfig:"CREATE_DEFAULT_PEER"`
		LdapEnabled       bool   `yaml:"ldapEnabled" envconfig:"LDAP_ENABLED"`
	} `yaml:"core"`
	Database users.Config     `yaml:"database"`
	Email    MailConfig       `yaml:"email"`
	LDAP     ldap.Config      `yaml:"ldap"`
	WG       wireguard.Config `yaml:"wg"`
}

func NewConfig() *Config {
	cfg := &Config{}

	// Default config
	cfg.Core.ListeningAddress = ":8123"
	cfg.Core.Title = "WireGuard VPN"
	cfg.Core.CompanyName = "WireGuard Portal"
	cfg.Core.ExternalUrl = "http://localhost:8123"
	cfg.Core.MailFrom = "WireGuard VPN <noreply@company.com>"
	cfg.Core.AdminUser = "admin@wgportal.local"
	cfg.Core.AdminPassword = "wgportal"
	cfg.Core.LdapEnabled = false

	cfg.Database.Typ = "sqlite"
	cfg.Database.Database = "data/wg_portal.db"

	cfg.LDAP.URL = "ldap://srv-ad01.company.local:389"
	cfg.LDAP.BaseDN = "DC=COMPANY,DC=LOCAL"
	cfg.LDAP.StartTLS = true
	cfg.LDAP.BindUser = "company\\\\ldap_wireguard"
	cfg.LDAP.BindPass = "SuperSecret"
	cfg.LDAP.Type = "AD"
	cfg.LDAP.UserClass = "organizationalPerson"
	cfg.LDAP.EmailAttribute = "mail"
	cfg.LDAP.FirstNameAttribute = "givenName"
	cfg.LDAP.LastNameAttribute = "sn"
	cfg.LDAP.PhoneAttribute = "telephoneNumber"
	cfg.LDAP.GroupMemberAttribute = "memberOf"
	cfg.LDAP.DisabledAttribute = "userAccountControl"
	cfg.LDAP.AdminLdapGroup = "CN=WireGuardAdmins,OU=_O_IT,DC=COMPANY,DC=LOCAL"

	cfg.WG.DeviceName = "wg0"
	cfg.WG.WireGuardConfig = "/etc/wireguard/wg0.conf"
	cfg.WG.ManageIPAddresses = true
	cfg.Email.Host = "127.0.0.1"
	cfg.Email.Port = 25

	// Load config from file and environment
	cfgFile, ok := os.LookupEnv("CONFIG_FILE")
	if !ok {
		cfgFile = "config.yml" // Default config file
	}
	err := loadConfigFile(cfg, cfgFile)
	if err != nil {
		logrus.Warnf("unable to load config.yml file: %v, using default configuration...", err)
	}
	err = loadConfigEnv(cfg)
	if err != nil {
		logrus.Warnf("unable to load environment config: %v", err)
	}

	if cfg.WG.ManageIPAddresses && runtime.GOOS != "linux" {
		logrus.Warnf("Managing IP addresses only works on linux! Feature disabled.")
		cfg.WG.ManageIPAddresses = false
	}

	return cfg
}
