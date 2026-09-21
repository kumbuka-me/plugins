**GitHub · platform/runbooks · main · `deploy.sh`**

```bash
set -eu
kubectl apply -f deployment.yaml
kubectl rollout status deployment/api
```

**2** Apply the reviewed deployment manifest.  
**3** Wait until the rollout is healthy.
