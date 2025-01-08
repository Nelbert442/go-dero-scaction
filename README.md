## Go-DERO-SCAction <!-- omit in toc -->

Small go program to execute dero smart contract actions or installations from input data. You can leverage this to install local contract files as new smart contracts as well as interact with on-chain smart contracts via supplied parameters. Please read and understand the [disclaimer](#disclaimer) prior to usage.

## Table of Contents <!-- omit in toc -->
- [Contributing](#contributing)
- [Examples](#examples)
  - [Example 1 - UpdateSignature Gnomon SCID](#example-1---updatesignature-gnomon-scid)
- [Disclaimer](#disclaimer)

## Contributing
[Bug fixes](./.github/ISSUE_TEMPLATE/bug_report.md), [feature requests](./.github/ISSUE_TEMPLATE/feature_request.md) etc. can be submitted through normal issues on this repository. Feel free to follow the [Pull Request Template](./.github/pull_request_template.md) for any code merges.

## Examples

### Example 1 - UpdateSignature Gnomon SCID
```bash
go run main.go --daemon-rpc-address=127.0.0.1:40402 --wallet-rpc-address=127.0.0.1:40403 --operation=action --scid=df3a698af94afb46e7f6de40bbb628df2e10f29f79900928524d97f30a1928a2 --entrypoint=UpdateSignature --ringsize=2 --debug
```

## Disclaimer

This repository and its' contents are purely provided and maintained to assist with smart contract utilization on the [DERO Blockchain](https://github.com/deroproject/derohe). This code is not to be treated as failproof or to be responsible for any unexpected contract misuse or loss of funds. You should always leverage this in a testnet or simulator environment first ahead of utilizing in any sort of production scenarios and take full responsibility for the software utilization and understanding.