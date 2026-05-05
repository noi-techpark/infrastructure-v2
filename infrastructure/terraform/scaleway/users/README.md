Needs first created via console:

- Application type IAM user
- API Key set to correct default projects
- Create a bucket in that same default project
- Add a bucket policy granting full access to resources <bucketname> and <bucketname>/* (the second one is not there by default) to the application user


```sh
source .env && terraform init
terraform apply
```