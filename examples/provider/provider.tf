terraform {
  required_providers {
    bugsink = {
      source = "nmehlei/bugsink"
    }
  }
}

provider "bugsink" {
  endpoint = "https://bugsink.example.com"
  # token is sensitive - prefer the BUGSINK_TOKEN environment variable over
  # hardcoding it here, especially in version-controlled configuration.
}
