# tfz53 (previously knows as bzfttr53rdutil)
A conversion utility for creating [Terraform](https://terraform.io) or [Cloudformation](https://aws.amazon.com/cloudformation/) resource definitions for AWS Route53 from BIND zonefiles.

## Installation
Download the [latest release](https://github.com/carlpett/tfz53/releases/latest).

## Usage
`tfz53 -domain <domain-name> [flags] > route53-domain.tf`

`tfz53 -cloudformation -domain <domain-name> [flags] > route53-domain.cfn.yaml`


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
go get github.com/carlpett/tfz53
cd $GOPATH/src/github.com/carlpett/tfz53
go build
```

You should now have a finished binary.

This project uses `dep` to manage external dependencies. See the [Github repo](https://github.com/golang/dep) for more information.
