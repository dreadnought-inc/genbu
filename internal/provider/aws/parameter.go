package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/dreadnought-inc/genbu/internal/config"
	"github.com/dreadnought-inc/genbu/internal/provider"
)

// SSMClient is the subset of the SSM API used by the parameter provider.
type SSMClient interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
	GetParametersByPath(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
}

// ParameterProvider fetches values from AWS SSM Parameter Store.
// It supports batch prefetching via GetParametersByPath to reduce API calls.
type ParameterProvider struct {
	client SSMClient
	cache  map[string]string // "region\x00paramName" -> value
}

// NewParameterProvider creates a ParameterProvider with the given SSM client.
func NewParameterProvider(client SSMClient) *ParameterProvider {
	return &ParameterProvider{client: client}
}

// Type returns "parameter".
func (p *ParameterProvider) Type() string {
	return "parameter"
}

// Prefetch batch-fetches parameters using GetParametersByPath.
// Keys are grouped by (region, parent path) and fetched in bulk.
// Results are cached for subsequent Resolve calls.
func (p *ParameterProvider) Prefetch(ctx context.Context, keys []provider.PrefetchKey) error {
	if p.cache == nil {
		p.cache = make(map[string]string)
	}

	type regionPath struct {
		region string
		path   string
	}
	groups := make(map[regionPath]bool)
	for _, k := range keys {
		parent := parentPath(k.Key)
		if parent == "" {
			continue
		}
		groups[regionPath{region: k.Region, path: parent}] = true
	}

	for rp := range groups {
		if err := p.fetchByPath(ctx, rp.region, rp.path); err != nil {
			return err
		}
	}
	return nil
}

// Resolve fetches a parameter value from SSM Parameter Store.
// If the value was prefetched, it is returned from cache.
func (p *ParameterProvider) Resolve(ctx context.Context, src *config.SourceConfig) (string, error) {
	key := src.EffectiveKey()
	if key == "" {
		return "", fmt.Errorf("aws parameter: key is required")
	}

	// Check cache first
	if p.cache != nil {
		if v, ok := p.cache[cacheKey(src.Region, key)]; ok {
			return v, nil
		}
	}

	// Cache miss: fall back to individual GetParameter
	withDecryption := true
	input := &ssm.GetParameterInput{
		Name:           &key,
		WithDecryption: &withDecryption,
	}

	var opts []func(*ssm.Options)
	if src.Region != "" {
		opts = append(opts, func(o *ssm.Options) {
			o.Region = src.Region
		})
	}

	output, err := p.client.GetParameter(ctx, input, opts...)
	if err != nil {
		return "", fmt.Errorf("aws parameter: getting %q: %w", key, err)
	}

	if output.Parameter == nil || output.Parameter.Value == nil {
		return "", fmt.Errorf("aws parameter: %q has no value", key)
	}

	return *output.Parameter.Value, nil
}

func (p *ParameterProvider) fetchByPath(ctx context.Context, region, path string) error {
	recursive := true
	withDecryption := true
	input := &ssm.GetParametersByPathInput{
		Path:           &path,
		Recursive:      &recursive,
		WithDecryption: &withDecryption,
	}

	var opts []func(*ssm.Options)
	if region != "" {
		opts = append(opts, func(o *ssm.Options) {
			o.Region = region
		})
	}

	for {
		output, err := p.client.GetParametersByPath(ctx, input, opts...)
		if err != nil {
			return fmt.Errorf("aws parameter: fetching path %q: %w", path, err)
		}

		for _, param := range output.Parameters {
			if param.Name != nil && param.Value != nil {
				p.cache[cacheKey(region, *param.Name)] = *param.Value
			}
		}

		if output.NextToken == nil {
			break
		}
		input.NextToken = output.NextToken
	}
	return nil
}

// cacheKey builds a composite key for the cache using null byte as separator.
func cacheKey(region, name string) string {
	return region + "\x00" + name
}

// parentPath returns the parent directory of a parameter path.
// "/a/b/c" -> "/a/b/", "/a" -> "/", "" -> ""
func parentPath(key string) string {
	if key == "" {
		return ""
	}
	idx := strings.LastIndex(key, "/")
	if idx < 0 {
		return ""
	}
	return key[:idx+1]
}
