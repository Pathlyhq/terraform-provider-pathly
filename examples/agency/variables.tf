variable "clients" {
  description = "Clients of the agency, one Space each."
  type = map(object({
    name          = string
    url           = string
    expect_text   = string
    public_status = bool
  }))
  default = {
    shop = {
      name          = "Example shop"
      url           = "https://shop.example.com"
      expect_text   = "Our products"
      public_status = true
    }
    blog = {
      name          = "Example blog"
      url           = "https://blog.example.com"
      expect_text   = "Latest posts"
      public_status = false
    }
  }
}
