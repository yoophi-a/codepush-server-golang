package s3

import (
	"context"
	"errors"
	"testing"

	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type fakeClient struct {
	headErr   error
	putErr    error
	deleteErr error
	lastKey   string
}

func (f *fakeClient) HeadBucket(context.Context, *awss3.HeadBucketInput, ...func(*awss3.Options)) (*awss3.HeadBucketOutput, error) {
	return &awss3.HeadBucketOutput{}, f.headErr
}

func (f *fakeClient) PutObject(_ context.Context, input *awss3.PutObjectInput, _ ...func(*awss3.Options)) (*awss3.PutObjectOutput, error) {
	if input.Key != nil {
		f.lastKey = *input.Key
	}
	return &awss3.PutObjectOutput{}, f.putErr
}

func (f *fakeClient) DeleteObject(_ context.Context, input *awss3.DeleteObjectInput, _ ...func(*awss3.Options)) (*awss3.DeleteObjectOutput, error) {
	if input.Key != nil {
		f.lastKey = *input.Key
	}
	return &awss3.DeleteObjectOutput{}, f.deleteErr
}

func TestStorageMethods(t *testing.T) {
	client := &fakeClient{}
	store := &Storage{client: client, bucket: "codepush"}

	if err := store.CheckHealth(context.Background()); err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}
	uri, err := store.PutObject(context.Background(), "artifacts/demo.txt", []byte("hello"), "text/plain")
	if err != nil {
		t.Fatalf("PutObject() error = %v", err)
	}
	if uri != "s3://codepush/artifacts/demo.txt" {
		t.Fatalf("unexpected URI %q", uri)
	}
	if client.lastKey != "artifacts/demo.txt" {
		t.Fatalf("unexpected put key %q", client.lastKey)
	}
	if err := store.DeleteObject(context.Background(), "artifacts/demo.txt"); err != nil {
		t.Fatalf("DeleteObject() error = %v", err)
	}
}

func TestStoragePropagatesErrors(t *testing.T) {
	store := &Storage{client: &fakeClient{headErr: errors.New("boom")}, bucket: "codepush"}
	if err := store.CheckHealth(context.Background()); err == nil {
		t.Fatalf("expected health error")
	}

	store = &Storage{client: &fakeClient{putErr: errors.New("boom")}, bucket: "codepush"}
	if _, err := store.PutObject(context.Background(), "artifacts/demo.txt", []byte("hello"), "text/plain"); err == nil {
		t.Fatalf("expected put error")
	}

	store = &Storage{client: &fakeClient{deleteErr: errors.New("boom")}, bucket: "codepush"}
	if err := store.DeleteObject(context.Background(), "artifacts/demo.txt"); err == nil {
		t.Fatalf("expected delete error")
	}
}

func TestNew(t *testing.T) {
	store, err := New(context.Background(), "us-east-1", "http://localhost:9000", "key", "secret", "codepush", true)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if store == nil || store.bucket != "codepush" || store.client == nil {
		t.Fatalf("unexpected store %#v", store)
	}
}
