## Cluster Ingress, DNS and SSL
Ingress is managed via a single central Network Load Balancer.
All route53 DNS records are pointed at that NLB.
Behind the NLB the is a ingress-nginx service who actually distributes the calls within the cluster
SSL certificates are handled by certmanager, which coordinates with nginx to satisfy letsencrypt challenges

### DNS
Dns is managed by route53. Records are to be created/edited/deleted exclusively via terraform.

### Load Balancer
We are using a single NLB instead of one ALB per ingress to save on unnecessary costs, and to simplify and centralize DNS and SSL management.
Proxying is done by ingress-nginx https://cert-manager.io/docs/tutorials/acme/nginx-ingress

### SSL
certmanager is responsible for creating the SSL certificates.  
Generally we are following this setup: https://cert-manager.io/docs/tutorials/acme/nginx-ingress

Analogous to our old infrastructure with Caddy proxy, we should eventually migrate to a ACME DNS challenge model: https://cert-manager.io/docs/configuration/acme/dns01/route53/ 
