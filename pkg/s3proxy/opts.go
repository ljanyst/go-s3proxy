//------------------------------------------------------------------------------
// Author: Lukasz Janyst <lukasz@jany.st>
// Date: 07.09.2019
//
// Licensed under the BSD License, see the LICENSE file for details.
//------------------------------------------------------------------------------

package s3proxy

import (
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
)

type BindAddress struct {
	Host    string // Either host name or an IP address (IPv4 or IPv6)
	Port    int
	IsHttps bool
}

type HttpsOpts struct {
	Cert string // Certificate file (mandatory)
	Key  string // Key file (mandatory)
}

type WebOpts struct {
	BindAddresses []BindAddress // List of addresses the server should bind to
	Https         HttpsOpts     // Https configuration
	EnableAuth    bool          // Enable basic HTTP auth
	HtpasswdFile  string        // path to the htpasswd file
}

type BucketOpts struct {
	Bucket string // Bucket name, if not specified, it's the same as the key
	Region string // AWS region
	Key    string // AWS key
	Secret string // AWS secret
}

type S3ProxyOpts struct {
	Web     WebOpts               // Web server configuration
	Buckets map[string]BucketOpts // Bucket configuration
}

// Create a S3ProxyOpts object with default settings filled in
func NewS3ProxyOpts() (opts *S3ProxyOpts) {
	opts = new(S3ProxyOpts)
	opts.Web.BindAddresses = []BindAddress{BindAddress{"localhost", 7649, false}}
	opts.Web.EnableAuth = false
	return
}

// Load the configuration data from a Yaml file
func (opts *S3ProxyOpts) LoadYaml(fileName string) error {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("Unable to read the configuration file %s: %s", fileName, err)
	}

	err = yaml.Unmarshal(data, opts)
	if err != nil {
		return fmt.Errorf("Malformed config %s: %s", fileName, err)
	}

	return nil
}
