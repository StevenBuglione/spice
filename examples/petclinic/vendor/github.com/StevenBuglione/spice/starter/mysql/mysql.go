// Package mysql provides a reviewed database/sql starter for MySQL.
package mysql

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
)

const (
	defaultApplicationName       = "spice"
	defaultMaxOpenConnections    = 20
	defaultMaxIdleConnections    = 10
	defaultConnectionMaxLifetime = 3 * time.Minute
	defaultConnectionMaxIdleTime = 1 * time.Minute
	defaultConnectTimeout        = 10 * time.Second
	maxConnectionURLBytes        = 16 << 10
	maxApplicationNameBytes      = 255
	maxConfiguredConnectionCount = 100_000
)

// Options defines a MySQL connection pool. URL must be a complete mysql URL.
// TLS certificate and hostname verification is enabled unless AllowInsecure is
// true and the URL explicitly contains tls=disable.
type Options struct {
	URL                   string
	ApplicationName       string
	MaxOpenConnections    int
	MaxIdleConnections    int
	ConnectionMaxLifetime time.Duration
	ConnectionMaxIdleTime time.Duration
	ConnectTimeout        time.Duration
	AllowInsecure         bool
}

// Open validates a complete connection URL and constructs a caller-owned
// database/sql pool. It performs no network I/O and registers no global driver
// configuration.
func Open(options Options) (*sql.DB, error) {
	config, normalized, err := driverConfig(options)
	if err != nil {
		return nil, err
	}
	connector, err := mysqldriver.NewConnector(config)
	if err != nil {
		return nil, errors.New("construct MySQL database: connection URL is invalid")
	}
	database := sql.OpenDB(connector)
	database.SetMaxOpenConns(normalized.MaxOpenConnections)
	database.SetMaxIdleConns(normalized.MaxIdleConnections)
	database.SetConnMaxLifetime(normalized.ConnectionMaxLifetime)
	database.SetConnMaxIdleTime(normalized.ConnectionMaxIdleTime)
	return database, nil
}

// Ping verifies one caller-owned database with the supplied context.
func Ping(ctx context.Context, database *sql.DB) error {
	switch {
	case ctx == nil:
		return errors.New("ping MySQL database: context is nil")
	case database == nil:
		return errors.New("ping MySQL database: database is nil")
	}
	if err := database.PingContext(ctx); err != nil {
		return fmt.Errorf("ping MySQL database: %w", err)
	}
	return nil
}

func driverConfig(options Options) (*mysqldriver.Config, Options, error) {
	normalized, parsed, insecure, err := normalizeOptions(options)
	if err != nil {
		return nil, Options{}, err
	}
	password, _ := parsed.User.Password()
	config := mysqldriver.NewConfig()
	config.User = parsed.User.Username()
	config.Passwd = password
	config.Net = "tcp"
	config.Addr = parsed.Host
	config.DBName = strings.TrimPrefix(parsed.EscapedPath(), "/")
	databaseName, err := url.PathUnescape(config.DBName)
	if err != nil {
		return nil, Options{}, errors.New("construct MySQL database: database name is invalid")
	}
	config.DBName = databaseName
	config.ParseTime = true
	config.ClientFoundRows = true
	config.Timeout = normalized.ConnectTimeout
	config.ConnectionAttributes = "program_name:" + normalized.ApplicationName
	if !insecure {
		config.TLS = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: parsed.Hostname(),
		}
	}
	return config, normalized, nil
}

func normalizeOptions(options Options) (Options, *url.URL, bool, error) {
	if options.URL == "" || len(options.URL) > maxConnectionURLBytes {
		return Options{}, nil, false, errors.New("construct MySQL database: connection URL is required")
	}
	parsed, err := url.Parse(options.URL)
	if err != nil || parsed == nil {
		return Options{}, nil, false, errors.New("construct MySQL database: connection URL is invalid")
	}
	if !validIdentity(parsed) {
		return Options{}, nil, false, errors.New("construct MySQL database: connection URL is incomplete")
	}
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		return Options{}, nil, false, errors.New("construct MySQL database: connection URL is invalid")
	}
	insecure := false
	for key, values := range query {
		if key != "tls" || len(values) != 1 || values[0] != "disable" {
			return Options{}, nil, false, errors.New("construct MySQL database: URL option is not permitted")
		}
		if !options.AllowInsecure {
			return Options{}, nil, false, errors.New("construct MySQL database: insecure TLS is not permitted")
		}
		insecure = true
	}

	if options.ApplicationName == "" {
		options.ApplicationName = defaultApplicationName
	}
	if !validApplicationName(options.ApplicationName) {
		return Options{}, nil, false, errors.New("construct MySQL database: application name is invalid")
	}
	if err := normalizePool(&options); err != nil {
		return Options{}, nil, false, err
	}
	parsed.RawQuery = ""
	return options, parsed, insecure, nil
}

func validIdentity(parsed *url.URL) bool {
	if !validURLShape(parsed) || !validCredentials(parsed.User) {
		return false
	}
	port, err := strconv.ParseUint(parsed.Port(), 10, 16)
	if err != nil || port == 0 {
		return false
	}
	hostname := parsed.Hostname()
	return net.ParseIP(hostname) != nil || validHostname(hostname)
}

func validURLShape(parsed *url.URL) bool {
	return parsed.Scheme == "mysql" &&
		parsed.User != nil &&
		parsed.Host != "" &&
		parsed.Hostname() != "" &&
		parsed.Port() != "" &&
		parsed.Path != "" &&
		parsed.Path != "/" &&
		parsed.Fragment == ""
}

func validCredentials(user *url.Userinfo) bool {
	password, present := user.Password()
	return user.Username() != "" && present && password != ""
}

func validHostname(hostname string) bool {
	if len(hostname) > 253 || strings.HasPrefix(hostname, ".") ||
		strings.HasSuffix(hostname, ".") {
		return false
	}
	for label := range strings.SplitSeq(hostname, ".") {
		if !validHostnameLabel(label) {
			return false
		}
	}
	return true
}

func validHostnameLabel(label string) bool {
	if label == "" || len(label) > 63 ||
		label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}
	for _, character := range []byte(label) {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '-' {
			continue
		}
		return false
	}
	return true
}

func validApplicationName(name string) bool {
	if name == "" || len(name) > maxApplicationNameBytes {
		return false
	}
	for _, character := range []byte(name) {
		if character < 0x20 || character > 0x7e || character == ',' || character == ':' {
			return false
		}
	}
	return true
}

func normalizePool(options *Options) error {
	if options.MaxOpenConnections == 0 {
		options.MaxOpenConnections = defaultMaxOpenConnections
	}
	if options.MaxIdleConnections == 0 {
		options.MaxIdleConnections = defaultMaxIdleConnections
	}
	if options.ConnectionMaxLifetime == 0 {
		options.ConnectionMaxLifetime = defaultConnectionMaxLifetime
	}
	if options.ConnectionMaxIdleTime == 0 {
		options.ConnectionMaxIdleTime = defaultConnectionMaxIdleTime
	}
	if options.ConnectTimeout == 0 {
		options.ConnectTimeout = defaultConnectTimeout
	}
	switch {
	case options.MaxOpenConnections < 1 ||
		options.MaxOpenConnections > maxConfiguredConnectionCount:
		return errors.New("construct MySQL database: max open connections is invalid")
	case options.MaxIdleConnections < 1 ||
		options.MaxIdleConnections > options.MaxOpenConnections:
		return errors.New("construct MySQL database: max idle connections is invalid")
	case options.ConnectionMaxLifetime < 0:
		return errors.New("construct MySQL database: connection max lifetime is invalid")
	case options.ConnectionMaxIdleTime < 0:
		return errors.New("construct MySQL database: connection max idle time is invalid")
	case options.ConnectTimeout < 0:
		return errors.New("construct MySQL database: connect timeout is invalid")
	default:
		return nil
	}
}
