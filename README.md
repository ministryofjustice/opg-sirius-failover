# opg-sirius-failover
Sirius Failover CLI: Managed by opg-org-infra &amp; Terraform

## Building Locally

```go
go build -mod vendor ./cmd/failover
```

## Install 

MacOS:

```bash
brew install ministryofjustice/opg/sirius-failover
```

## Usage

Usage: sirius-failover -env <ENVIRONMENT>
  -env string
    	The Environment to failover
