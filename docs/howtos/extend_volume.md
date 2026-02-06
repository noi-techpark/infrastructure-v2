# How to extend volumes

reference:
https://oneuptime.com/blog/post/2026-01-19-kubernetes-resize-persistent-volumes/view

## Storageclass expandable
make sure your storageclass has `allowVolumeExpansion: true`  
otherwise patch it:
```sh
kubectl patch storageclass gp3 -p '{"allowVolumeExpansion": true}';
```
## Resize (mongodb)
Patch the PersistentVolumeClaim to resize.  

Here we resize all three Mongodb volumes.
```sh
kubectl patch pvc datadir-mongodb-0 -n core -p '{"spec":{"resources":{"requests":{"storage":"320Gi"}}}}';
kubectl patch pvc datadir-mongodb-1 -n core -p '{"spec":{"resources":{"requests":{"storage":"320Gi"}}}}';
kubectl patch pvc datadir-mongodb-2 -n core -p '{"spec":{"resources":{"requests":{"storage":"320Gi"}}}}';
```