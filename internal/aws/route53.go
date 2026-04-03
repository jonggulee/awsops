package aws

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
)

type Route53Record struct {
	Profile     string
	ZoneID      string
	ZoneName    string
	ZoneType    string // "Public" / "Private"
	Name        string
	Type        string // A, AAAA, CNAME, MX, NS, TXT, SOA, ...
	TTL         int64  // 0 = alias record
	Values      []string
	AliasTarget string // alias record일 때 대상 DNS 이름
}

// FirstValue returns the first value for table display.
func (r Route53Record) FirstValue() string {
	if r.AliasTarget != "" {
		return "→ " + r.AliasTarget
	}
	if len(r.Values) == 0 {
		return "-"
	}
	if len(r.Values) == 1 {
		return r.Values[0]
	}
	return r.Values[0] + " ..."
}

// TTLStr returns a human-readable TTL string.
func (r Route53Record) TTLStr() string {
	if r.AliasTarget != "" {
		return "alias"
	}
	if r.TTL == 0 {
		return "0"
	}
	return formatTTL(r.TTL)
}

func formatTTL(ttl int64) string {
	switch {
	case ttl%86400 == 0:
		return fmt.Sprintf("%dd", ttl/86400)
	case ttl%3600 == 0:
		return fmt.Sprintf("%dh", ttl/3600)
	case ttl%60 == 0:
		return fmt.Sprintf("%dm", ttl/60)
	default:
		return fmt.Sprintf("%ds", ttl)
	}
}

// FetchAllRoute53Records fetches Route 53 records from all profiles concurrently.
// Route 53 is global — region은 무관하므로 DefaultRegion을 사용한다.
func FetchAllRoute53Records(ctx context.Context) ([]Route53Record, []error) {
	profiles, err := LoadProfiles()
	if err != nil {
		return nil, []error{err}
	}

	type result struct {
		records []Route53Record
		err     error
	}
	ch := make(chan result, len(profiles))
	var wg sync.WaitGroup

	for _, p := range profiles {
		wg.Add(1)
		go func(profile string) {
			defer wg.Done()
			records, err := fetchRoute53Records(ctx, profile)
			ch <- result{records, err}
		}(p)
	}

	wg.Wait()
	close(ch)

	var all []Route53Record
	var errs []error
	for r := range ch {
		if r.err != nil {
			errs = append(errs, r.err)
			continue
		}
		all = append(all, r.records...)
	}
	return all, errs
}

type r53Zone struct {
	id       string
	name     string
	zoneType string
}

func fetchRoute53Records(ctx context.Context, profile string) ([]Route53Record, error) {
	client, err := NewProfileClient(ctx, profile, DefaultRegion)
	if err != nil {
		return nil, err
	}

	// hosted zone 목록 조회
	var zones []r53Zone
	var nextMarker *string
	for {
		out, err := client.Route53.ListHostedZones(ctx, &route53.ListHostedZonesInput{
			Marker: nextMarker,
		})
		if err != nil {
			return nil, err
		}
		for _, z := range out.HostedZones {
			zt := "Public"
			if z.Config != nil && z.Config.PrivateZone {
				zt = "Private"
			}
			zones = append(zones, r53Zone{
				id:       aws.ToString(z.Id),
				name:     strings.TrimSuffix(aws.ToString(z.Name), "."),
				zoneType: zt,
			})
		}
		if !out.IsTruncated {
			break
		}
		nextMarker = out.NextMarker
	}

	// 각 zone의 레코드를 병렬 조회
	type zoneResult struct {
		records []Route53Record
	}
	zch := make(chan zoneResult, len(zones))
	var zwg sync.WaitGroup

	for _, z := range zones {
		zwg.Add(1)
		go func(zone r53Zone) {
			defer zwg.Done()
			zch <- zoneResult{records: fetchZoneRecords(ctx, client, profile, zone)}
		}(z)
	}
	zwg.Wait()
	close(zch)

	var all []Route53Record
	for r := range zch {
		all = append(all, r.records...)
	}
	return all, nil
}

func fetchZoneRecords(ctx context.Context, client *ProfileClient, profile string, zone r53Zone) []Route53Record {
	var records []Route53Record
	var startName *string
	var startType *string

	for {
		input := &route53.ListResourceRecordSetsInput{
			HostedZoneId:    aws.String(zone.id),
			StartRecordName: startName,
		}
		// StartRecordType은 string이 아닌 타입이므로 별도 처리
		_ = startType

		out, err := client.Route53.ListResourceRecordSets(ctx, input)
		if err != nil {
			break
		}

		for _, rr := range out.ResourceRecordSets {
			name := strings.TrimSuffix(aws.ToString(rr.Name), ".")

			var values []string
			aliasTarget := ""
			if rr.AliasTarget != nil {
				aliasTarget = strings.TrimSuffix(aws.ToString(rr.AliasTarget.DNSName), ".")
			} else {
				for _, v := range rr.ResourceRecords {
					values = append(values, aws.ToString(v.Value))
				}
			}

			ttl := int64(0)
			if rr.TTL != nil {
				ttl = *rr.TTL
			}

			records = append(records, Route53Record{
				Profile:     profile,
				ZoneID:      zone.id,
				ZoneName:    zone.name,
				ZoneType:    zone.zoneType,
				Name:        name,
				Type:        string(rr.Type),
				TTL:         ttl,
				Values:      values,
				AliasTarget: aliasTarget,
			})
		}

		if !out.IsTruncated {
			break
		}
		startName = out.NextRecordName
	}

	return records
}
