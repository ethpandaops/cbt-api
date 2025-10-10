package telemetry

import (
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// CustomSampler implements rate-based sampling with always-on-error strategy.
type CustomSampler struct {
	parentSampler      sdktrace.Sampler // Parent-based sampler (respects upstream decisions)
	rateSampler        sdktrace.Sampler // Rate-based sampler
	alwaysSampleErrors bool             // Always sample spans with errors
}

// NewCustomSampler creates a sampler that:
// 1. Respects parent trace decisions (if trace context exists)
// 2. Always samples errors (status >= 400) if enabled
// 3. Otherwise samples based on configured rate.
func NewCustomSampler(sampleRate float64, alwaysSampleErrors bool) sdktrace.Sampler {
	rateSampler := sdktrace.TraceIDRatioBased(sampleRate)
	parentSampler := sdktrace.ParentBased(rateSampler)

	return &CustomSampler{
		parentSampler:      parentSampler,
		rateSampler:        rateSampler,
		alwaysSampleErrors: alwaysSampleErrors,
	}
}

// ShouldSample implements the trace.Sampler interface.
func (s *CustomSampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	// First, check parent-based sampling (respects upstream trace decisions)
	parentResult := s.parentSampler.ShouldSample(p)

	// If parent-based sampler says to sample, do it
	if parentResult.Decision == sdktrace.RecordAndSample {
		return parentResult
	}

	// Check if this is an error span (status code >= 400)
	if s.alwaysSampleErrors {
		for _, attr := range p.Attributes {
			if attr.Key == HTTPStatusCodeKey {
				if statusCode := attr.Value.AsInt64(); statusCode >= 400 {
					return sdktrace.SamplingResult{
						Decision:   sdktrace.RecordAndSample,
						Attributes: p.Attributes,
					}
				}
			}
		}
	}

	// Otherwise, use rate-based sampling
	return s.rateSampler.ShouldSample(p)
}

// Description returns the sampler description.
func (s *CustomSampler) Description() string {
	return "CustomSampler{parent-based + rate-based + always-on-error}"
}
