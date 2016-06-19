# BIND-zonefile-to-Terraform-Route53-resource-definition utility
Or BZFTTR53RDUtil, for "short". Lack of nice, pronouncible name aside, this small utility creates a [Terraform](https://terraform.io) file for Route53 resources from a BIND zonefile.

## Usage
`bzfttr53rdutil <domain-name> > route53-domain.tf`

The utility accepts a domain name as only parameter, and assumes that there is a file in the working directory named `<domain-name>.zone`. It writes the Terraform resources to stdout.

## Building
Just `go build`!

This project uses Godeps. See their [Github page](https://github.com/tools/godep) for more information.
