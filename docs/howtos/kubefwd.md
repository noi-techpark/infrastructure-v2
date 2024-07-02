# Using kubefwd to forward kubernetes service for local development

[Kubefwd](https://github.com/txn2/kubefwd) is a tool to forwards kubernetes services and mirror the DNS records via host file.

# On Linux
Install by downloading either the packages or binary (it's just a single one) from their github release section

kubefwd must be run as root, so you will have to login your AWS cli for root:
```sh
sudo aws configure
```
Now you can use your existing (non-root) kubectl config to connect:
```sh
# Forward core and collector namespaces.
# refer to kubefwd documentation for syntax. 
# Important is the -c ~/.kube/config bit, which tells the tool to use your non-root config
sudo -E kubefwd svc -n core -n collector -c ~/.kube/config
```

You should now be able to just use the service URLs like inside the cluster.  

# Mongodb
The DNS service is only spoofed, not mirrored, so if you are using more advanced DNS lookups like mongodb "+srv" type URL, it won't work.  
Modify the servicebind URL by omitting the +srv part and it should just work (maybe not if you need to write as well)