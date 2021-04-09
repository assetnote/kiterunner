# Symfony

## Signatures

GET /

```
No route found for "GET /"
```

GET /app_dev.php

```
You are not allowed to access this file.
```

GET /doesntexist

```
Oops! An Error Occurred
``` 

GET /api/aaa

```
Oops! An Error Occurred
```

## Additional References

- https://censys.io/ipv4/report?field=443.https.tls.certificate.parsed.subject.organization.raw&max_buckets=1000&q=%22No+route+found+for+%22
- https://censys.io/ipv4/report?field=443.https.tls.certificate.parsed.subject.organization.raw&max_buckets=1000&q=%22Oops%21+An+Error+Occurred%22