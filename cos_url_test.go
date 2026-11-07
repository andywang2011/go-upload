package upload

import "testing"

func TestCosBucketURL_baseURL(t *testing.T) {
	u, err := cosBucketURL(COSConfig{
		BaseURL: "https://custom.example/bucket",
	})
	if err != nil || u.String() != "https://custom.example/bucket" {
		t.Fatal(err, u)
	}
}

func TestCosBucketURL_invalid(t *testing.T) {
	_, err := cosBucketURL(COSConfig{BaseURL: "://bad"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCosBucketURL_withAppID(t *testing.T) {
	u, err := cosBucketURL(COSConfig{
		SecretID: "a", SecretKey: "b", Bucket: "mybucket", AppID: "1250000000", Region: "ap-shanghai",
	})
	if err != nil || u.Host == "" {
		t.Fatal(err, u)
	}
}
