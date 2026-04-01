package aws

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type accountResult struct {
	profile   string
	accountID string
	err       error
}

// FetchAccountIDs resolves AWS account IDs for all profiles via STS GetCallerIdentity.
// Returns a map of profile → accountID and a reverse map of accountID → profile.
func FetchAccountIDs(ctx context.Context) (profileToAccount map[string]string, accountToProfile map[string]string, errs []error) {
	profiles, err := LoadProfiles()
	if err != nil {
		return nil, nil, []error{err}
	}

	results := make(chan accountResult, len(profiles))
	var wg sync.WaitGroup

	for _, p := range profiles {
		wg.Add(1)
		go func(profile string) {
			defer wg.Done()
			client, err := NewProfileClient(ctx, profile, DefaultRegion)
			if err != nil {
				results <- accountResult{profile: profile, err: err}
				return
			}
			out, err := client.STS.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
			if err != nil {
				results <- accountResult{profile: profile, err: err}
				return
			}
			accountID := ""
			if out.Account != nil {
				accountID = *out.Account
			}
			results <- accountResult{profile: profile, accountID: accountID}
		}(p)
	}

	wg.Wait()
	close(results)

	profileToAccount = make(map[string]string)
	accountToProfile = make(map[string]string)
	for r := range results {
		if r.err != nil {
			errs = append(errs, r.err)
			continue
		}
		profileToAccount[r.profile] = r.accountID
		accountToProfile[r.accountID] = r.profile
	}
	return profileToAccount, accountToProfile, errs
}
