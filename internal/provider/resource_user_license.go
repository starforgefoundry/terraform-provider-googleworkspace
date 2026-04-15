// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package googleworkspace

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"google.golang.org/api/licensing/v1"
)

func resourceUserLicense() *schema.Resource {
	return &schema.Resource{
		Description: "User License resource manages Google Workspace license assignments. A license " +
			"assignment represents a single user being assigned a specific product SKU. User License " +
			"resides under the `https://www.googleapis.com/auth/apps.licensing` client scope.",

		CreateContext: resourceUserLicenseCreate,
		ReadContext:   resourceUserLicenseRead,
		UpdateContext: resourceUserLicenseUpdate,
		DeleteContext: resourceUserLicenseDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceUserLicenseImport,
		},

		Schema: map[string]*schema.Schema{
			"product_id": {
				Description: "A product's unique identifier. For more information about products in " +
					"this version of the API, see " +
					"[Product and SKU IDs](https://developers.google.com/admin-sdk/licensing/v1/how-tos/products).",
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"sku_id": {
				Description: "A product SKU's unique identifier. For more information about available " +
					"SKUs in this version of the API, see " +
					"[Product and SKU IDs](https://developers.google.com/admin-sdk/licensing/v1/how-tos/products). " +
					"Changing `sku_id` will reassign the user to a different SKU within the same product.",
				Type:     schema.TypeString,
				Required: true,
			},
			"user_id": {
				Description: "The user's current primary email address. If the user's email address " +
					"changes, use the new email address in your API requests.",
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"etags": {
				Description: "ETag of the resource.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"product_name": {
				Description: "Display Name of the product.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"sku_name": {
				Description: "Display Name of the SKU of the product.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"self_link": {
				Description: "Link to this page.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

// userLicenseId builds the resource ID used to uniquely identify a license
// assignment. The Google API requires a product/sku/user tuple, so we encode
// that into a single colon-delimited string.
func userLicenseId(productId, skuId, userId string) string {
	return fmt.Sprintf("%s:%s:%s", productId, skuId, userId)
}

func resourceUserLicenseCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*apiClient)

	productId := d.Get("product_id").(string)
	skuId := d.Get("sku_id").(string)
	userId := d.Get("user_id").(string)

	log.Printf("[DEBUG] Creating User License product:%s, sku:%s, user:%s", productId, skuId, userId)

	licensingService, diags := client.NewLicensingService()
	if diags.HasError() {
		return diags
	}

	licenseAssignmentsService, diags := GetLicenseAssignmentsService(licensingService)
	if diags.HasError() {
		return diags
	}

	la, err := licenseAssignmentsService.Insert(productId, skuId, &licensing.LicenseAssignmentInsert{
		UserId: userId,
	}).Do()
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(userLicenseId(la.ProductId, la.SkuId, la.UserId))

	log.Printf("[DEBUG] Finished creating User License %q", d.Id())

	return resourceUserLicenseRead(ctx, d, meta)
}

func resourceUserLicenseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*apiClient)

	// Use the current state for the API call. sku_id may have been changed by the
	// last Update, but product_id and user_id are ForceNew so they're stable.
	productId := d.Get("product_id").(string)
	skuId := d.Get("sku_id").(string)
	userId := d.Get("user_id").(string)

	log.Printf("[DEBUG] Getting User License %q", d.Id())

	licensingService, diags := client.NewLicensingService()
	if diags.HasError() {
		return diags
	}

	licenseAssignmentsService, diags := GetLicenseAssignmentsService(licensingService)
	if diags.HasError() {
		return diags
	}

	la, err := licenseAssignmentsService.Get(productId, skuId, userId).Do()
	if err != nil {
		return handleNotFoundError(err, d, d.Id())
	}

	if la == nil {
		return diag.Errorf("No license assignment was returned for %s.", d.Id())
	}

	d.Set("product_id", la.ProductId)
	d.Set("sku_id", la.SkuId)
	d.Set("user_id", la.UserId)
	d.Set("etags", la.Etags)
	d.Set("product_name", la.ProductName)
	d.Set("sku_name", la.SkuName)
	d.Set("self_link", la.SelfLink)
	d.SetId(userLicenseId(la.ProductId, la.SkuId, la.UserId))

	log.Printf("[DEBUG] Finished getting User License %q", d.Id())

	return diags
}

func resourceUserLicenseUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*apiClient)

	productId := d.Get("product_id").(string)
	userId := d.Get("user_id").(string)

	// The only mutable field that triggers Update is sku_id: reassign the user
	// to a new SKU within the same product.
	oldSkuRaw, newSkuRaw := d.GetChange("sku_id")
	oldSkuId := oldSkuRaw.(string)
	newSkuId := newSkuRaw.(string)

	log.Printf("[DEBUG] Updating User License %q: moving from sku %s to %s", d.Id(), oldSkuId, newSkuId)

	licensingService, diags := client.NewLicensingService()
	if diags.HasError() {
		return diags
	}

	licenseAssignmentsService, diags := GetLicenseAssignmentsService(licensingService)
	if diags.HasError() {
		return diags
	}

	la, err := licenseAssignmentsService.Patch(productId, oldSkuId, userId, &licensing.LicenseAssignment{
		ProductId: productId,
		SkuId:     newSkuId,
		UserId:    userId,
	}).Do()
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(userLicenseId(la.ProductId, la.SkuId, la.UserId))

	log.Printf("[DEBUG] Finished updating User License %q", d.Id())

	return resourceUserLicenseRead(ctx, d, meta)
}

func resourceUserLicenseDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*apiClient)

	productId := d.Get("product_id").(string)
	skuId := d.Get("sku_id").(string)
	userId := d.Get("user_id").(string)

	log.Printf("[DEBUG] Deleting User License %q", d.Id())

	licensingService, diags := client.NewLicensingService()
	if diags.HasError() {
		return diags
	}

	licenseAssignmentsService, diags := GetLicenseAssignmentsService(licensingService)
	if diags.HasError() {
		return diags
	}

	if _, err := licenseAssignmentsService.Delete(productId, skuId, userId).Do(); err != nil {
		return handleNotFoundError(err, d, d.Id())
	}

	log.Printf("[DEBUG] Finished deleting User License %q", d.Id())

	return diags
}

func resourceUserLicenseImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), ":", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return nil, fmt.Errorf(
			"invalid user license import id %q: expected format product_id:sku_id:user_id",
			d.Id(),
		)
	}

	d.Set("product_id", parts[0])
	d.Set("sku_id", parts[1])
	d.Set("user_id", parts[2])
	d.SetId(userLicenseId(parts[0], parts[1], parts[2]))

	return []*schema.ResourceData{d}, nil
}
