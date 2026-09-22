# Model Pricing Data

This directory contains the local fallback catalog used when the remote pricing
catalog cannot be downloaded. It is a curated snapshot: prices here have been
audited against sub2api's own billing policy, and two golden tests in
`internal/service/pricing_service_test.go` pin specific values from it
(`TestDefaultPricingIncludesGemini36FlashRates`,
`TestDefaultPricingUsesCurrentCodexAutoReviewBaseRates`). Do not refresh it
blindly from an upstream catalog — see "Relationship to the remote catalog".

## Source

The primary catalog is served by our own repository:

- Remote catalog (`pricing.remote_url` default):
  https://raw.githubusercontent.com/Mxucc/model-price-repo/main/model_prices_and_context_window.json
- Remote hash (`pricing.hash_url` default):
  https://raw.githubusercontent.com/Mxucc/model-price-repo/main/model_prices_and_context_window.sha256
- Repository: https://github.com/Mxucc/model-price-repo

That repository runs a GitHub Actions workflow every 10 minutes which:

1. downloads the upstream LiteLLM file
   (https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json),
2. filters it by the `prefix_filters` in `config.json`,
3. merges new models additively — `update_existing: false` means a model that is
   already published keeps its current prices and only absorbs newly added fields,
4. applies `aliases` and always injects `custom_models`.

`custom_models` in `config.json` is the intended place to hand-maintain our own
price list: those entries are written last and therefore win over upstream values.

## Relationship to the remote catalog

The merge order in `pricing_service.go` is: remote catalog → **fallback (this
directory, only for models the catalog does not contain)** → `override_file`.
Consequences:

- A model present in the remote catalog never picks up prices from this file.
  To reprice it, edit `custom_models` in model-price-repo, or use the
  `pricing.override_file` layer documented in `deploy/config.example.yaml`.
- This file only matters when the remote download fails, or for models the
  remote catalog does not serve.
- Its prices intentionally differ from the raw upstream LiteLLM numbers in a few
  places (long-context ladders, Codex internal models, cache-write tiers). Do not
  regenerate it from upstream without re-checking the golden tests above.

## Purpose

This local copy serves as a fallback when the remote file cannot be downloaded due to:

- Network restrictions
- Firewall rules
- DNS resolution issues
- GitHub being blocked in certain regions
- Docker container network limitations

The `pricingService` will:

1. First attempt to download the latest version from the remote catalog URL
2. If the download fails, use this local copy as fallback
3. Log a warning when using the fallback file

## Manual Update

Only refresh this file from a source whose prices you have reviewed:

```bash
curl -s https://raw.githubusercontent.com/Mxucc/model-price-repo/main/model_prices_and_context_window.json -o model_prices_and_context_window.json
cd ../service && go test ./... -run 'TestDefaultPricing'
```

## File Format

The file contains JSON data with model pricing information including:

- Model names and identifiers
- Input/output token costs
- Context window sizes
- Model capabilities
