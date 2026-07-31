package unittests

import (
	"context"
	"io"
	"jfrog-credential-provider/internal/handlers"
	"jfrog-credential-provider/internal/utils"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
)

type fakeWebIdentitySTS struct {
	call func(context.Context, *sts.AssumeRoleWithWebIdentityInput) (*sts.AssumeRoleWithWebIdentityOutput, error)
}

func (f fakeWebIdentitySTS) AssumeRoleWithWebIdentity(ctx context.Context, input *sts.AssumeRoleWithWebIdentityInput, _ ...func(*sts.Options)) (*sts.AssumeRoleWithWebIdentityOutput, error) {
	return f.call(ctx, input)
}

type fakeAssumeRoleSTS struct {
	call func(context.Context, *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error)
}

func (f fakeAssumeRoleSTS) AssumeRole(ctx context.Context, input *sts.AssumeRoleInput, _ ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	return f.call(ctx, input)
}

func TestAWSWebIdentityFlowWithInjectedSTS(t *testing.T) {
	imds := successfulTokenAndRegionIMDS(t)
	routed := newRoutedService(t, imds, http.NotFoundHandler())
	t.Setenv("aws_region", "")

	stsClient := fakeWebIdentitySTS{call: func(_ context.Context, input *sts.AssumeRoleWithWebIdentityInput) (*sts.AssumeRoleWithWebIdentityOutput, error) {
		if aws.ToString(input.RoleArn) != "arn:aws:iam::123456789012:role/workload" {
			t.Errorf("unexpected role ARN: %q", aws.ToString(input.RoleArn))
		}
		if aws.ToString(input.WebIdentityToken) != "projected-service-account-token" {
			t.Errorf("unexpected web identity token: %q", aws.ToString(input.WebIdentityToken))
		}
		if !strings.HasPrefix(aws.ToString(input.RoleSessionName), "jfrog-credential-provider-") {
			t.Errorf("unexpected role session name: %q", aws.ToString(input.RoleSessionName))
		}
		return &sts.AssumeRoleWithWebIdentityOutput{Credentials: sdkCredentials()}, nil
	}}

	signed, err := handlers.GetAWSSignedRequestWithClients(
		routed.service,
		context.Background(),
		"projected-service-account-token",
		utils.AWSEnvVariables{
			AWSAuthMethod: "web_identity",
			AWSRoleName:   "arn:aws:iam::123456789012:role/workload",
		},
		handlers.AWSSTSClients{WebIdentity: stsClient},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertSignedRequest(t, signed)
}

func TestAWSExternalRoleFlowWithInjectedSTS(t *testing.T) {
	imds := successfulTokenAndRegionIMDS(t)
	routed := newRoutedService(t, imds, http.NotFoundHandler())
	t.Setenv("aws_region", "")

	stsClient := fakeAssumeRoleSTS{call: func(_ context.Context, input *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
		if aws.ToString(input.RoleArn) != "arn:aws:iam::123456789012:role/external" {
			t.Errorf("unexpected role ARN: %q", aws.ToString(input.RoleArn))
		}
		if aws.ToInt32(input.DurationSeconds) != 900 {
			t.Errorf("unexpected duration: %d", aws.ToInt32(input.DurationSeconds))
		}
		return &sts.AssumeRoleOutput{Credentials: sdkCredentials()}, nil
	}}

	signed, err := handlers.GetAWSSignedRequestWithClients(
		routed.service,
		context.Background(),
		"",
		utils.AWSEnvVariables{
			AWSAuthMethod:                  "assume_external_role",
			AWSExternalRoleARN:             "arn:aws:iam::123456789012:role/external",
			AWSExternalRoleDurationSeconds: 900,
		},
		handlers.AWSSTSClients{AssumeRole: stsClient},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertSignedRequest(t, signed)
}

func successfulTokenAndRegionIMDS(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest/api/token":
			_, _ = io.WriteString(w, "imds-token")
		case "/latest/meta-data/placement/region":
			assertIMDSToken(t, r)
			_, _ = io.WriteString(w, "us-east-1")
		default:
			http.NotFound(w, r)
		}
	})
}

func sdkCredentials() *ststypes.Credentials {
	expiration := time.Now().Add(time.Hour)
	return &ststypes.Credentials{
		AccessKeyId:     aws.String(testAccessKey),
		SecretAccessKey: aws.String(testSecretKey),
		SessionToken:    aws.String(testSession),
		Expiration:      &expiration,
	}
}

func assertSignedRequest(t *testing.T, request *http.Request) {
	t.Helper()
	if request.URL.Host != "sts.us-east-1.amazonaws.com" {
		t.Errorf("unexpected signed STS host: %q", request.URL.Host)
	}
	if got := request.Header.Get("X-Amz-Region-Set"); got != "us-east-1" {
		t.Errorf("unexpected signed region: %q", got)
	}
	if got := request.Header.Get("X-Amz-Security-Token"); got != testSession {
		t.Errorf("unexpected signed session token: %q", got)
	}
	if got := request.Header.Get("Authorization"); !strings.HasPrefix(got, "AWS4-ECDSA-P256-SHA256 ") {
		t.Errorf("missing SigV4a authorization: %q", got)
	}
}
