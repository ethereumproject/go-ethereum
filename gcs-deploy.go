package main

import (
	"cloud.google.com/go/storage"
	"flag"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	"io"
	"log"
	"os"
)

// Use: go run gcs-deploy.go -bucket builds.etcdevteam.com -object go-ethereum/$(cat version-base.txt)/geth-classic-$TRAVIS_OS_NAME-$(cat version-app.txt).zip -file geth-classic-linux-14.0.zip -key ./.gcloud.key

// write treats "object" and "file" as same
func write(client *storage.Client, bucket, object, file string) error {
	ctx := context.Background()
	// [START upload_file]
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write object to storage, ensuring basename for file/object if exists.
	wc := client.Bucket(bucket).Object(object).NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	// [END upload_file]
	log.Printf(`Successfully uploaded:
	bucket: %v
	object: %v
	file: %v`, bucket, object, file)
	return nil
}

func main() {

	var key, file, bucket, object string
	flag.StringVar(&bucket, "bucket", "", "gcp bucket name")
	flag.StringVar(&object, "object", "", "gcp object path")
	flag.StringVar(&file, "file", "", "file to upload")
	flag.StringVar(&key, "key", "", "service account json key file")
	flag.Parse()

	if _, e := os.Stat(file); e != nil {
		log.Fatal(file, e)
	}

	// Ensure key file exists.
	if _, e := os.Stat(key); e != nil {
		log.Fatal(file, e)
	}

	ctx := context.Background()

	client, err := storage.NewClient(ctx, option.WithServiceAccountFile(key))
	if err != nil {
		log.Fatal(err)
	}

	e := write(client, bucket, object, file)
	if e != nil {
		log.Fatal(e)
	}
}
