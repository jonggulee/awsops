package aws

import (
	"fmt"
	"net"
	"strings"
)

type CheckStatus string

const (
	StatusOK      CheckStatus = "ok"
	StatusFail    CheckStatus = "fail"
	StatusUnknown CheckStatus = "unknown"
)

type ConnectivityStep struct {
	Step        int
	Description string
	Status      CheckStatus
	Detail      string
}

type ConnectivityResult struct {
	SrcSubnetID string
	DstSubnetID string
	SrcVpcID    string
	DstVpcID    string
	SrcVpcCIDR  string
	DstVpcCIDR  string
	Reachable   bool
	Steps       []ConnectivityStep
}

// CheckConnectivity runs a 5-step connectivity check from srcSubnetID to dstSubnetID.
//
// Step order (follows the actual packet path):
//  1. Source subnet route table → TGW for dst VPC CIDR
//  2. Source VPC TGW attachment active
//  3. TGW route table → dst VPC CIDR
//  4. Destination VPC TGW attachment active
//  5. Destination subnet route table → TGW for src VPC CIDR
func CheckConnectivity(
	srcSubnetID, dstSubnetID string,
	attachments []TGWAttachment,
	associations []TGWAssociation,
	tgwRoutes []TGWRoute,
	vpcs []VPC,
	subnets []Subnet,
	routeTables []VPCRouteTable,
	accountToProfile map[string]string,
) ConnectivityResult {
	result := ConnectivityResult{
		SrcSubnetID: srcSubnetID,
		DstSubnetID: dstSubnetID,
	}

	// Resolve subnets → VPC IDs
	var srcSubnet, dstSubnet *Subnet
	for i := range subnets {
		if subnets[i].SubnetID == srcSubnetID {
			srcSubnet = &subnets[i]
		}
		if subnets[i].SubnetID == dstSubnetID {
			dstSubnet = &subnets[i]
		}
	}
	if srcSubnet != nil {
		result.SrcVpcID = srcSubnet.VpcID
	}
	if dstSubnet != nil {
		result.DstVpcID = dstSubnet.VpcID
	}

	// Find VPC CIDRs
	for _, v := range vpcs {
		if v.VpcID == result.SrcVpcID {
			result.SrcVpcCIDR = v.CidrBlock
		}
		if v.VpcID == result.DstVpcID {
			result.DstVpcCIDR = v.CidrBlock
		}
	}

	// Find TGW attachments for src/dst VPCs
	var srcAtt, dstAtt *TGWAttachment
	for i := range attachments {
		a := &attachments[i]
		if a.ResourceType == "vpc" {
			if a.ResourceID == result.SrcVpcID {
				srcAtt = a
			}
			if a.ResourceID == result.DstVpcID {
				dstAtt = a
			}
		}
	}

	n := 1

	// ── Step 1: Source subnet route table → TGW for dst VPC CIDR ─────────
	srcRTs := rtForSubnet(routeTables, srcSubnetID, result.SrcVpcID)
	if len(srcRTs) == 0 {
		result.Steps = append(result.Steps, ConnectivityStep{
			Step: n, Description: "Source subnet route table → TGW",
			Status: StatusUnknown,
			Detail: fmt.Sprintf("Route table for %s not available (no profile access)", srcSubnetID),
		})
	} else {
		r, rtID, isExplicit := findRouteViaTGWWithInfo(srcRTs, result.DstVpcCIDR)
		if r == nil {
			result.Steps = append(result.Steps, ConnectivityStep{
				Step: n, Description: "Source subnet route table → TGW",
				Status: StatusFail,
				Detail: fmt.Sprintf("No route to %s via TGW in %s", result.DstVpcCIDR, rtID),
			})
			return result
		}
		tableNote := "main"
		if isExplicit {
			tableNote = "explicit"
		}
		result.Steps = append(result.Steps, ConnectivityStep{
			Step: n, Description: "Source subnet route table → TGW",
			Status: StatusOK,
			Detail: fmt.Sprintf("%s → %s  [%s: %s]", r.DestinationCIDR, r.GatewayID, tableNote, rtID),
		})
	}
	n++

	// ── Step 2: Source VPC TGW attachment active ──────────────────────────
	if srcAtt == nil {
		result.Steps = append(result.Steps, ConnectivityStep{
			Step: n, Description: "Source VPC → TGW attachment",
			Status: StatusFail,
			Detail: fmt.Sprintf("%s has no visible TGW attachment", result.SrcVpcID),
		})
		return result
	}
	s2 := StatusOK
	if srcAtt.State != "available" {
		s2 = StatusFail
	}
	result.Steps = append(result.Steps, ConnectivityStep{
		Step: n, Description: "Source VPC → TGW attachment",
		Status: s2,
		Detail: fmt.Sprintf("%s  state: %s", srcAtt.AttachmentID, srcAtt.State),
	})
	n++
	if s2 == StatusFail {
		return result
	}

	// ── Step 3: TGW route table → dst VPC CIDR ───────────────────────────
	assocRTID := ""
	for _, a := range associations {
		if a.AttachmentID == srcAtt.AttachmentID {
			assocRTID = a.RouteTableID
			break
		}
	}
	if assocRTID == "" {
		result.Steps = append(result.Steps, ConnectivityStep{
			Step: n, Description: "TGW route table → destination",
			Status: StatusUnknown,
			Detail: "No route table association for source attachment",
		})
	} else {
		tr := findTGWRouteForCIDR(tgwRoutes, assocRTID, result.DstVpcCIDR)
		if tr == nil {
			result.Steps = append(result.Steps, ConnectivityStep{
				Step: n, Description: "TGW route table → destination",
				Status: StatusFail,
				Detail: fmt.Sprintf("No route to %s in TGW route table %s", result.DstVpcCIDR, assocRTID),
			})
			return result
		}
		nextHop := tr.ResourceID
		if nextHop == "" {
			nextHop = tr.AttachmentID
		}
		result.Steps = append(result.Steps, ConnectivityStep{
			Step: n, Description: "TGW route table → destination",
			Status: StatusOK,
			Detail: fmt.Sprintf("%s → %s  type: %s  state: %s", result.DstVpcCIDR, nextHop, tr.RouteType, tr.State),
		})
	}
	n++

	// ── Step 4: Destination VPC TGW attachment active ─────────────────────
	if dstAtt == nil {
		result.Steps = append(result.Steps, ConnectivityStep{
			Step: n, Description: "Destination VPC → TGW attachment",
			Status: StatusUnknown,
			Detail: fmt.Sprintf("%s attachment not visible (may be cross-account)", result.DstVpcID),
		})
	} else {
		s4 := StatusOK
		if dstAtt.State != "available" {
			s4 = StatusFail
		}
		result.Steps = append(result.Steps, ConnectivityStep{
			Step: n, Description: "Destination VPC → TGW attachment",
			Status: s4,
			Detail: fmt.Sprintf("%s  state: %s", dstAtt.AttachmentID, dstAtt.State),
		})
		if s4 == StatusFail {
			return result
		}
	}
	n++

	// ── Step 5: Destination subnet route table → TGW for src VPC CIDR ────
	dstRTs := rtForSubnet(routeTables, dstSubnetID, result.DstVpcID)
	if len(dstRTs) == 0 {
		result.Steps = append(result.Steps, ConnectivityStep{
			Step: n, Description: "Destination subnet route table → TGW",
			Status: StatusUnknown,
			Detail: fmt.Sprintf("Route table for %s not available (no profile access)", dstSubnetID),
		})
	} else {
		r, rtID, isExplicit := findRouteViaTGWWithInfo(dstRTs, result.SrcVpcCIDR)
		if r == nil {
			result.Steps = append(result.Steps, ConnectivityStep{
				Step: n, Description: "Destination subnet route table → TGW",
				Status: StatusFail,
				Detail: fmt.Sprintf("No route to %s via TGW in %s", result.SrcVpcCIDR, rtID),
			})
			return result
		}
		tableNote := "main"
		if isExplicit {
			tableNote = "explicit"
		}
		result.Steps = append(result.Steps, ConnectivityStep{
			Step: n, Description: "Destination subnet route table → TGW",
			Status: StatusOK,
			Detail: fmt.Sprintf("%s → %s  [%s: %s]", r.DestinationCIDR, r.GatewayID, tableNote, rtID),
		})
	}

	result.Reachable = true
	for _, s := range result.Steps {
		if s.Status == StatusFail {
			result.Reachable = false
			break
		}
	}
	return result
}

// SubnetTGWRoute is a TGW route entry from a subnet's route table,
// enriched with route table context.
type SubnetTGWRoute struct {
	DestinationCIDR string
	GatewayID       string
	RouteTableID    string
	IsExplicit      bool // true = explicit subnet association, false = VPC main RT
}

// TGWRoutesForSubnet returns all active TGW routes from the route table
// associated with the given subnet (explicit association preferred, main RT fallback).
func TGWRoutesForSubnet(routeTables []VPCRouteTable, subnetID, vpcID string) []SubnetTGWRoute {
	rts := rtForSubnet(routeTables, subnetID, vpcID)
	var routes []SubnetTGWRoute
	for _, rt := range rts {
		isExplicit := len(rt.SubnetIDs) > 0
		for _, r := range rt.Routes {
			if r.State == "active" && strings.HasPrefix(r.GatewayID, "tgw-") {
				routes = append(routes, SubnetTGWRoute{
					DestinationCIDR: r.DestinationCIDR,
					GatewayID:       r.GatewayID,
					RouteTableID:    rt.RouteTableID,
					IsExplicit:      isExplicit,
				})
			}
		}
	}
	return routes
}

// CIDRCovers reports whether routeCIDR's network contains targetCIDR's network address.
func CIDRCovers(routeCIDR, targetCIDR string) bool {
	return cidrCovers(routeCIDR, targetCIDR)
}

// ── helpers ───────────────────────────────────────────────────────────────

// rtForSubnet returns the route table(s) for the given subnet.
// It prefers an explicit subnet association; falls back to the VPC main route table.
func rtForSubnet(rts []VPCRouteTable, subnetID, vpcID string) []VPCRouteTable {
	// Explicit association first
	for _, rt := range rts {
		for _, sid := range rt.SubnetIDs {
			if sid == subnetID {
				return []VPCRouteTable{rt}
			}
		}
	}
	// Fall back to VPC main route table
	for _, rt := range rts {
		if rt.VpcID == vpcID && rt.IsMain {
			return []VPCRouteTable{rt}
		}
	}
	return nil
}

// findRouteViaTGWWithInfo returns the matching route, its route table ID, and
// whether it was from an explicit subnet association (true) or main (false).
func findRouteViaTGWWithInfo(rts []VPCRouteTable, destCIDR string) (*VPCRoute, string, bool) {
	for ri := range rts {
		for i := range rts[ri].Routes {
			r := &rts[ri].Routes[i]
			if r.State != "active" {
				continue
			}
			if !strings.HasPrefix(r.GatewayID, "tgw-") {
				continue
			}
			if cidrCovers(r.DestinationCIDR, destCIDR) {
				isExplicit := len(rts[ri].SubnetIDs) > 0
				return r, rts[ri].RouteTableID, isExplicit
			}
		}
	}
	return nil, "", false
}

// findTGWRouteForCIDR returns the first active TGW route covering destCIDR in routeTableID.
func findTGWRouteForCIDR(routes []TGWRoute, routeTableID, destCIDR string) *TGWRoute {
	for i := range routes {
		r := &routes[i]
		if r.RouteTableID != routeTableID || r.State != "active" {
			continue
		}
		if cidrCovers(r.DestinationCIDR, destCIDR) {
			return r
		}
	}
	return nil
}

// cidrCovers reports whether routeCIDR's network contains targetCIDR's network address.
func cidrCovers(routeCIDR, targetCIDR string) bool {
	if targetCIDR == "" {
		return false
	}
	_, routeNet, err := net.ParseCIDR(routeCIDR)
	if err != nil {
		return routeCIDR == targetCIDR
	}
	targetIP, _, err := net.ParseCIDR(targetCIDR)
	if err != nil {
		ip := net.ParseIP(targetCIDR)
		if ip == nil {
			return false
		}
		return routeNet.Contains(ip)
	}
	return routeNet.Contains(targetIP)
}
