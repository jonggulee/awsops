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
	SrcVpcID  string
	DstVpcID  string
	SrcCIDR   string
	DstCIDR   string
	Reachable bool
	Steps     []ConnectivityStep
}

// CheckConnectivity runs a 5-step connectivity check from srcVpcID to dstID (vpc-xxx or subnet-xxx).
func CheckConnectivity(
	srcVpcID, dstID string,
	attachments []TGWAttachment,
	associations []TGWAssociation,
	tgwRoutes []TGWRoute,
	vpcs []VPC,
	subnets []Subnet,
	routeTables []VPCRouteTable,
	accountToProfile map[string]string,
) ConnectivityResult {
	result := ConnectivityResult{SrcVpcID: srcVpcID}

	// Resolve dst VPC ID (subnet-xxx → vpc-xxx)
	dstVpcID := dstID
	if strings.HasPrefix(dstID, "subnet-") {
		for _, s := range subnets {
			if s.SubnetID == dstID {
				dstVpcID = s.VpcID
				break
			}
		}
	}
	result.DstVpcID = dstVpcID

	// Find CIDRs
	for _, v := range vpcs {
		if v.VpcID == srcVpcID {
			result.SrcCIDR = v.CidrBlock
		}
		if v.VpcID == dstVpcID {
			result.DstCIDR = v.CidrBlock
		}
	}

	// Find src/dst attachments (may be nil if cross-account and not visible)
	var srcAtt, dstAtt *TGWAttachment
	for i := range attachments {
		a := &attachments[i]
		if a.ResourceType == "vpc" {
			if a.ResourceID == srcVpcID {
				srcAtt = a
			}
			if a.ResourceID == dstVpcID {
				dstAtt = a
			}
		}
	}

	n := 1

	// ── Step 1: Source VPC TGW attachment active ──────────────────────────
	if srcAtt == nil {
		result.Steps = append(result.Steps, ConnectivityStep{
			Step:        n,
			Description: "Source VPC → TGW attachment",
			Status:      StatusFail,
			Detail:      fmt.Sprintf("%s has no visible TGW attachment", srcVpcID),
		})
		return result
	}
	step1Status := StatusOK
	if srcAtt.State != "available" {
		step1Status = StatusFail
	}
	result.Steps = append(result.Steps, ConnectivityStep{
		Step:        n,
		Description: "Source VPC → TGW attachment",
		Status:      step1Status,
		Detail:      fmt.Sprintf("%s  state: %s", srcAtt.AttachmentID, srcAtt.State),
	})
	n++
	if step1Status == StatusFail {
		return result
	}

	// ── Step 2: Source VPC route table → TGW for dst CIDR ────────────────
	srcRTs := rtForVPC(routeTables, srcVpcID)
	if len(srcRTs) == 0 {
		result.Steps = append(result.Steps, ConnectivityStep{
			Step:        n,
			Description: "Source VPC route table → TGW",
			Status:      StatusUnknown,
			Detail:      fmt.Sprintf("Route tables for %s not available (no profile access)", srcVpcID),
		})
	} else {
		r := findRouteViaTGW(srcRTs, result.DstCIDR)
		if r == nil {
			result.Steps = append(result.Steps, ConnectivityStep{
				Step:        n,
				Description: "Source VPC route table → TGW",
				Status:      StatusFail,
				Detail:      fmt.Sprintf("No route to %s via TGW in %s route tables", result.DstCIDR, srcVpcID),
			})
			return result
		}
		result.Steps = append(result.Steps, ConnectivityStep{
			Step:        n,
			Description: "Source VPC route table → TGW",
			Status:      StatusOK,
			Detail:      fmt.Sprintf("%s → %s  state: %s", r.DestinationCIDR, r.GatewayID, r.State),
		})
	}
	n++

	// ── Step 3: TGW route table → dst CIDR ───────────────────────────────
	assocRTID := ""
	for _, a := range associations {
		if a.AttachmentID == srcAtt.AttachmentID {
			assocRTID = a.RouteTableID
			break
		}
	}
	if assocRTID == "" {
		result.Steps = append(result.Steps, ConnectivityStep{
			Step:        n,
			Description: "TGW route table → destination",
			Status:      StatusUnknown,
			Detail:      "No route table association for source attachment",
		})
	} else {
		tr := findTGWRouteForCIDR(tgwRoutes, assocRTID, result.DstCIDR)
		if tr == nil {
			result.Steps = append(result.Steps, ConnectivityStep{
				Step:        n,
				Description: "TGW route table → destination",
				Status:      StatusFail,
				Detail:      fmt.Sprintf("No route to %s in route table %s", result.DstCIDR, assocRTID),
			})
			return result
		}
		nextHop := tr.ResourceID
		if tr.AttachmentID != "" && nextHop == "" {
			nextHop = tr.AttachmentID
		}
		result.Steps = append(result.Steps, ConnectivityStep{
			Step:        n,
			Description: "TGW route table → destination",
			Status:      StatusOK,
			Detail:      fmt.Sprintf("%s → %s  type: %s  state: %s", result.DstCIDR, nextHop, tr.RouteType, tr.State),
		})
	}
	n++

	// ── Step 4: Destination VPC TGW attachment active ─────────────────────
	if dstAtt == nil {
		result.Steps = append(result.Steps, ConnectivityStep{
			Step:        n,
			Description: "Destination VPC → TGW attachment",
			Status:      StatusUnknown,
			Detail:      fmt.Sprintf("%s attachment not visible (may be cross-account)", dstVpcID),
		})
	} else {
		step4Status := StatusOK
		if dstAtt.State != "available" {
			step4Status = StatusFail
		}
		result.Steps = append(result.Steps, ConnectivityStep{
			Step:        n,
			Description: "Destination VPC → TGW attachment",
			Status:      step4Status,
			Detail:      fmt.Sprintf("%s  state: %s", dstAtt.AttachmentID, dstAtt.State),
		})
		if step4Status == StatusFail {
			return result
		}
	}
	n++

	// ── Step 5: Destination VPC route table → TGW for src CIDR ───────────
	dstRTs := rtForVPC(routeTables, dstVpcID)
	if len(dstRTs) == 0 {
		result.Steps = append(result.Steps, ConnectivityStep{
			Step:        n,
			Description: "Destination VPC route table → TGW",
			Status:      StatusUnknown,
			Detail:      fmt.Sprintf("Route tables for %s not available (no profile access)", dstVpcID),
		})
	} else {
		r := findRouteViaTGW(dstRTs, result.SrcCIDR)
		if r == nil {
			result.Steps = append(result.Steps, ConnectivityStep{
				Step:        n,
				Description: "Destination VPC route table → TGW",
				Status:      StatusFail,
				Detail:      fmt.Sprintf("No route to %s via TGW in %s route tables", result.SrcCIDR, dstVpcID),
			})
			return result
		}
		result.Steps = append(result.Steps, ConnectivityStep{
			Step:        n,
			Description: "Destination VPC route table → TGW",
			Status:      StatusOK,
			Detail:      fmt.Sprintf("%s → %s  state: %s", r.DestinationCIDR, r.GatewayID, r.State),
		})
	}

	// Final verdict: reachable only if no step failed
	result.Reachable = true
	for _, s := range result.Steps {
		if s.Status == StatusFail {
			result.Reachable = false
			break
		}
	}
	return result
}

// ── helpers ───────────────────────────────────────────────────────────────

func rtForVPC(rts []VPCRouteTable, vpcID string) []VPCRouteTable {
	var out []VPCRouteTable
	for _, rt := range rts {
		if rt.VpcID == vpcID {
			out = append(out, rt)
		}
	}
	return out
}

// findRouteViaTGW returns the first active route in rts that:
//   - points to a TGW (gateway starts with "tgw-")
//   - covers destCIDR
func findRouteViaTGW(rts []VPCRouteTable, destCIDR string) *VPCRoute {
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
				return r
			}
		}
	}
	return nil
}

// findTGWRouteForCIDR returns the first active route in the given TGW route table
// whose destination covers destCIDR.
func findTGWRouteForCIDR(routes []TGWRoute, routeTableID, destCIDR string) *TGWRoute {
	for i := range routes {
		r := &routes[i]
		if r.RouteTableID != routeTableID {
			continue
		}
		if r.State != "active" {
			continue
		}
		if cidrCovers(r.DestinationCIDR, destCIDR) {
			return r
		}
	}
	return nil
}

// cidrCovers reports whether routeCIDR contains the network address of targetCIDR.
// e.g. "10.0.0.0/8" covers "10.1.0.0/16".
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
