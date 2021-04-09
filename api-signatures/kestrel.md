# Kestrel

## Signature

GET /api/v2.0/pinga

```
HTTP/1.1 404 Not Found
Content-Length: 0
Server: Kestrel
```

GET /

```
HTTP/1.1 404 Not Found
Content-Length: 0
Server: Kestrel
```

## Additional References

- https://censys.io/ipv4/report?q=8080.http.get.metadata.product%3A+%22Kestrel%22+OR+443.https.get.metadata.product%3A+%22Kestrel%22+OR+80.http.get.metadata.product%3A+%22Kestrel%22+OR+8888.http.get.metadata.product%3A+%22Kestrel%22&field=443.https.tls.certificate.parsed.subject.organization.raw&max_buckets=1000