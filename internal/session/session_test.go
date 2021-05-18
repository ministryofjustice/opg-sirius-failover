package session

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/stretchr/testify/assert"
)

func TestNewSession(t *testing.T) {
	testCases := []struct {
		region     string
		role       *string
		wantRegion string
		wantErr    bool
	}{
		{"eu-west-1", nil, "eu-west-1", false},
		{"", nil, "eu-west-1", false},
		{"us-west-1", nil, "us-west-1", false},
		{"", aws.String("some-iam-role"), "eu-west-1", false},
	}

	for _, tc := range testCases {
		os.Unsetenv("AWS_REGION")
		os.Unsetenv("AWS_IAM_ROLE")

		if tc.region != "" {
			os.Setenv("AWS_REGION", tc.region)
		}
		if tc.role != nil {
			os.Setenv("AWS_IAM_ROLE", *tc.role)
		}

		got, err := NewSession(tc.region)
		if tc.wantErr {
			assert.Error(t, err)
		} else {
			assert.Nil(t, err)
		}
		assert.IsType(t, new(session.Session), got.AwsSession)
		assert.Equal(t, tc.wantRegion, *got.AwsSession.Config.Region)
	}
}
