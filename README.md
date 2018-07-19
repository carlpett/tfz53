# BIND-zonefile-to-Terraform-Route53-resource-definition utility
Or BZFTTR53RDUtil, for "short". Lack of nice, pronouncible name aside, this small utility creates a [Terraform](https://terraform.io) file for Route53 resources from a BIND zonefile.

## Installation
Download the [latest release](https://github.com/carlpett/bzfttr53rdutil/releases/latest).

## Usage
`bzfttr53rdutil -domain <domain-name> [flags] > route53-domain.tf`

## Flags
| Name       | Description                                        | Default         |
|------------|----------------------------------------------------|-----------------|
| -domain    | Name of domain. Required.                          |                 |
| -zone-file | Path to zone file. Optional.                       | `<domain>.zone` |
| -exclude   | Record types to ignore, comma-separated. Optional. | `SOA,NS`        |


## Building
If you want to build from source, you will first need the Go tools. Instructions for installation are available from the [documentation](https://golang.org/doc/install#install).

Once that is done, run 

```bash
go get github.com/carlpett/bzfttr53rdutil
cd $GOPATH/src/github.com/carlpett/bzfttr53rdutil
go build
```

You should now have a finished binary.

This project uses `dep` to manage external dependencies. See the [Github repo](https://github.com/golang/dep) for more information.
