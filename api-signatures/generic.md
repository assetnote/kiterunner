# Generic

## Signatures

GET /api/v1/Test

```
"Invalid API key."
```

OR

```
"Invalid API token." 
```

OR

```
"Invalid key."
```

OR

GET /

```
Internal Server Error
```

GET /

```
"message":"Page not found"
```

GET /thisdoesntexist

```
"message":"Page not found"
```

GET /

```
404 Not Found
```

GET /api/aaa

```
HTTP/1.1 404 Not Found
Server: Microsoft-IIS/8.5
Content-Length: 0
```

OR

```
HTTP/1.1 404 Not Found
Server: nginx
Content-Length: 0
```

GET /api/test

```
"message":"not_found"
```

GET /

```
"message":"Not Found"
```

## Additional References

- https://censys.io/ipv4/report?field=443.https.tls.certificate.parsed.subject.organization.raw&max_buckets=1000&q=%22Internal+Server+Error%22
- https://censys.io/ipv4/report?field=443.https.tls.certificate.parsed.subject.organization.raw&max_buckets=1000&q=%22%5C%22message%5C%22%3A%5C%22Page+not+found%5C%22%22
- https://censys.io/ipv4/report?field=443.https.tls.certificate.parsed.subject.organization.raw&max_buckets=1000&q=443.https.get.body_sha256%3A+%227d04f7431bbfa41a04bcc7e6b98b9de0d919756c4c671c5785c99fff45f16402%22
- https://censys.io/ipv4?q=%22%5C%22message%5C%22%3A%5C%22not_found%5C%22%22
- https://censys.io/ipv4?q=%22%5C%22message%5C%22%3A%5C%22Not+Found%5C%22%22