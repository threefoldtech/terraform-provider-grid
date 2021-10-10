module github.com/threefoldtech/terraform-provider-grid

go 1.16

require (
	github.com/google/uuid v1.3.0
	github.com/hashicorp/terraform-plugin-docs v0.4.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.6.1
	github.com/pkg/errors v0.9.1
	github.com/shurcooL/graphql v0.0.0-20200928012149-18c5c3165e3a
	github.com/threefoldtech/go-rmb v0.1.3
	github.com/threefoldtech/substrate-client v0.0.0-20211007134519-74137b8f68ec
	github.com/threefoldtech/zos v0.4.10-0.20210930143237-31899c4a55e2
	golang.org/x/net v0.0.0-20210813160813-60bc85c4be6d // indirect
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20210803171230-4253848d036c
)

replace github.com/threefoldtech/zos => ../zos
