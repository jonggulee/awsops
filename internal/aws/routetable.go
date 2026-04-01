package aws

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type VPCRoute struct {
	DestinationCIDR string
	GatewayID       string // tgw-xxx, igw-xxx, local, etc.
	State           string // active, blackhole
	Origin          string // CreateRouteTable, CreateRoute, EnableVgwRoutePropagation
}

type VPCRouteTable struct {
	Profile      string
	Region       string
	RouteTableID string
	VpcID        string
	SubnetIDs    []string // associated subnets (empty = main route table)
	IsMain       bool
	Routes       []VPCRoute
}

type rtResult struct {
	tables []VPCRouteTable
	err    error
}

// FetchAllRouteTables fetches VPC route tables from all profiles × regions.
func FetchAllRouteTables(ctx context.Context, regions []string) ([]VPCRouteTable, []error) {
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

	results := make(chan rtResult, len(targets))
	var wg sync.WaitGroup

	for _, t := range targets {
		wg.Add(1)
		go func(p, r string) {
			defer wg.Done()
			tables, err := fetchRouteTables(ctx, p, r)
			results <- rtResult{tables: tables, err: err}
		}(t.profile, t.region)
	}

	wg.Wait()
	close(results)

	var allTables []VPCRouteTable
	var errs []error
	for r := range results {
		if r.err != nil {
			errs = append(errs, r.err)
			continue
		}
		allTables = append(allTables, r.tables...)
	}
	return allTables, errs
}

func fetchRouteTables(ctx context.Context, profile, region string) ([]VPCRouteTable, error) {
	client, err := NewProfileClient(ctx, profile, region)
	if err != nil {
		return nil, err
	}

	out, err := client.EC2.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{})
	if err != nil {
		return nil, err
	}

	var tables []VPCRouteTable
	for _, rt := range out.RouteTables {
		tables = append(tables, toVPCRouteTable(profile, region, rt))
	}
	return tables, nil
}

func toVPCRouteTable(profile, region string, rt types.RouteTable) VPCRouteTable {
	table := VPCRouteTable{
		Profile:      profile,
		Region:       region,
		RouteTableID: aws.ToString(rt.RouteTableId),
		VpcID:        aws.ToString(rt.VpcId),
	}

	for _, assoc := range rt.Associations {
		if aws.ToBool(assoc.Main) {
			table.IsMain = true
		}
		if assoc.SubnetId != nil {
			table.SubnetIDs = append(table.SubnetIDs, aws.ToString(assoc.SubnetId))
		}
	}

	for _, r := range rt.Routes {
		dest := aws.ToString(r.DestinationCidrBlock)
		if dest == "" {
			dest = aws.ToString(r.DestinationIpv6CidrBlock)
		}
		gw := gatewayID(r)
		table.Routes = append(table.Routes, VPCRoute{
			DestinationCIDR: dest,
			GatewayID:       gw,
			State:           string(r.State),
			Origin:          string(r.Origin),
		})
	}
	return table
}

// gatewayID returns the most relevant gateway identifier for a route.
func gatewayID(r types.Route) string {
	switch {
	case r.TransitGatewayId != nil:
		return aws.ToString(r.TransitGatewayId)
	case r.GatewayId != nil:
		return aws.ToString(r.GatewayId)
	case r.NatGatewayId != nil:
		return aws.ToString(r.NatGatewayId)
	case r.VpcPeeringConnectionId != nil:
		return aws.ToString(r.VpcPeeringConnectionId)
	case r.InstanceId != nil:
		return aws.ToString(r.InstanceId)
	case r.NetworkInterfaceId != nil:
		return aws.ToString(r.NetworkInterfaceId)
	default:
		return ""
	}
}
