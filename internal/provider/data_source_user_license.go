// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package googleworkspace

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceUserLicense() *schema.Resource {
	dsSchema := datasourceSchemaFromResourceSchema(resourceUserLicense().Schema)
	addRequiredFieldsToSchema(dsSchema, "product_id", "sku_id", "user_id")

	return &schema.Resource{
		Description: "User License data source in the Terraform Googleworkspace provider. User License " +
			"resides under the `https://www.googleapis.com/auth/apps.licensing` client scope.",

		ReadContext: dataSourceUserLicenseRead,

		Schema: dsSchema,
	}
}

func dataSourceUserLicenseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	productId := d.Get("product_id").(string)
	skuId := d.Get("sku_id").(string)
	userId := d.Get("user_id").(string)

	d.SetId(userLicenseId(productId, skuId, userId))

	return resourceUserLicenseRead(ctx, d, meta)
}
