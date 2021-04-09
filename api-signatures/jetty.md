# Jetty

## Signatures

GET /

```
Problem accessing /. Reason:
```

GET /

```
Server: Jetty
```

GET /thisshouldntexist

```
Problem accessing /thisshouldntexist. Reason:
```

## Additional References

- https://censys.io/ipv4/report?field=443.https.tls.certificate.parsed.subject.organization.raw&max_buckets=1000&q=%22Problem+accessing+%2F.+Reason%3A%22
- https://censys.io/ipv4/report?field=443.https.tls.certificate.parsed.subject.organization.raw&max_buckets=1000&q=443.https.get.metadata.product%3A+Jetty