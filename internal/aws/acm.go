package aws

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
)

type Certificate struct {
	Profile            string
	Region             string
	ARN                string
	DomainName         string
	SANs               []string // Subject Alternative Names
	Status             string
	Type               string   // AMAZON_ISSUED, IMPORTED
	KeyAlgorithm       string
	NotAfter           time.Time // expiry; zero if not issued yet
	NotBefore          time.Time
	InUseBy            []string // ARNs of resources using this cert
}

func (c Certificate) ExpiryStr() string {
	if c.NotAfter.IsZero() {
		return "-"
	}
	return c.NotAfter.In(time.Local).Format("2006-01-02")
}

// FetchAllCertificates fetches ACM certificates from all profiles × regions concurrently.
func FetchAllCertificates(ctx context.Context, regions []string) ([]Certificate, []error) {
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
		certs []Certificate
		err   error
	}
	ch := make(chan result, len(targets))
	var wg sync.WaitGroup

	for _, t := range targets {
		wg.Add(1)
		go func(p, r string) {
			defer wg.Done()
			certs, err := fetchCertificates(ctx, p, r)
			ch <- result{certs, err}
		}(t.profile, t.region)
	}

	wg.Wait()
	close(ch)

	var all []Certificate
	var errs []error
	for res := range ch {
		if res.err != nil {
			errs = append(errs, res.err)
			continue
		}
		all = append(all, res.certs...)
	}
	return all, errs
}

func fetchCertificates(ctx context.Context, profile, region string) ([]Certificate, error) {
	client, err := NewProfileClient(ctx, profile, region)
	if err != nil {
		return nil, err
	}

	// ListCertificates with full detail included
	var certs []Certificate
	var nextToken *string
	for {
		out, err := client.ACM.ListCertificates(ctx, &acm.ListCertificatesInput{
			NextToken: nextToken,
			Includes: &types.Filters{
				KeyTypes: []types.KeyAlgorithm{
					types.KeyAlgorithmRsa1024,
					types.KeyAlgorithmRsa2048,
					types.KeyAlgorithmRsa3072,
					types.KeyAlgorithmRsa4096,
					types.KeyAlgorithmEcPrime256v1,
					types.KeyAlgorithmEcSecp384r1,
					types.KeyAlgorithmEcSecp521r1,
				},
			},
		})
		if err != nil {
			return nil, err
		}

		for _, summary := range out.CertificateSummaryList {
			certs = append(certs, toCertificate(profile, region, summary))
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	// DescribeCertificate to get InUseBy and full SAN list
	for i := range certs {
		desc, err := client.ACM.DescribeCertificate(ctx, &acm.DescribeCertificateInput{
			CertificateArn: aws.String(certs[i].ARN),
		})
		if err != nil {
			continue // non-fatal
		}
		if d := desc.Certificate; d != nil {
			certs[i].SANs = d.SubjectAlternativeNames
			certs[i].InUseBy = d.InUseBy
		}
	}

	return certs, nil
}

func toCertificate(profile, region string, s types.CertificateSummary) Certificate {
	notAfter := time.Time{}
	if s.NotAfter != nil {
		notAfter = *s.NotAfter
	}
	notBefore := time.Time{}
	if s.NotBefore != nil {
		notBefore = *s.NotBefore
	}

	sans := s.SubjectAlternativeNameSummaries

	return Certificate{
		Profile:      profile,
		Region:       region,
		ARN:          aws.ToString(s.CertificateArn),
		DomainName:   aws.ToString(s.DomainName),
		SANs:         sans,
		Status:       string(s.Status),
		Type:         string(s.Type),
		KeyAlgorithm: string(s.KeyAlgorithm),
		NotAfter:     notAfter,
		NotBefore:    notBefore,
	}
}
