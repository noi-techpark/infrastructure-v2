module "opendatahub_web_docs_test_s3" {
  source          = "./modules/s3-static-website"
  bucket_name     = "opendatahub-web-docs-test"
  index_document  = "index.html"
  error_document  = "error.html"
}