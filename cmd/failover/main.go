package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/ministryofjustice/opg-aws-failover-cli/internal/session"

	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/manifoldco/promptui"
)

var environment string

type Params struct {
	Environment         string
	AccountID           string
	HostedZone          string
	RecordName          string
	LBHostedZoneLondon  string
	LBDnsLondon         string
	LBHostedZoneIreland string
	LBDnsIreland        string
	GlobalCluster       string
	TargetCluster       string
}

type SecurityTokenService struct {
	Client stsiface.STSAPI
}

type Route53Service struct {
	Client route53iface.Route53API
}

type RDSService struct {
	Client rdsiface.RDSAPI
}

func main() {
	flag.Usage = func() {
		fmt.Println("Usage: failover -env <environment>")
		flag.PrintDefaults()
	}

	flag.String("help", "", "this help information")
	flag.StringVar(&environment, "env", "", "environment to failover")
	flag.Parse()

	if environment == "" {
		fmt.Println("Environment not set")
		flag.Usage()
	}

	if environment == "production" {
		c := yesNo()
		if !c {
			fmt.Println("You confirmed you didn't want to failover production.")
			flag.Usage()
			return
		}
	}

	sess, err := session.NewSession("eu-west-1")
	if err != nil {
		fmt.Println(err)
	}

	params := getRequiredParms(sess, environment)

	fmt.Print(params)
	// failoverDNS(sess, params)
	// failoverCluster(sess, p.Environment+"-membrane-global", "arn:aws:rds:eu-west-2:"+p.AccountID+":cluster:membrane-"+p.Environment)
	// failoverCluster(sess, p.Environment+"-api-global", "arn:aws:rds:eu-west-2:"+p.AccountID+":cluster:api-"+p.Environment)
}

func yesNo() bool {
	prompt := promptui.Select{
		Label: "Select[Yes/No]",
		Items: []string{"Yes", "No"},
	}
	_, result, err := prompt.Run()
	if err != nil {
		log.Fatalf("Prompt failed %v\n", err)
	}
	return result == "Yes"
}

func getRequiredParms(s *session.Session, environment string) Params {
	token := SecurityTokenService{
		Client: sts.New(s.AwsSession),
	}

	p := Params{
		Environment: environment,
		AccountID:   token.GetAccountID(),
	}

	if p.Environment == "production" {
		p.RecordName = "sirius-opg.uk"
	} else {
		p.RecordName = "preproduction.sirius.opg.digital"
	}

	r53 := Route53Service{
		Client: route53.New(s.AwsSession),
	}
	r53.LookupHostedZone(p.RecordName)

	dnsl, hzl := lookupLoadBalancer(p.Environment, "eu-west-2")
	p.LBHostedZoneLondon = dnsl
	p.LBDnsLondon = hzl

	dnsi, hzi := lookupLoadBalancer(p.Environment, "eu-west-1")
	p.LBHostedZoneIreland = dnsi
	p.LBDnsIreland = hzi

	// Once we have the correct hosted zone, override the RecordName with the record for production
	if p.Environment == "production" {
		p.RecordName = "live.sirius-opg.uk"
	}

	return p
}

func (s *SecurityTokenService) GetAccountID() string {

	res, err := s.Client.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		fmt.Println(err)
	}

	return *res.Account
}

func (r *Route53Service) LookupHostedZone(name string) string {
	res, err := r.Client.ListHostedZonesByName(&route53.ListHostedZonesByNameInput{
		DNSName:  aws.String(name),
		MaxItems: aws.String("1"),
	})

	if err != nil {
		fmt.Println(err)
	}

	// Get the single zone returned
	// just get the Zone ID, we dont need the /hostedzone/ prefix
	if len(res.HostedZones) == 1 {
		z := res.HostedZones[0]
		id := strings.SplitAfter(*z.Id, "/hostedzone/")
		return id[0]
	}
	return "Did not get hosted zone"
}

func (r *Route53Service) FailoverDNS(p Params) {

	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{ // Required
			Changes: []*route53.Change{ // Required
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name:          aws.String(p.RecordName),
						Type:          aws.String("A"),
						Weight:        aws.Int64(100),
						SetIdentifier: aws.String("eu-west-2"),
						AliasTarget: &route53.AliasTarget{
							HostedZoneId:         aws.String(p.LBHostedZoneLondon),
							DNSName:              aws.String(p.LBDnsLondon),
							EvaluateTargetHealth: aws.Bool(false),
						},
					},
				},
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name:          aws.String(p.RecordName),
						Type:          aws.String("A"),
						Weight:        aws.Int64(0),
						SetIdentifier: aws.String("eu-west-1"),
						AliasTarget: &route53.AliasTarget{
							HostedZoneId:         aws.String(p.LBHostedZoneIreland),
							DNSName:              aws.String(p.LBDnsIreland),
							EvaluateTargetHealth: aws.Bool(false),
						},
					},
				},
			},
			Comment: aws.String("Failing over to london"),
		},
		HostedZoneId: aws.String(p.HostedZone),
	}

	res, err := r.Client.ChangeResourceRecordSets(params)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("%s has started to failover", *res.ChangeInfo.Id)
}

// Failover a global cluster
func (r *RDSService) FailoverCluster(g string, t string) {
	_, err := r.Client.FailoverGlobalCluster(&rds.FailoverGlobalClusterInput{
		GlobalClusterIdentifier:   aws.String(g),
		TargetDbClusterIdentifier: aws.String(t),
	})

	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("%s has started to failover", g)
}

func lookupLoadBalancer(name string, region string) (string, string) {
	sess, err := session.NewSession(region)
	if err != nil {
		fmt.Println(err)
	}

	svc := elbv2.New(sess.AwsSession)
	res, err := svc.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{
		Names: []*string{
			aws.String(name),
		},
	})

	if err != nil {
		fmt.Println(err)
	}

	fmt.Print(res)

	lb := res.LoadBalancers[0]

	return *lb.CanonicalHostedZoneId, *lb.DNSName
}
