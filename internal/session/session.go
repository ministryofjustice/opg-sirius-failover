package session

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
)

type Session struct {
	AwsSession *session.Session
}

func NewSession(region string) (*Session, error) {

	sess, err := session.NewSession(&aws.Config{Region: &region})
	if err != nil {
		return nil, err
	}

	if iamRole, ok := os.LookupEnv("AWS_IAM_ROLE"); ok {
		c := stscreds.NewCredentials(sess, iamRole)
		*sess.Config.Credentials = *c
	}

	return &Session{sess}, nil
}
