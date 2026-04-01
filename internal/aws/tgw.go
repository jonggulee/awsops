package aws

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type TransitGateway struct {
	Profile     string
	Region      string
	TgwID       string
	Name        string
	OwnerID     string // AWS Account ID
	State       string
	Description string
	Tags        map[string]string
}

type TGWAttachment struct {
	Profile         string
	Region          string
	AttachmentID    string
	TgwID           string
	TgwOwnerID      string // TGW 소유 계정 ID
	ResourceType    string // vpc, vpn, direct-connect-gateway, peering, ...
	ResourceID      string // vpc-xxx 등
	ResourceOwnerID string // 리소스 소유 계정 ID
	State           string
	Name            string
	Tags            map[string]string
}

// TGWRouteTable represents a Transit Gateway route table.
type TGWRouteTable struct {
	Profile      string
	Region       string
	RouteTableID string
	TgwID        string
	Name         string
	State        string
	Tags         map[string]string
}

// TGWRoute represents a single route entry in a TGW route table.
type TGWRoute struct {
	RouteTableID    string
	DestinationCIDR string
	State           string // active, blackhole
	RouteType       string // static, propagated
	AttachmentID    string
	ResourceType    string
	ResourceID      string
}

// TGWAssociation represents the association between an attachment and a route table.
type TGWAssociation struct {
	RouteTableID string
	AttachmentID string
	ResourceType string
	ResourceID   string
	State        string
}

type tgwResult struct {
	gateways     []TransitGateway
	attachments  []TGWAttachment
	routeTables  []TGWRouteTable
	routes       []TGWRoute
	associations []TGWAssociation
	err          error
}

// FetchAllTGWs fetches Transit Gateways, Attachments, Route Tables, Routes, and Associations
// from all profiles × regions concurrently.
func FetchAllTGWs(ctx context.Context, regions []string) (
	[]TransitGateway, []TGWAttachment, []TGWRouteTable, []TGWRoute, []TGWAssociation, []error,
) {
	profiles, err := LoadProfiles()
	if err != nil {
		return nil, nil, nil, nil, nil, []error{err}
	}

	type target struct{ profile, region string }
	var targets []target
	for _, p := range profiles {
		for _, r := range regions {
			targets = append(targets, target{p, r})
		}
	}

	results := make(chan tgwResult, len(targets))
	var wg sync.WaitGroup

	for _, t := range targets {
		wg.Add(1)
		go func(p, r string) {
			defer wg.Done()
			gws, atts, rts, routes, assocs, err := fetchTGWs(ctx, p, r)
			results <- tgwResult{
				gateways: gws, attachments: atts,
				routeTables: rts, routes: routes, associations: assocs,
				err: err,
			}
		}(t.profile, t.region)
	}

	wg.Wait()
	close(results)

	var allGWs []TransitGateway
	var allAtts []TGWAttachment
	var allRTs []TGWRouteTable
	var allRoutes []TGWRoute
	var allAssocs []TGWAssociation
	var errs []error

	// Deduplicate route tables and routes by ID (multiple profiles may see the same TGW owner's data).
	seenRT := map[string]struct{}{}
	seenRoute := map[string]struct{}{}
	seenAssoc := map[string]struct{}{}

	for r := range results {
		if r.err != nil {
			errs = append(errs, r.err)
			continue
		}
		allGWs = append(allGWs, r.gateways...)
		allAtts = append(allAtts, r.attachments...)
		for _, rt := range r.routeTables {
			if _, seen := seenRT[rt.RouteTableID]; !seen {
				seenRT[rt.RouteTableID] = struct{}{}
				allRTs = append(allRTs, rt)
			}
		}
		for _, route := range r.routes {
			key := route.RouteTableID + "|" + route.DestinationCIDR + "|" + route.AttachmentID
			if _, seen := seenRoute[key]; !seen {
				seenRoute[key] = struct{}{}
				allRoutes = append(allRoutes, route)
			}
		}
		for _, assoc := range r.associations {
			key := assoc.RouteTableID + "|" + assoc.AttachmentID
			if _, seen := seenAssoc[key]; !seen {
				seenAssoc[key] = struct{}{}
				allAssocs = append(allAssocs, assoc)
			}
		}
	}
	return allGWs, allAtts, allRTs, allRoutes, allAssocs, errs
}

func fetchTGWs(ctx context.Context, profile, region string) (
	[]TransitGateway, []TGWAttachment, []TGWRouteTable, []TGWRoute, []TGWAssociation, error,
) {
	client, err := NewProfileClient(ctx, profile, region)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	tgwOut, err := client.EC2.DescribeTransitGateways(ctx, &ec2.DescribeTransitGatewaysInput{})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	attOut, err := client.EC2.DescribeTransitGatewayAttachments(ctx, &ec2.DescribeTransitGatewayAttachmentsInput{})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	var gws []TransitGateway
	for _, g := range tgwOut.TransitGateways {
		gws = append(gws, toTGW(profile, region, g))
	}

	var atts []TGWAttachment
	for _, a := range attOut.TransitGatewayAttachments {
		atts = append(atts, toTGWAttachment(profile, region, a))
	}

	// Route tables are only visible to the TGW owner account.
	// Non-owner profiles will get an empty list — that's fine.
	rtOut, err := client.EC2.DescribeTransitGatewayRouteTables(ctx, &ec2.DescribeTransitGatewayRouteTablesInput{})
	if err != nil {
		// Not fatal — this profile may not own any TGW.
		return gws, atts, nil, nil, nil, nil
	}

	var rts []TGWRouteTable
	var allRoutes []TGWRoute
	var allAssocs []TGWAssociation

	for _, rt := range rtOut.TransitGatewayRouteTables {
		rtID := aws.ToString(rt.TransitGatewayRouteTableId)
		rts = append(rts, toTGWRouteTable(profile, region, rt))

		// Fetch routes for this route table.
		routeOut, err := client.EC2.SearchTransitGatewayRoutes(ctx, &ec2.SearchTransitGatewayRoutesInput{
			TransitGatewayRouteTableId: aws.String(rtID),
			Filters: []types.Filter{
				{Name: aws.String("state"), Values: []string{"active", "blackhole"}},
			},
		})
		if err == nil {
			for _, r := range routeOut.Routes {
				allRoutes = append(allRoutes, toTGWRoute(rtID, r))
			}
		}

		// Fetch associations for this route table.
		assocOut, err := client.EC2.GetTransitGatewayRouteTableAssociations(ctx,
			&ec2.GetTransitGatewayRouteTableAssociationsInput{
				TransitGatewayRouteTableId: aws.String(rtID),
			})
		if err == nil {
			for _, a := range assocOut.Associations {
				allAssocs = append(allAssocs, TGWAssociation{
					RouteTableID: rtID,
					AttachmentID: aws.ToString(a.TransitGatewayAttachmentId),
					ResourceType: string(a.ResourceType),
					ResourceID:   aws.ToString(a.ResourceId),
					State:        string(a.State),
				})
			}
		}
	}

	return gws, atts, rts, allRoutes, allAssocs, nil
}

func toTGW(profile, region string, g types.TransitGateway) TransitGateway {
	tags := make(map[string]string)
	name := ""
	for _, t := range g.Tags {
		k, v := aws.ToString(t.Key), aws.ToString(t.Value)
		tags[k] = v
		if k == "Name" {
			name = v
		}
	}
	return TransitGateway{
		Profile:     profile,
		Region:      region,
		TgwID:       aws.ToString(g.TransitGatewayId),
		Name:        name,
		OwnerID:     aws.ToString(g.OwnerId),
		State:       string(g.State),
		Description: aws.ToString(g.Description),
		Tags:        tags,
	}
}

func toTGWAttachment(profile, region string, a types.TransitGatewayAttachment) TGWAttachment {
	tags := make(map[string]string)
	name := ""
	for _, t := range a.Tags {
		k, v := aws.ToString(t.Key), aws.ToString(t.Value)
		tags[k] = v
		if k == "Name" {
			name = v
		}
	}
	return TGWAttachment{
		Profile:         profile,
		Region:          region,
		AttachmentID:    aws.ToString(a.TransitGatewayAttachmentId),
		TgwID:           aws.ToString(a.TransitGatewayId),
		TgwOwnerID:      aws.ToString(a.TransitGatewayOwnerId),
		ResourceType:    string(a.ResourceType),
		ResourceID:      aws.ToString(a.ResourceId),
		ResourceOwnerID: aws.ToString(a.ResourceOwnerId),
		State:           string(a.State),
		Name:            name,
		Tags:            tags,
	}
}

func toTGWRouteTable(profile, region string, rt types.TransitGatewayRouteTable) TGWRouteTable {
	tags := make(map[string]string)
	name := ""
	for _, t := range rt.Tags {
		k, v := aws.ToString(t.Key), aws.ToString(t.Value)
		tags[k] = v
		if k == "Name" {
			name = v
		}
	}
	return TGWRouteTable{
		Profile:      profile,
		Region:       region,
		RouteTableID: aws.ToString(rt.TransitGatewayRouteTableId),
		TgwID:        aws.ToString(rt.TransitGatewayId),
		Name:         name,
		State:        string(rt.State),
		Tags:         tags,
	}
}

func toTGWRoute(routeTableID string, r types.TransitGatewayRoute) TGWRoute {
	route := TGWRoute{
		RouteTableID:    routeTableID,
		DestinationCIDR: aws.ToString(r.DestinationCidrBlock),
		State:           string(r.State),
		RouteType:       string(r.Type),
	}
	// A route may have multiple attachments; use the first active one.
	for _, att := range r.TransitGatewayAttachments {
		route.AttachmentID = aws.ToString(att.TransitGatewayAttachmentId)
		route.ResourceType = string(att.ResourceType)
		route.ResourceID = aws.ToString(att.ResourceId)
		break
	}
	return route
}
