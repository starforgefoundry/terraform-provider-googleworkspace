# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "googleworkspace_user" "dwight" {
  primary_email = "dwight.schrute@example.com"
  password      = "34819d7beeabb9260a5c854bc85b3e44"
  hash_function = "MD5"

  name {
    family_name = "Schrute"
    given_name  = "Dwight"
  }
}

# Assign a Google Workspace Business Standard license to the user.
# For a list of available product and SKU IDs see:
# https://developers.google.com/admin-sdk/licensing/v1/how-tos/products
resource "googleworkspace_user_license" "dwight-license" {
  product_id = "Google-Apps"
  sku_id     = "1010020028"
  user_id    = googleworkspace_user.dwight.primary_email
}
