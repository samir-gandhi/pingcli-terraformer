package exporter

import "github.com/samir-gandhi/pingcli-plugin-terraformer/internal/utils"

// NamedHCL is a thin alias to utils.NamedHCL to keep exporter tests stable.
type NamedHCL = utils.NamedHCL

// joinHCLBlocksSorted delegates to utils.JoinHCLBlocksSorted.
func joinHCLBlocksSorted(blocks []NamedHCL) string {
	return utils.JoinHCLBlocksSorted(blocks)
}

// sortAllResourceBlocks delegates to utils.SortAllResourceBlocks.
func sortAllResourceBlocks(hcl string) string {
	return utils.SortAllResourceBlocks(hcl)
}
