# Terraform Provider for NextDNS

> [!WARNING]
> This provider is not affiliated with or endorsed by NextDNS Inc.
>
> Developed with the assistance of Claude Code.

Manage [NextDNS](https://nextdns.io/) profiles with Terraform.

## Usage

The documentation of every resource and data source is on the [Terraform Registry](https://registry.terraform.io/providers/jfgmesquita/nextdns/latest/docs).

```terraform
terraform {
  required_providers {
    nextdns = {
      source  = "jfgmesquita/nextdns"
      version = "~> 0.1.0"
    }
  }
}

provider "nextdns" {
  api_key = var.nextdns_api_key
}
```

Create an API key on the [NextDNS account page](https://my.nextdns.io/account).

## Development

Requires [Go](https://go.dev/doc/install) and [Terraform](https://developer.hashicorp.com/terraform/install).

```shell
make build     # build the provider
make test      # run the unit tests
make generate  # regenerate docs/ from the code and examples/
make testacc   # run the acceptance tests
```

The acceptance tests create and delete real profiles. Run them with the API key of a dedicated test NextDNS account, set in `NEXTDNS_API_KEY`.

## Credits

Inspired by [amalucelli/terraform-provider-nextdns](https://github.com/amalucelli/terraform-provider-nextdns).
