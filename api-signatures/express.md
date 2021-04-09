# Express

## Signature

GET /

```
<pre>Cannot GET /</pre>
```

GET /

```
X-Powered-By: Express
```

## Additional References

- https://censys.io/ipv4?q=%22%3Cpre%3ECannot+GET+%2F%3C%2Fpre%3E%22
- https://censys.io/ipv4/report?field=443.https.tls.certificate.parsed.subject.organization.raw&max_buckets=1000&q=%22%3Cpre%3ECannot+GET+%2F%3C%2Fpre%3E%22
- https://censys.io/ipv4?q=80.http.get.headers.x_powered_by%3AExpress+OR+443.https.get.headers.x_powered_by%3AExpress
- https://censys.io/ipv4/report?field=443.https.tls.certificate.parsed.subject.organization.raw&max_buckets=1000&q=80.http.get.headers.x_powered_by:Express%20OR%20443.https.get.headers.x_powered_by:Express