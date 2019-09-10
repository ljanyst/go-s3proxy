S3Proxy
=======

S3Proxy, as the name suggests, is a proxy server that allows you to access your
authenticated S3 resources using https and HTTP BasicAuth. It prefetches a 1MB
ahead of every new read that does not match the currently pre-fetched data chunk
and position.

You can configure the HTTP passwords globally using a tool like Apache's
`htpasswd`:

    ]==> htpasswd -c -B passwords.txt user1
    ]==> htpasswd -B passwords.txt user2

Where: `-c` stands for create, `-B` stands for bcrypt.

You can also configure as many buckets as you want. While accessing the data,
the first component of the path is assumed to be the bucket name, while the
remaining components are assumed to constitute a key within that bucket.

Here's an example config file:

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
    "test-bucket-2": {
      "Region": "eu-west-1",
      "Key": "asdf",
      "Secret": "foo"
    },
  }
}
```
