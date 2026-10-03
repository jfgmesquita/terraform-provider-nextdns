package provider

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jfgmesquita/terraform-provider-nextdns/internal/nextdns"
)

// emptyStringSet is the default for optional lists of IDs, such as tlds.
func emptyStringSet() types.Set {
	return types.SetValueMust(types.StringType, []attr.Value{})
}

// setToListItems converts a set of IDs from the configuration into the
// [{"id": ...}] list the API expects. It is sorted, so requests are the same
// on every run, and never nil, so an empty set clears the list.
func setToListItems(ctx context.Context, set types.Set) ([]nextdns.ListItem, diag.Diagnostics) {
	var ids []string
	diags := set.ElementsAs(ctx, &ids, false)
	if diags.HasError() {
		return nil, diags
	}
	sort.Strings(ids)

	items := make([]nextdns.ListItem, 0, len(ids))
	for _, id := range ids {
		items = append(items, nextdns.ListItem{ID: id})
	}
	return items, diags
}

// listItemsToSet converts a [{"id": ...}] list from the API into a set of IDs.
func listItemsToSet(ctx context.Context, items []nextdns.ListItem) (types.Set, diag.Diagnostics) {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return types.SetValueFrom(ctx, types.StringType, ids)
}
