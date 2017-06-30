package main

import (
	"cloud.google.com/go/storage"
	"flag"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	"io"
	"log"
	"os"
	"strings"
	"path"
)

// Use: go run gcs-deploy.go -f geth-classic-linux-14.0.zip -key ./.gcloud.key

// write treats "object" and "file" as same
func write(client *storage.Client, bucket, object string) error {
	ctx := context.Background()
	// [START upload_file]
	f, err := os.Open(object)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write object to storage, ensuring basename for file/object if exists.
	wc := client.Bucket(bucket).Object(path.Base(object)).NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	// [END upload_file]
	return nil
}

func main() {

	var k, s string
	flag.StringVar(&s, "f", "", "source object; in the format of <bucket:object>")
	flag.StringVar(&k, "key", "", "specify a service account json key file")
	flag.Parse()

	names := strings.Split(s, ":")
	if len(names) < 2 {
		log.Fatal("missing -f flag.\n use: $ go run gc-deploy.go -f <bucket:object>  <- [bucket:object to write to gcs]")
	}
	bucket, object := names[0], names[1]
	if _, e := os.Stat(object); e != nil {
		log.Fatalf("no object or file by name: %s", object)
	}

	// Ensure key file exists.
	if _, e := os.Stat(k); e != nil {
		log.Fatal(k)
	}

	ctx := context.Background()

	client, err := storage.NewClient(ctx, option.WithServiceAccountFile(k))
	if err != nil {
		log.Fatal(err)
	}

	e := write(client, bucket, object)
	if e != nil {
		log.Fatal(e)
	}
}
