/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package config

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	etcdV3Client "go.etcd.io/etcd/client/v3"
)

// FreeAccountStr is the string used to identify a free account
const FreeAccountStr = "free"

func getEnv(key string) string {
	return os.Getenv(strings.ToUpper(key))
}

// GetGoPath returns gopath if defined
func GetGoPath() string {
	if goPath := getEnv("GOPATH"); goPath != "" {
		return goPath
	}
	return ""
}

// GetConfigPath returns path of toml config file
func GetConfigPath() string {
	goPath := GetGoPath()
	srcPath := filepath.Join("src", "github.ibm.com", "alchemy-containers", "armada-performance", "api")
	return filepath.Join(goPath, srcPath, "config")
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

// Config ...
type Config struct {
	API       *APIServerConfig
	Bluemix   *BluemixConfig
	Etcd      *EtcdConfig
	Location  *LocationConfig
	Softlayer *SoftlayerConfig
	Bootstrap *BootstrapConfig
	Deploy    *DeployConfig
	Request   *RequestConfig
}

// APIServerConfig defines properties required for interaction with an Armada API Server
type APIServerConfig struct {
	APIServerScheme string `toml:"api_server_scheme"`
	APIServerIP     string `toml:"api_server_ip"`
	APIServerPort   string `toml:"api_server_port"`
	APIVersion      string `toml:"api_version"`
}

// EtcdConfig defines properties required for interaction an etcd server.
// Only used when running in dummy mode.
type EtcdConfig struct {
	EtcdVersion         int      `toml:"etcd_api_version"`
	EtcdEndpoints       string   `toml:"etcd_endpoints"`
	EtcdTimeout         duration `toml:"etcd_timeout"`
	EtcdAuth            bool     `toml:"etcd_auth"`
	EtcdUser            string   `toml:"etcd_user"`
	EtcdPassword        string   `toml:"etcd_password"`
	EtcdCert            string   `toml:"etcd_cert"`
	EtcdFakeSLEndpoints string   `toml:"etcd_fakesl_endpoints"`
	EtcdRetries         int      `toml:"etcd_retries"`
}

// LocationConfig defines location properties required for interaction with an Armada API Server
type LocationConfig struct {
	Region      string `toml:"region"`
	Datacenter  string `toml:"datacenter"`
	Environment string `toml:"environment"`
}

// RequestConfig contains miscellaneous configuration items
type RequestConfig struct {
	CreateCluster     string `toml:"create_clusters"`
	AddWorkers        string `toml:"add_workers"`
	CreateWorkerPool  string `toml:"create_worker_pool"`
	ResizeWorkerPool  string `toml:"resize_worker_pool"`
	AddWorkerPoolZone string `toml:"add_worker_pool_zone"`

	// Folowing properties are supplied via command line
	AdminConfig             bool
	PrivateVLAN, PublicVLAN bool
	DeleteResources         bool
	ShowResources           bool
	WorkerPollInterval      duration `toml:"worker_poll_interval"`
	MasterPollInterval      duration
}

// SoftlayerConfig contains softlayer account details
type SoftlayerConfig struct {
	SoftlayerDummy              bool   `toml:"softlayer_dummy"`
	SoftlayerUsername           string `toml:"softlayer_username"`
	SoftlayerAPIKey             string `toml:"softlayer_api_key"`
	SoftlayerPrivateVLAN        string `toml:"softlayer_private_vlan"`
	SoftlayerPublicVLAN         string `toml:"softlayer_public_vlan"`
	SoftlayerChurnPrivateVLAN   string `toml:"softlayer_churn_private_vlan"`
	SoftlayerChurnPublicVLAN    string `toml:"softlayer_churn_public_vlan"`
	SoftlayerBilling            string `toml:"softlayer_billing"`
	SoftlayerIsolation          string `toml:"softlayer_isolation"`
	SoftlayerPortableSubnet     bool   `toml:"softlayer_portable_subnet"`
	SoftlayerPortableSubnetSize int    `toml:"softlayer_portable_subnet_size"`
	SoftlayerDiskEncryption     bool   `toml:"softlayer_disk_encryption"`
}

// BootstrapConfig contains worker bootstrap control information
type BootstrapConfig struct {
	BootstrapDummy       bool     `toml:"bootstrap_dummy"`
	BootstrapWorkerDelay duration `toml:"bootstrap_worker_delay"`
}

// DeployConfig contains master deployment control information
type DeployConfig struct {
	DeployDummy       bool     `toml:"deploy_dummy"`
	DeployMasterDelay duration `toml:"deploy_master_delay"`
}

// BluemixConfig contains Bluemix account details
type BluemixConfig struct {
	BluemixDummy bool   `toml:"bluemix_dummy"`
	IAMURL       string `toml:"iam_url"`
	Username     string `toml:"username"`
	Password     string `toml:"password"`
	APIKey       string `toml:"api_key"`
	IAMToken     string `toml:"iam_token"`
	AccountID    string `toml:"account_id"`
}

// ParseConfig ...
func ParseConfig(filePath string, conf interface{}) {
	if _, err := toml.DecodeFile(filePath, conf); err != nil {
		if len(GetGoPath()) == 0 {
			log.Println("GOPATH isn't set !!!!")
		}
		log.Fatalf("Error parsing config file : %s\n", err.Error())
	}
}

// GetConfigString ...
func GetConfigString(envKey, defaultConf string) string {
	if val := getEnv(envKey); val != "" {
		return val
	}
	return defaultConf
}

// InitEtcdV3Client Helper method to get an EtcdClient v3 from the config
func InitEtcdV3Client(conf *EtcdConfig) *etcdV3Client.Client {
	authRequired := conf.EtcdAuth
	var user, password string
	var tlsConfig *tls.Config

	if authRequired {
		user = GetConfigString("armada_etcd_user", conf.EtcdUser)
		password = GetConfigString("armada_etcd_password", conf.EtcdPassword) // pragma: allowlist secret
		log.Printf("Etcd Authentication enabled, authenticating using user %s\n", user)
		if len(user) == 0 {
			log.Fatalln("Auth=true but no username found for etcd client")
		}
		caPath := GetConfigString("armada_etcd_cert", conf.EtcdCert)
		// #nosec G304
		caCert, err := ioutil.ReadFile(caPath)
		if err != nil {
			log.Fatalf("Error occurred reading file: %s", err.Error())
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig = &tls.Config{
			RootCAs: caCertPool,
		}
	}
	endpoints := GetConfigString("armada_etcd_endpoints", conf.EtcdEndpoints)

	etcdCfg := etcdV3Client.Config{
		Endpoints: strings.Split(endpoints, ","),
		Username:  user,
		Password:  password, // pragma: allowlist secret
		TLS:       tlsConfig,
		// Timeout per request to fail fast when the target endpoint is unavailable
		DialTimeout: conf.EtcdTimeout.Duration,
	}

	etcdCLIv3, err := etcdV3Client.New(etcdCfg)
	if err != nil {
		log.Fatalln(err.Error())
	}
	return etcdCLIv3
}
