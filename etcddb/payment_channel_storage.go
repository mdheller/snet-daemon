package etcddb

import (
	"strings"

	"github.com/singnet/snet-daemon/config"
	"github.com/spf13/viper"
)

// PaymentChannelStorageServerConf contains embedded etcd server config
// ID - unique name of the etcd server node
// Scheme - URL schema used to create client and peer and urls
// Host - host where the etcd server is executed
// ClientPort - port to listen clients, used together with
//              Schema and host to compose listen-client-urls (see link below)
// PeerPort - port to listen etcd peers, used together with
//              Schema and host to compose listen-client-urls (see link below)
// Token - unique initial cluster token. Using unique token etcd can generate unique
//         cluster IDs and member IDs for the clusters even if they otherwise have
//         the exact same configuration. This can protect etcd from
//         cross-cluster-interaction, which might corrupt the clusters.
// Enabled - enable running embedded etcd server
// For more details see etcd Clustering Guide link:
// https://github.com/etcd-io/etcd/blob/master/Documentation/op-guide/clustering.md
type PaymentChannelStorageServerConf struct {
	ID         string
	Scheme     string
	Host       string
	ClientPort int `mapstructure:"CLIENT_PORT"`
	PeerPort   int `mapstructure:"PEER_PORT"`
	Token      string
	Cluster    string
	Enabled    bool
}

// GetPaymentChannelStorageServerConf gets PaymentChannelStorageServerConf from viper
// The DefaultPaymentChannelStorageServerConf is used in case the PAYMENT_CHANNEL_STORAGE_SERVER field
// is not set in the configuration file
func GetPaymentChannelStorageServerConf(vip *viper.Viper) (conf *PaymentChannelStorageServerConf, err error) {

	conf = &PaymentChannelStorageServerConf{
		ID:         "storage-1",
		Scheme:     "http",
		Host:       "127.0.0.1",
		ClientPort: 2379,
		PeerPort:   2380,
		Token:      "unique-token",
		Cluster:    "storage-1=http://127.0.0.1:2380",
		Enabled:    false,
	}

	if !vip.InConfig(strings.ToLower(config.PaymentChannelStorageServerKey)) {
		return
	}

	conf.Enabled = true

	err = vip.UnmarshalKey(config.PaymentChannelStorageServerKey, conf)
	return
}

// PaymentChannelStorageClientConf config
// ConnectionTimeout - timeout for failing to establish a connection
// RequestTimeout    - per request timeout
// Endpoints         - cluster endpoints
type PaymentChannelStorageClientConf struct {
	ConnectionTimeout int `mapstructure:"connection_timeout"`
	RequestTimeout    int `mapstructure:"request_timeout"`
	Endpoints         []string
}

// GetPaymentChannelStorageClientConf gets PaymentChannelStorageServerConf from viper
// The DefaultPaymentChannelStorageClientConf is used in case the PAYMENT_CHANNEL_STORAGE_CLIENT field
// is not set in the configuration file
func GetPaymentChannelStorageClientConf(vip *viper.Viper) (conf *PaymentChannelStorageClientConf, err error) {

	conf = &PaymentChannelStorageClientConf{
		ConnectionTimeout: 5000,
		RequestTimeout:    3000,
		Endpoints:         []string{"http://127.0.0.1:2379"},
	}

	if !vip.InConfig(strings.ToLower(config.PaymentChannelStorageClientKey)) {
		return
	}

	err = vip.UnmarshalKey(config.PaymentChannelStorageClientKey, conf)
	return
}

func normalizeDefaultConf(conf string) string {
	return strings.Replace(conf, "_", "", -1)
}
