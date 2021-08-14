module github.com/ashraffouda/grid-provider

go 1.15

require (
	github.com/emicklei/dot v0.16.0
	github.com/google/uuid v1.2.0
	github.com/hashicorp/terraform-plugin-docs v0.4.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.6.1
	github.com/pkg/errors v0.9.1
	github.com/threefoldtech/zos v0.4.10-0.20210804135636-7f25d677f88c
	github.com/urfave/cli v1.22.5
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20210803171230-4253848d036c
)

replace github.com/threefoldtech/zos => ../zos
