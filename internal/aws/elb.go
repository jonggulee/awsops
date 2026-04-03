package aws

import (
	"context"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

type LoadBalancer struct {
	Profile          string
	Region           string
	Name             string
	DNSName          string // 매칭 키: Route53 alias target과 비교
	ARN              string
	LBType           string // "application" / "network" / "gateway"
	Scheme           string // "internet-facing" / "internal"
	State            string
	VpcID            string
	AvailabilityZones []string
	SecurityGroupIDs  []string
}

// DNSNameNorm returns the DNS name in lowercase without trailing dot (비교용).
func (lb LoadBalancer) DNSNameNorm() string {
	return strings.ToLower(strings.TrimSuffix(lb.DNSName, "."))
}

func (lb LoadBalancer) TypeShort() string {
	switch lb.LBType {
	case "application":
		return "ALB"
	case "network":
		return "NLB"
	case "gateway":
		return "GWLB"
	default:
		return strings.ToUpper(lb.LBType)
	}
}

// FetchAllLoadBalancers fetches ALB/NLB from all profiles × regions concurrently.
func FetchAllLoadBalancers(ctx context.Context, regions []string) ([]LoadBalancer, []error) {
	profiles, err := LoadProfiles()
	if err != nil {
		return nil, []error{err}
	}

	type target struct{ profile, region string }
	var targets []target
	for _, p := range profiles {
		for _, r := range regions {
			targets = append(targets, target{p, r})
		}
	}

	type result struct {
		lbs []LoadBalancer
		err error
	}
	ch := make(chan result, len(targets))
	var wg sync.WaitGroup

	for _, t := range targets {
		wg.Add(1)
		go func(p, r string) {
			defer wg.Done()
			lbs, err := fetchLoadBalancers(ctx, p, r)
			ch <- result{lbs, err}
		}(t.profile, t.region)
	}

	wg.Wait()
	close(ch)

	var all []LoadBalancer
	var errs []error
	for res := range ch {
		if res.err != nil {
			errs = append(errs, res.err)
			continue
		}
		all = append(all, res.lbs...)
	}
	return all, errs
}

func fetchLoadBalancers(ctx context.Context, profile, region string) ([]LoadBalancer, error) {
	client, err := NewProfileClient(ctx, profile, region)
	if err != nil {
		return nil, err
	}

	var lbs []LoadBalancer
	var nextMarker *string

	for {
		out, err := client.ELBv2.DescribeLoadBalancers(ctx, &elbv2.DescribeLoadBalancersInput{
			Marker: nextMarker,
		})
		if err != nil {
			return nil, err
		}

		for _, lb := range out.LoadBalancers {
			var azs []string
			for _, az := range lb.AvailabilityZones {
				azs = append(azs, aws.ToString(az.ZoneName))
			}
			var sgIDs []string
			sgIDs = append(sgIDs, lb.SecurityGroups...)

			state := "-"
			if lb.State != nil {
				state = string(lb.State.Code)
			}

			lbs = append(lbs, LoadBalancer{
				Profile:           profile,
				Region:            region,
				Name:              aws.ToString(lb.LoadBalancerName),
				DNSName:           aws.ToString(lb.DNSName),
				ARN:               aws.ToString(lb.LoadBalancerArn),
				LBType:            string(lb.Type),
				Scheme:            string(lb.Scheme),
				State:             state,
				VpcID:             aws.ToString(lb.VpcId),
				AvailabilityZones: azs,
				SecurityGroupIDs:  sgIDs,
			})
		}

		if out.NextMarker == nil {
			break
		}
		nextMarker = out.NextMarker
	}

	return lbs, nil
}
