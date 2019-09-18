S3Proxy
=======

S3Proxy, as the name suggests, is a proxy server that allows you to access your
authenticated S3 resources using HTTPS and HTTP BasicAuth. It prefetches a 1MB
ahead of every new read that does not match the currently pre-fetched data chunk
and position. This works nicely with sequential downloads and with video
streaming.

You can configure the HTTP passwords globally using a tool like Apache's
`htpasswd`:

    ]==> htpasswd -c -B passwords.txt user1
    ]==> htpasswd -B passwords.txt user2

Where: `-c` stands for create, `-B` stands for bcrypt.

You can also configure as many buckets as you want. While accessing the data,
the first component of the path is assumed to be the mount point name, while the
remaining components are assumed to constitute a key within the bucket that the
mount point is associated with.

Below is an example config file. In it, you can see a map of mount points to
buckets. If the bucket name in the map item is omited, it is assumed to be
the same as the mount point name. If the region is omited, it is asumed to be
`us-west-1`.

```json
{
  "Web": {
    "BindAddresses": [
      {
        "Host": "127.0.0.1",
        "Port": 7649,
        "IsHttps": true,
      }
    ],
    "Https": {
      "Cert": "cert.pem",
      "Key": "key.pem"
    },
    "EnableAuth": true,
    "HtpasswdFile": "passords.txt",
  },
  "Buckets": {
    "test-bucket-1": {
      "Region": "eu-west-1",
      "Key": "asdf",
      "Secret": "foo"
    },
    "data": {
      "Bucket": "test-bucket-2",
      "Region": "eu-west-1",
      "Key": "asdf",
      "Secret": "foo"
    },
  }
}
```
