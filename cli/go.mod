module cerberus.dev/cli

go 1.25.0

require (
	cerberus.dev/pkg v0.0.0
	cerberus.dev/sdk v0.0.0
	github.com/spf13/cobra v1.9.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	golang.org/x/crypto v0.50.0 // indirect
)

replace (
	cerberus.dev/pkg => ../pkg
	cerberus.dev/sdk => ../sdk
)
