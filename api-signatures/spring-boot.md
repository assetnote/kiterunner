# Django Rest Framework

## Signature

GET /

When cURL'ing, the response you get is:

```
"message":"No message available","path":"/"}
```

When viewing in browser, the response is:

```
Whitelabel Error Page
```

GET /thisshouldnotexist

```
"No message available"
```

GET /profile

```
{
  "_links" : {
```

## Additional References

- https://censys.io/ipv4?q=%22No+message+available%22
- https://censys.io/ipv4?q=%22Whitelabel+Error+Page%22
- https://censys.io/ipv4/report?q=%22No+message+available%22&field=443.https.tls.certificate.parsed.subject.organization.raw&max_buckets=1000