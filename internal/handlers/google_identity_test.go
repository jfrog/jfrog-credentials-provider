package handlers

import "testing"

func TestGoogleMetadataIdentityURL(t *testing.T) {
	got := googleMetadataIdentityURL("sts.amazonaws.com")
	want := "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity?audience=sts.amazonaws.com&format=full"
	if got != want {
		t.Fatalf("url = %s, want %s", got, want)
	}
}
