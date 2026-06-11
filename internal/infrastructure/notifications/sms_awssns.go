package notifications

// AWS SNS (and SNS-compatible, e.g. Yandex Cloud Notifications) SMS sender.
// Sends an SNS Publish with a PhoneNumber + Message, authenticated with AWS
// Signature V4 over static service-account access keys. Yandex is supported by
// pointing Endpoint at https://notifications.yandexcloud.net with its region.
//
// SigV4 is implemented inline (no aws-sdk dependency) for the single Publish
// call: canonical request -> string-to-sign -> HMAC signing-key chain.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	snsService    = "sns"
	snsAPIVersion = "2010-03-31"
)

// sendAwsSNS publishes an SMS via SNS Publish, SigV4-signed.
func (c *smsConfig) sendAwsSNS(ctx context.Context, to, text string) error {
	form := url.Values{}
	form.Set("Action", "Publish")
	form.Set("Version", snsAPIVersion)
	form.Set("PhoneNumber", to)
	form.Set("Message", text)
	// Sender ID (alphanumeric "from"), when configured, as an SMS message attribute.
	if c.From != "" {
		form.Set("MessageAttributes.entry.1.Name", "AWS.SNS.SMS.SenderID")
		form.Set("MessageAttributes.entry.1.Value.DataType", "String")
		form.Set("MessageAttributes.entry.1.Value.StringValue", c.From)
	}
	body := form.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	if err := c.signV4(req, []byte(body), time.Now().UTC()); err != nil {
		return err
	}
	return c.do(req)
}

// signV4 adds the AWS Signature V4 Authorization header to req for the SNS
// service in c.Region using c.AccessKeyID / c.SecretAccessKey.
func (c *smsConfig) signV4(req *http.Request, payload []byte, now time.Time) error {
	host := req.URL.Host
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	req.Header.Set("Host", host)
	req.Header.Set("X-Amz-Date", amzDate)

	payloadHash := hexSHA256(payload)

	// Canonical headers (sorted): content-type, host, x-amz-date.
	contentType := req.Header.Get("Content-Type")
	canonicalHeaders := "content-type:" + contentType + "\n" +
		"host:" + host + "\n" +
		"x-amz-date:" + amzDate + "\n"
	signedHeaders := "content-type;host;x-amz-date"

	canonicalURI := req.URL.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	canonicalRequest := strings.Join([]string{
		http.MethodPost,
		canonicalURI,
		req.URL.RawQuery, // empty for POST-body form
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	scope := strings.Join([]string{dateStamp, c.Region, snsService, "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		hexSHA256([]byte(canonicalRequest)),
	}, "\n")

	signingKey := sigV4SigningKey(c.SecretAccessKey, dateStamp, c.Region, snsService)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	auth := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.AccessKeyID, scope, signedHeaders, signature,
	)
	req.Header.Set("Authorization", auth)
	return nil
}

func sigV4SigningKey(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func hexSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
