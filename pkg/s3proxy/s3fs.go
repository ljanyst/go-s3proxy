//------------------------------------------------------------------------------
// Author: Lukasz Janyst <lukasz@jany.st>
// Date: 07.09.2019
//
// Licensed under the BSD License, see the LICENSE file for details.
//------------------------------------------------------------------------------

package s3proxy

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	ljfs "github.com/ljanyst/go-srvutils/fs"
	log "github.com/sirupsen/logrus"
)

const listingTemplateText = `
<!DOCTYPE html>
<html>
<head>
<title>{{.BucketName}}</title>
<style>
table {
  font-family: arial, sans-serif;
  border-collapse: collapse;
  width: 100%;
  padding: 0;
  margin: 0;
  border: 1px solid #ddd;
}

td {
  padding-top: 0.1em;
  padding-bottom: 0.1em;
  padding-left: 0.5em;
  padding-right: 0.5em;
}

tr:nth-child(even) {
  background-color: #dddddd;
}
td.name {
  text-align: left;
}

td.md {
  text-align: right;
}
</style>
</head>
<body>

<h2>{{.BucketName}}</h2>

<table>
  {{range .Objects}}
    <tr>
      <td class="name"><a href="/{{$.MountName}}/{{.Key}}">{{.Key}}</a></td>
      <td class="md">{{.LastModified}}</td>
    </tr>
  {{end}}
</table>
</body>
</html>
`

var listingTemplate *template.Template

func init() {
	var err error
	listingTemplate, err = template.New("listing").Parse(listingTemplateText)
	if err != nil {
		log.Fatalf("Cannot parse listing template: %s", err)
	}
}

type ListingItem struct {
	Key          string
	LastModified time.Time
}

type Listing struct {
	BucketName string
	MountName  string
	Objects    []ListingItem
}

const CHUNK_SIZE = 1024 * 1024 * 1024

type S3Fs struct {
	clients map[string]*s3.S3
	buckets map[string]string
}

type S3Chunk struct {
	offset int64
	end    int64
	body   io.ReadCloser
}

type S3File struct {
	client *s3.S3
	bucket string
	key    string
	size   int64
	offset int64
	mod    time.Time
	chunk  S3Chunk
}

func NewS3File(cl *s3.S3, bucket, key string) (*S3File, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := cl.HeadObject(input)
	if err != nil {
		log.Errorf("Cannot head object %q from bucket %q: %s", key, bucket, err)
		return nil, os.ErrInvalid
	}

	return &S3File{
		cl,
		bucket,
		key,
		*result.ContentLength,
		0,
		*result.LastModified,
		S3Chunk{0, 0, nil},
	}, nil

}

func (f *S3File) fetchChunk(start int64) error {
	if f.chunk.body != nil {
		f.chunk.body.Close()
		f.chunk.body = nil
	}

	byteRange := fmt.Sprintf("bytes=%d-%d", start, start+CHUNK_SIZE-1)
	input := &s3.GetObjectInput{
		Bucket: aws.String(f.bucket),
		Key:    aws.String(f.key),
		Range:  aws.String(byteRange),
	}

	result, err := f.client.GetObject(input)
	if err != nil {
		return err
	}

	f.chunk = S3Chunk{start, start + CHUNK_SIZE, result.Body}
	return nil
}

func (f *S3File) Read(p []byte) (n int, err error) {
	if f.offset >= f.size {
		return 0, io.EOF
	}

	toRead := len(p)
	bufferOffset := 0

	if f.offset+int64(toRead) > f.size {
		toRead = int(f.size - f.offset)
	}

	if f.offset != f.chunk.offset || f.offset+int64(toRead) > f.chunk.end {
		if err := f.fetchChunk(f.offset); err != nil {
			return 0, err
		}
	}

	for toRead > 0 {
		read, err := f.chunk.body.Read(p[bufferOffset:])
		if err != nil && err != io.EOF {
			log.Errorf("Read error: %v", err)
			return bufferOffset, err
		}
		toRead -= read
		f.offset += int64(read)
		f.chunk.offset += int64(read)
		bufferOffset += read
	}

	return bufferOffset, nil
}

func (f *S3File) Seek(offset int64, whence int) (int64, error) {
	var off int64
	switch whence {
	case io.SeekStart:
		off = offset
	case io.SeekCurrent:
		off = f.offset + offset
	case io.SeekEnd:
		off = f.size + offset
	default:
		return 0, os.ErrInvalid
	}

	if off < 0 || off > f.size {
		return 0, os.ErrInvalid
	}
	f.offset = off

	return f.offset, nil
}

func (f *S3File) Close() error {
	if f.chunk.body != nil {
		return f.chunk.body.Close()
	}
	f.client = nil
	return nil
}

func (f *S3File) Readdir(count int) ([]os.FileInfo, error) {
	return nil, fmt.Errorf("Cannot Readdir from a file")
}

func (f *S3File) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *S3File) IsDir() bool {
	return false
}

func (f *S3File) ModTime() time.Time {
	return f.mod
}

func (f *S3File) Mode() os.FileMode {
	return 0444
}

func (f *S3File) Size() int64 {
	return f.size
}

func (f *S3File) Name() string {
	return filepath.Base(f.key)
}

func (f *S3File) Sys() interface{} {
	return nil
}

func NewS3Fs(bucketOpts map[string]BucketOpts) http.FileSystem {
	fs := new(S3Fs)
	fs.clients = make(map[string]*s3.S3)
	fs.buckets = make(map[string]string)

	for k, v := range bucketOpts {
		region := v.Region
		if region == "" {
			region = "us-west-1"
		}
		bucket := k
		if v.Bucket != "" {
			bucket = v.Bucket
		}

		sess, err := session.NewSession(&aws.Config{
			Region:      aws.String(region),
			Credentials: credentials.NewStaticCredentials(v.Key, v.Secret, ""),
		})
		if err != nil {
			log.Errorf("Unable to initialize session for %s: %s", k, err)
			continue
		}
		fs.clients[bucket] = s3.New(sess)
		fs.buckets[k] = bucket
	}
	return fs
}

func (fs S3Fs) getListing(bucket, mount string) (http.File, error) {
	cl, ok := fs.clients[bucket]
	if !ok {
		return nil, os.ErrNotExist
	}

	if cl == nil {
		return nil, os.ErrNotExist
	}

	input := &s3.ListObjectsInput{
		Bucket:  aws.String(bucket),
		MaxKeys: aws.Int64(100000),
	}

	result, err := cl.ListObjects(input)
	if err != nil {
		log.Errorf("Cannot list bucket %q: %s", bucket, err)
		return nil, os.ErrInvalid
	}

	var l Listing
	l.BucketName = bucket
	l.MountName = mount

	for _, item := range result.Contents {
		l.Objects = append(l.Objects, ListingItem{*item.Key, *item.LastModified})
	}

	var b bytes.Buffer
	if err := listingTemplate.Execute(&b, l); err != nil {
		log.Errorf("Cannot execute listing template for bucket %q: %s", bucket, err)
		return nil, os.ErrInvalid
	}

	return ljfs.VirtualFile{
		bucket,
		int64(b.Len()),
		false,
		bytes.NewReader(b.Bytes()),
	}, nil
}

func (fs S3Fs) getObject(bucket, key string) (http.File, error) {
	cl, ok := fs.clients[bucket]
	if !ok {
		return nil, os.ErrNotExist
	}

	if cl == nil {
		return nil, os.ErrNotExist
	}

	return NewS3File(cl, bucket, key)
}

func (fs S3Fs) Open(filePath string) (http.File, error) {
	cleaned := path.Clean(filePath)
	components := strings.Split(cleaned[1:], "/")
	mount := components[0]
	key := ""
	if len(components) > 1 {
		key = "/" + path.Join(components[1:]...)
	}

	bucket, ok := fs.buckets[mount]
	if !ok {
		return nil, os.ErrNotExist
	}

	if key == "" {
		return fs.getListing(bucket, mount)
	}

	return fs.getObject(bucket, key)
}
