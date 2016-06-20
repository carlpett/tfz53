# BIND-zonefile-to-Terraform-Route53-resource-definition utility
Or BZFTTR53RDUtil, for "short". Lack of nice, pronouncible name aside, this small utility creates a [Terraform](https://terraform.io) file for Route53 resources from a BIND zonefile.

## Usage
`bzfttr53rdutil -domain <domain-name> [flags] > route53-domain.tf`

## Flags
| Name       | Description                                        | Default         |
|------------|----------------------------------------------------|-----------------|
| -domain    | Name of domain. Required.                          |                 |
| -zone-file | Path to zone file. Optional.                       | `<domain>.zone` |
| -exclude   | Record types to ignore, comma-separated. Optional. | `SOA,NS`        |


## Building
Just `go build`!

This project uses Godeps. See their [Github page](https://github.com/tools/godep) for more information.
