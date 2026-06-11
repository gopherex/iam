package notifications

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAwsSNSSignV4(t *testing.T) {
	c := &smsConfig{
		Type:            "aws_sns",
		Region:          "ru-central1",
		AccessKeyID:     "AKIDEXAMPLE",
		SecretAccessKey: "secretkey",
		Endpoint:        "https://notifications.yandexcloud.net",
	}
	body := "Action=Publish&Message=hi&PhoneNumber=%2B79991112233&Version=2010-03-31"
	at := time.Date(2026, 6, 11, 1, 2, 3, 0, time.UTC)

	mk := func() *http.Request {
		r, err := http.NewRequest(http.MethodPost, c.Endpoint, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
		return r
	}

	r1 := mk()
	if err := c.signV4(r1, []byte(body), at); err != nil {
		t.Fatal(err)
	}
	auth := r1.Header.Get("Authorization")
	wantCred := "Credential=AKIDEXAMPLE/20260611/ru-central1/sns/aws4_request"
	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 ") || !strings.Contains(auth, wantCred) {
		t.Fatalf("authorization = %q, want SigV4 with %s", auth, wantCred)
	}
	if !strings.Contains(auth, "SignedHeaders=content-type;host;x-amz-date") {
		t.Fatalf("authorization missing signed headers: %q", auth)
	}
	if r1.Header.Get("X-Amz-Date") != "20260611T010203Z" {
		t.Fatalf("x-amz-date = %q", r1.Header.Get("X-Amz-Date"))
	}

	// Deterministic: same inputs => same signature.
	r2 := mk()
	if err := c.signV4(r2, []byte(body), at); err != nil {
		t.Fatal(err)
	}
	if r2.Header.Get("Authorization") != auth {
		t.Fatal("signV4 is not deterministic for identical inputs")
	}

	// A different secret must change the signature.
	c2 := *c
	c2.SecretAccessKey = "othersecret"
	r3 := mk()
	if err := c2.signV4(r3, []byte(body), at); err != nil {
		t.Fatal(err)
	}
	if r3.Header.Get("Authorization") == auth {
		t.Fatal("signature did not change with a different secret")
	}
}
