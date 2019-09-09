//------------------------------------------------------------------------------
// Author: Lukasz Janyst <lukasz@jany.st>
// Date: 07.09.2019
//
// Licensed under the BSD License, see the LICENSE file for details.
//------------------------------------------------------------------------------

package s3proxy

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/foomo/htpasswd"
	"github.com/ljanyst/go-srvutils/auth"
	log "github.com/sirupsen/logrus"
)

func RunWebServer(opts *S3ProxyOpts) {
	s3Fs := http.FileServer(NewS3Fs(opts.Buckets))

	if opts.Web.EnableAuth {
		authFile := opts.Web.HtpasswdFile
		passwords, err := htpasswd.ParseHtpasswdFile(authFile)
		if err != nil {
			log.Fatalf(`Authentication enabled but cannot open htpassword file "%s": %s`,
				authFile, err)
		}

		log.Infof("Loaded authentication data from: %s", authFile)

		http.Handle("/", auth.NewBasicAuthHandler("s3proxy", passwords, s3Fs))

	} else {
		http.Handle("/", s3Fs)
	}

	var wg sync.WaitGroup
	wg.Add(len(opts.Web.BindAddresses))

	for _, addr := range opts.Web.BindAddresses {
		go func() {
			protocol := "http"
			if addr.IsHttps {
				protocol = "https"
			}
			log.Infof("Listening on %s://%s:%d", protocol, addr.Host, addr.Port)
			addressString := fmt.Sprintf("%s:%d", addr.Host, addr.Port)
			if addr.IsHttps {
				log.Fatal("Server failure: ",
					http.ListenAndServeTLS(addressString, opts.Web.Https.Cert, opts.Web.Https.Key, nil))
			} else {
				log.Fatal("Server failure: ", http.ListenAndServe(addressString, nil))
			}
			wg.Done()
		}()
	}

	wg.Wait()
}
