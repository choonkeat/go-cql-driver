package cql

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gocql/gocql"
)

// NewClusterConfig returns a new gocql ClusterConfig
func NewClusterConfig(hosts ...string) *gocql.ClusterConfig {
	clusterConfig := gocql.NewCluster(hosts...)
	if len(clusterConfig.Hosts) < 1 {
		clusterConfig.Hosts = []string{"127.0.0.1"}
	}
	return clusterConfig
}

// ClusterConfigToConfigString converts a gocql ClusterConfig to a config string
// https://godoc.org/github.com/gocql/gocql#ClusterConfig
func ClusterConfigToConfigString(clusterConfig *gocql.ClusterConfig) string {
	clusterConfigDefault := gocql.NewCluster()
	stringConfig := strings.Join(clusterConfig.Hosts, ",") + "?"

	if clusterConfig.Consistency != clusterConfigDefault.Consistency {
		consistency, ok := DbConsistency[clusterConfig.Consistency]
		if !ok {
			panic(fmt.Sprint("clusterConfig.Consistency value not found in DbConsistency: ", clusterConfig.Consistency))
		}
		stringConfig += "consistency=" + consistency + "&"
	}
	if clusterConfig.Timeout >= 0 {
		stringConfig += "timeout=" + clusterConfig.Timeout.String() + "&"
	}
	if clusterConfig.ConnectTimeout >= 0 {
		stringConfig += "connectTimeout=" + clusterConfig.ConnectTimeout.String() + "&"
	}
	if clusterConfig.Keyspace != "" {
		stringConfig += "keyspace=" + clusterConfig.Keyspace + "&"
	}
	if clusterConfig.NumConns > 1 {
		stringConfig += "numConns=" + strconv.FormatInt(int64(clusterConfig.NumConns), 10) + "&"
	}
	if clusterConfig.IgnorePeerAddr != clusterConfigDefault.IgnorePeerAddr {
		stringConfig += "ignorePeerAddr=" + fmt.Sprint(clusterConfig.IgnorePeerAddr) + "&"
	}
	if clusterConfig.DisableInitialHostLookup != clusterConfigDefault.DisableInitialHostLookup {
		stringConfig += "disableInitialHostLookup=" + fmt.Sprint(clusterConfig.DisableInitialHostLookup) + "&"
	}
	if clusterConfig.WriteCoalesceWaitTime != clusterConfigDefault.WriteCoalesceWaitTime {
		stringConfig += "writeCoalesceWaitTime=" + fmt.Sprint(clusterConfig.WriteCoalesceWaitTime) + "&"
	}

	if clusterConfig.Authenticator != nil {
		passwordAuthenticator, ok := clusterConfig.Authenticator.(gocql.PasswordAuthenticator)
		if ok {
			if passwordAuthenticator.Username != "" {
				stringConfig += "username=" + url.QueryEscape(passwordAuthenticator.Username) + "&"
			}
			if passwordAuthenticator.Password != "" {
				stringConfig += "password=" + url.QueryEscape(passwordAuthenticator.Password) + "&"
			}
		}
	}

	if sslOpts := clusterConfig.SslOpts; sslOpts != nil {
		defaultSslOpts := gocql.SslOptions{}
		if s := strconv.FormatBool(sslOpts.EnableHostVerification); sslOpts.EnableHostVerification != defaultSslOpts.EnableHostVerification {
			stringConfig += "enableHostVerification=" + s + "&"
		}
		if s := sslOpts.KeyPath; sslOpts.KeyPath != defaultSslOpts.KeyPath {
			stringConfig += "keyPath=" + url.QueryEscape(s) + "&"
		}
		if s := sslOpts.CertPath; sslOpts.CertPath != defaultSslOpts.CertPath {
			stringConfig += "certPath=" + url.QueryEscape(s) + "&"
		}
		if s := sslOpts.CaPath; sslOpts.CaPath != defaultSslOpts.CaPath {
			stringConfig += "caPath=" + url.QueryEscape(s) + "&"
		}
	}

	return stringConfig[:len(stringConfig)-1]
}

// ConfigStringToClusterConfig converts a config string to a gocql ClusterConfig
func ConfigStringToClusterConfig(configString string) (*gocql.ClusterConfig, error) {
	clusterConfig := NewClusterConfig()
	configStringSplit := strings.SplitN(configString, "?", 2)

	if len(configStringSplit[0]) > 1 {
		hostsSplit := strings.Split(configStringSplit[0], ",")
		if len(hostsSplit) > 0 {
			clusterConfig.Hosts = make([]string, len(hostsSplit))
			for i := 0; i < len(hostsSplit); i++ {
				clusterConfig.Hosts[i] = strings.TrimSpace(hostsSplit[i])
			}
		}
	}

	passwordAuthenticator := gocql.PasswordAuthenticator{}
	sslOpts := gocql.SslOptions{}

	if len(configStringSplit) > 1 && len(configStringSplit[1]) > 1 {
		dataSplit := strings.Split(configStringSplit[1], "&")
		if len(dataSplit) > 0 {
			for i := 0; i < len(dataSplit); i++ {
				settingSplit := strings.SplitN(dataSplit[i], "=", 2)
				if len(settingSplit) != 2 {
					return nil, fmt.Errorf("missing =")
				}
				key, value := strings.TrimSpace(settingSplit[0]), settingSplit[1]
				switch key {
				case "consistency":
					consistency, ok := DbConsistencyLevels[value]
					if !ok {
						return nil, fmt.Errorf("failed for: %v = %v", key, value)
					}
					clusterConfig.Consistency = gocql.Consistency(consistency)
				case "keyspace":
					if value == "" {
						return nil, fmt.Errorf("failed for: %v = %v", key, value)
					}
					clusterConfig.Keyspace = value
				case "timeout":
					data, err := time.ParseDuration(value)
					if err != nil {
						return nil, fmt.Errorf("failed for: %v = %v", key, value)
					}
					if data >= 0 {
						clusterConfig.Timeout = data
					}
				case "connectTimeout":
					data, err := time.ParseDuration(value)
					if err != nil {
						return nil, fmt.Errorf("failed for: %v = %v", key, value)
					}
					if data >= 0 {
						clusterConfig.ConnectTimeout = data
					}
				case "numConns":
					data, err := strconv.ParseInt(value, 10, 64)
					if err != nil {
						return nil, fmt.Errorf("failed for: %v = %v", key, value)
					}
					if data > 0 {
						clusterConfig.NumConns = int(data)
					}
				case "ignorePeerAddr":
					data, err := strconv.ParseBool(value)
					if err != nil {
						return nil, fmt.Errorf("failed for: %v = %v", key, value)
					}
					clusterConfig.IgnorePeerAddr = data
				case "disableInitialHostLookup":
					data, err := strconv.ParseBool(value)
					if err != nil {
						return nil, fmt.Errorf("failed for: %v = %v", key, value)
					}
					clusterConfig.DisableInitialHostLookup = data
				case "writeCoalesceWaitTime":
					data, err := time.ParseDuration(value)
					if err != nil {
						return nil, fmt.Errorf("failed for: %v = %v", key, value)
					}
					clusterConfig.WriteCoalesceWaitTime = data
				case "username":
					data, err := url.QueryUnescape(value)
					if err != nil {
						return nil, fmt.Errorf("failed for: %v = %v", key, value)
					}
					passwordAuthenticator.Username = data
					clusterConfig.Authenticator = passwordAuthenticator
				case "password":
					data, err := url.QueryUnescape(value)
					if err != nil {
						return nil, fmt.Errorf("failed for: %v = %v", key, value)
					}
					passwordAuthenticator.Password = data
					clusterConfig.Authenticator = passwordAuthenticator
				case "enableHostVerification":
					data, err := strconv.ParseBool(value)
					if err != nil {
						return nil, fmt.Errorf("failed for: %v = %v", key, value)
					}
					sslOpts.EnableHostVerification = data
					clusterConfig.SslOpts = &sslOpts
				case "certPath":
					data, err := url.QueryUnescape(value)
					if err != nil {
						return nil, fmt.Errorf("failed for: %v = %v", key, value)
					}
					sslOpts.CertPath = data
					clusterConfig.SslOpts = &sslOpts
				case "keyPath":
					data, err := url.QueryUnescape(value)
					if err != nil {
						return nil, fmt.Errorf("failed for: %v = %v", key, value)
					}
					sslOpts.KeyPath = data
					clusterConfig.SslOpts = &sslOpts
				case "caPath":
					data, err := url.QueryUnescape(value)
					if err != nil {
						return nil, fmt.Errorf("failed for: %v = %v", key, value)
					}
					sslOpts.CaPath = data
					clusterConfig.SslOpts = &sslOpts
				default:
					return nil, fmt.Errorf("invalid key: %v", key)
				}
			}
		}
	}

	return clusterConfig, nil
}
