# Nest (JavaScript)

## Signatures

GET /

```
{"statusCode":404,"message":"Cannot GET /","error":"Not Found"}
```

GET /thisshouldnotexist

```
{"statusCode":404,"message":"Cannot GET /thisshouldnotexist","error":"Not Found"}
```

## Additional References

- https://censys.io/ipv4/report?field=443.https.tls.certificate.parsed.subject.organization.raw&max_buckets=1000&q=%22%5C%22message%5C%22%3A%5C%22Cannot+GET+%2F%22
- https://docs.nestjs.com/