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
      source = "jfgmesquita/nextdns"
    }
  }
}

# Reads the API key from the NEXTDNS_API_KEY environment variable.
provider "nextdns" {}

resource "nextdns_profile" "office" {
  name = "Office"
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

The acceptance tests create and delete real profiles, so run them with the API key of a NextDNS account used only for testing, in `NEXTDNS_API_KEY`.

## Credits

Inspired by [amalucelli/terraform-provider-nextdns](https://github.com/amalucelli/terraform-provider-nextdns).
