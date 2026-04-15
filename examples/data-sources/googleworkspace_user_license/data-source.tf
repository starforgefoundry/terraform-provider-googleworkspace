# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "googleworkspace_user_license" "dwight-license" {
  product_id = "Google-Apps"
  sku_id     = "1010020028"
  user_id    = "dwight.schrute@example.com"
}

output "sku_name" {
  value = data.googleworkspace_user_license.dwight-license.sku_name
}
