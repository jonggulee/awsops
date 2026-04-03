package aws

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
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
	Tags             map[string]string
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

	// 태그 조회 (DescribeTags는 ARN 기준, 최대 20개씩 배치 호출)
	for i := 0; i < len(lbs); i += 20 {
		end := i + 20
		if end > len(lbs) {
			end = len(lbs)
		}
		arns := make([]string, end-i)
		for j, lb := range lbs[i:end] {
			arns[j] = lb.ARN
		}
		tagOut, err := client.ELBv2.DescribeTags(ctx, &elbv2.DescribeTagsInput{
			ResourceArns: arns,
		})
		if err != nil {
			// 태그 조회 실패는 치명적이지 않으므로 무시
			continue
		}
		arnToTags := make(map[string]map[string]string, len(tagOut.TagDescriptions))
		for _, td := range tagOut.TagDescriptions {
			m := make(map[string]string, len(td.Tags))
			for _, t := range td.Tags {
				m[aws.ToString(t.Key)] = aws.ToString(t.Value)
			}
			arnToTags[aws.ToString(td.ResourceArn)] = m
		}
		for j := i; j < end; j++ {
			if tags, ok := arnToTags[lbs[j].ARN]; ok {
				lbs[j].Tags = tags
			}
		}
	}

	return lbs, nil
}

// --- Listener ---

// Listener represents an ELBv2 listener.
type Listener struct {
	ARN            string
	LBArn          string
	Profile        string
	Region         string
	Port           int32
	Protocol       string
	SSLPolicy      string
	CertARNs       []string
	DefaultActions []LBAction
}

// IsALB returns true for HTTP/HTTPS protocols (ALB).
func (l Listener) IsALB() bool {
	return l.Protocol == "HTTP" || l.Protocol == "HTTPS"
}

func (l Listener) Title() string {
	if l.Port > 0 {
		return l.Protocol + ":" + aws.ToString(&[]string{fmt.Sprintf("%d", l.Port)}[0])
	}
	return l.Protocol
}

// LBAction represents a listener or rule action.
type LBAction struct {
	Type           string // forward, redirect, fixed-response, authenticate-cognito
	TargetGroupARN string
	RedirectCode   string // 301, 302
	RedirectTarget string // "host:port/path"
}

// --- ListenerRule ---

// ListenerRule represents an ALB listener rule.
type ListenerRule struct {
	ARN         string
	ListenerARN string
	Priority    string
	IsDefault   bool
	Conditions  []RuleCondition
	Actions     []LBAction
}

// ForwardTGARNs returns ARNs of target groups in forward actions.
func (r ListenerRule) ForwardTGARNs() []string {
	var out []string
	for _, a := range r.Actions {
		if a.Type == "forward" && a.TargetGroupARN != "" {
			out = append(out, a.TargetGroupARN)
		}
	}
	return out
}

// RuleCondition represents a single condition in a listener rule.
type RuleCondition struct {
	Field  string
	Values []string
}

// --- TargetGroup ---

// TargetGroup represents an ELBv2 target group.
type TargetGroup struct {
	ARN         string
	Name        string
	Profile     string
	Region      string
	Protocol    string
	Port        int32
	TargetType  string
	VpcID       string
	HealthCheck TGHealthCheck
}

// TGHealthCheck holds health check configuration.
type TGHealthCheck struct {
	Protocol           string
	Path               string
	Port               string
	HealthyThreshold   int32
	UnhealthyThreshold int32
	TimeoutSeconds     int32
	IntervalSeconds    int32
}

// TargetEntry represents a target with its health state.
type TargetEntry struct {
	ID          string
	Port        int32
	AZ          string
	State       string
	Description string
}

// --- Fetch functions ---

// FetchListenersForLB fetches all listeners for a given load balancer.
func FetchListenersForLB(ctx context.Context, profile, region, lbARN string) ([]Listener, error) {
	client, err := NewProfileClient(ctx, profile, region)
	if err != nil {
		return nil, err
	}
	var listeners []Listener
	var marker *string
	for {
		out, err := client.ELBv2.DescribeListeners(ctx, &elbv2.DescribeListenersInput{
			LoadBalancerArn: aws.String(lbARN),
			Marker:          marker,
		})
		if err != nil {
			return nil, err
		}
		for _, l := range out.Listeners {
			listeners = append(listeners, toListener(profile, region, l))
		}
		if out.NextMarker == nil {
			break
		}
		marker = out.NextMarker
	}
	return listeners, nil
}

func toListener(profile, region string, l elbv2types.Listener) Listener {
	li := Listener{
		ARN:       aws.ToString(l.ListenerArn),
		LBArn:     aws.ToString(l.LoadBalancerArn),
		Profile:   profile,
		Region:    region,
		Protocol:  string(l.Protocol),
		SSLPolicy: aws.ToString(l.SslPolicy),
	}
	if l.Port != nil {
		li.Port = *l.Port
	}
	for _, cert := range l.Certificates {
		if cert.IsDefault == nil || !*cert.IsDefault {
			li.CertARNs = append(li.CertARNs, aws.ToString(cert.CertificateArn))
		}
	}
	for _, a := range l.DefaultActions {
		li.DefaultActions = append(li.DefaultActions, toAction(a))
	}
	return li
}

func toAction(a elbv2types.Action) LBAction {
	action := LBAction{Type: string(a.Type)}
	if a.TargetGroupArn != nil {
		action.TargetGroupARN = *a.TargetGroupArn
	}
	if a.RedirectConfig != nil {
		action.RedirectCode = string(a.RedirectConfig.StatusCode)
		host := aws.ToString(a.RedirectConfig.Host)
		port := aws.ToString(a.RedirectConfig.Port)
		path := aws.ToString(a.RedirectConfig.Path)
		action.RedirectTarget = host + ":" + port + path
	}
	// ForwardConfig (weighted) - use first TG
	if a.ForwardConfig != nil && len(a.ForwardConfig.TargetGroups) > 0 && action.TargetGroupARN == "" {
		action.TargetGroupARN = aws.ToString(a.ForwardConfig.TargetGroups[0].TargetGroupArn)
	}
	return action
}

// FetchRulesForListener fetches all rules for a given ALB listener.
func FetchRulesForListener(ctx context.Context, profile, region, listenerARN string) ([]ListenerRule, error) {
	client, err := NewProfileClient(ctx, profile, region)
	if err != nil {
		return nil, err
	}
	var rules []ListenerRule
	var marker *string
	for {
		out, err := client.ELBv2.DescribeRules(ctx, &elbv2.DescribeRulesInput{
			ListenerArn: aws.String(listenerARN),
			Marker:      marker,
		})
		if err != nil {
			return nil, err
		}
		for _, r := range out.Rules {
			rules = append(rules, toRule(r))
		}
		if out.NextMarker == nil {
			break
		}
		marker = out.NextMarker
	}
	return rules, nil
}

func toRule(r elbv2types.Rule) ListenerRule {
	rule := ListenerRule{
		ARN:       aws.ToString(r.RuleArn),
		Priority:  aws.ToString(r.Priority),
		IsDefault: aws.ToBool(r.IsDefault),
	}
	for _, c := range r.Conditions {
		rule.Conditions = append(rule.Conditions, toCondition(c))
	}
	for _, a := range r.Actions {
		rule.Actions = append(rule.Actions, toAction(a))
	}
	return rule
}

func toCondition(c elbv2types.RuleCondition) RuleCondition {
	field := aws.ToString(c.Field)
	var values []string
	switch field {
	case "host-header":
		if c.HostHeaderConfig != nil {
			values = c.HostHeaderConfig.Values
		}
	case "path-pattern":
		if c.PathPatternConfig != nil {
			values = c.PathPatternConfig.Values
		}
	case "http-header":
		if c.HttpHeaderConfig != nil {
			values = c.HttpHeaderConfig.Values
		}
	case "source-ip":
		if c.SourceIpConfig != nil {
			values = c.SourceIpConfig.Values
		}
	default:
		values = c.Values
	}
	return RuleCondition{Field: field, Values: values}
}

// FetchTargetGroupsForLB fetches all target groups for a given load balancer.
func FetchTargetGroupsForLB(ctx context.Context, profile, region, lbARN string) ([]TargetGroup, error) {
	client, err := NewProfileClient(ctx, profile, region)
	if err != nil {
		return nil, err
	}
	var tgs []TargetGroup
	var marker *string
	for {
		out, err := client.ELBv2.DescribeTargetGroups(ctx, &elbv2.DescribeTargetGroupsInput{
			LoadBalancerArn: aws.String(lbARN),
			Marker:          marker,
		})
		if err != nil {
			return nil, err
		}
		for _, tg := range out.TargetGroups {
			tgs = append(tgs, toTargetGroup(profile, region, tg))
		}
		if out.NextMarker == nil {
			break
		}
		marker = out.NextMarker
	}
	return tgs, nil
}

func toTargetGroup(profile, region string, tg elbv2types.TargetGroup) TargetGroup {
	hc := TGHealthCheck{
		Protocol:           string(tg.HealthCheckProtocol),
		Path:               aws.ToString(tg.HealthCheckPath),
		Port:               aws.ToString(tg.HealthCheckPort),
		HealthyThreshold:   aws.ToInt32(tg.HealthyThresholdCount),
		UnhealthyThreshold: aws.ToInt32(tg.UnhealthyThresholdCount),
		TimeoutSeconds:     aws.ToInt32(tg.HealthCheckTimeoutSeconds),
		IntervalSeconds:    aws.ToInt32(tg.HealthCheckIntervalSeconds),
	}
	t := TargetGroup{
		ARN:         aws.ToString(tg.TargetGroupArn),
		Name:        aws.ToString(tg.TargetGroupName),
		Profile:     profile,
		Region:      region,
		Protocol:    string(tg.Protocol),
		TargetType:  string(tg.TargetType),
		VpcID:       aws.ToString(tg.VpcId),
		HealthCheck: hc,
	}
	if tg.Port != nil {
		t.Port = *tg.Port
	}
	return t
}

// FetchTargetHealthForTG fetches all targets and their health for a target group.
func FetchTargetHealthForTG(ctx context.Context, profile, region, tgARN string) ([]TargetEntry, error) {
	client, err := NewProfileClient(ctx, profile, region)
	if err != nil {
		return nil, err
	}
	out, err := client.ELBv2.DescribeTargetHealth(ctx, &elbv2.DescribeTargetHealthInput{
		TargetGroupArn: aws.String(tgARN),
	})
	if err != nil {
		return nil, err
	}
	var entries []TargetEntry
	for _, thd := range out.TargetHealthDescriptions {
		e := TargetEntry{}
		if thd.Target != nil {
			e.ID = aws.ToString(thd.Target.Id)
			if thd.Target.Port != nil {
				e.Port = *thd.Target.Port
			}
			e.AZ = aws.ToString(thd.Target.AvailabilityZone)
		}
		if thd.TargetHealth != nil {
			e.State = string(thd.TargetHealth.State)
			e.Description = aws.ToString(thd.TargetHealth.Description)
		}
		entries = append(entries, e)
	}
	return entries, nil
}
