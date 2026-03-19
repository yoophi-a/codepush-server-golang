package minio

import (
	"context"
	"errors"
	"io"
	"testing"

	miniolib "github.com/minio/minio-go/v7"
)

type fakeClient struct {
	bucketExistsResult bool
	bucketExistsErr    error
	putObjectErr       error
	removeObjectErr    error
	lastBucket         string
	lastKey            string
}

func (f *fakeClient) BucketExists(context.Context, string) (bool, error) {
	return f.bucketExistsResult, f.bucketExistsErr
}

func (f *fakeClient) PutObject(_ context.Context, bucket, key string, _ io.Reader, _ int64, _ miniolib.PutObjectOptions) (miniolib.UploadInfo, error) {
	f.lastBucket = bucket
	f.lastKey = key
	return miniolib.UploadInfo{}, f.putObjectErr
}

func (f *fakeClient) RemoveObject(_ context.Context, bucket, key string, _ miniolib.RemoveObjectOptions) error {
	f.lastBucket = bucket
	f.lastKey = key
	return f.removeObjectErr
}

func TestStorageMethods(t *testing.T) {
	client := &fakeClient{bucketExistsResult: true}
	store := &Storage{client: client, bucket: "codepush"}

	if err := store.CheckHealth(context.Background()); err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}

	uri, err := store.PutObject(context.Background(), "artifacts/demo.txt", []byte("hello"), "text/plain")
	if err != nil {
		t.Fatalf("PutObject() error = %v", err)
	}
	if uri != "minio://codepush/artifacts/demo.txt" {
		t.Fatalf("unexpected URI %q", uri)
	}
	if client.lastBucket != "codepush" || client.lastKey != "artifacts/demo.txt" {
		t.Fatalf("unexpected put object target %q/%q", client.lastBucket, client.lastKey)
	}

	if err := store.DeleteObject(context.Background(), "artifacts/demo.txt"); err != nil {
		t.Fatalf("DeleteObject() error = %v", err)
	}
}

func TestStoragePropagatesErrors(t *testing.T) {
	store := &Storage{client: &fakeClient{bucketExistsErr: errors.New("boom")}, bucket: "codepush"}
	if err := store.CheckHealth(context.Background()); err == nil {
		t.Fatalf("expected health error")
	}

	store = &Storage{client: &fakeClient{putObjectErr: errors.New("boom")}, bucket: "codepush"}
	if _, err := store.PutObject(context.Background(), "artifacts/demo.txt", []byte("hello"), "text/plain"); err == nil {
		t.Fatalf("expected put object error")
	}

	store = &Storage{client: &fakeClient{removeObjectErr: errors.New("boom")}, bucket: "codepush"}
	if err := store.DeleteObject(context.Background(), "artifacts/demo.txt"); err == nil {
		t.Fatalf("expected delete object error")
	}
}
