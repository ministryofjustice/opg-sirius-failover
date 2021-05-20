package main

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/stretchr/testify/assert"
)

type mockAWSSecurityTokenService struct {
	stsiface.STSAPI
	Resp sts.GetCallerIdentityOutput
}

type mockRoute53Service struct {
	route53iface.Route53API
	List   route53.ListHostedZonesByNameOutput
	Change route53.ChangeResourceRecordSetsOutput
}

type mockRDSService struct {
	rdsiface.RDSAPI
	Resp rds.FailoverGlobalClusterOutput
}

func (m mockAWSSecurityTokenService) GetCallerIdentity(in *sts.GetCallerIdentityInput) (*sts.GetCallerIdentityOutput, error) {
	return &m.Resp, nil
}

func (m mockRoute53Service) ListHostedZonesByName(in *route53.ListHostedZonesByNameInput) (*route53.ListHostedZonesByNameOutput, error) {
	return &m.List, nil
}

func (m mockRoute53Service) ChangeResourceRecordSets(in *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error) {
	return &m.Change, nil
}

func (m mockRDSService) FailoverGlobalCluster(in *rds.FailoverGlobalClusterInput) (*rds.FailoverGlobalClusterOutput, error) {
	return &m.Resp, nil
}

func TestSecurityTokenService_GetAccountID(t *testing.T) {
	cases := []struct {
		Resp sts.GetCallerIdentityOutput
	}{
		{
			Resp: sts.GetCallerIdentityOutput{
				Account: aws.String("012345678"),
				Arn:     aws.String("arn:test::value"),
				UserId:  aws.String("12345"),
			},
		},
	}

	for _, tc := range cases {
		s := SecurityTokenService{Client: mockAWSSecurityTokenService{Resp: tc.Resp}}

		resp := s.GetAccountID()
		assert.Equal(t, resp, "012345678")
	}
}

func TestRoute53Service_LookupHostedZone(t *testing.T) {
	cases := []struct {
		Resp route53.ListHostedZonesByNameOutput
	}{
		{
			Resp: route53.ListHostedZonesByNameOutput{
				HostedZones: []*route53.HostedZone{
					{
						CallerReference:        aws.String("Call Ref 1"),
						Name:                   aws.String("test.service.com"),
						Id:                     aws.String("/hostedzone/Z12345678"),
						ResourceRecordSetCount: aws.Int64(2),
					},
				},
			},
		},
	}

	for _, tc := range cases {
		s := Route53Service{Client: mockRoute53Service{List: tc.Resp}}
		resp := s.LookupHostedZone("test.service.com")

		assert.Equal(t, "Z12345678", resp)
	}
}
